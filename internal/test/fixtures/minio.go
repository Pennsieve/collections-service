package fixtures

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sync"
	"time"
)

type MinIOOption func(minIO *MinIO)

type MinIO struct {
	s3Client     *s3.Client
	knownBuckets map[string]bool
}

func NewMinIOWithDefaultClient(ctx context.Context, t require.TestingT) *MinIO {
	minIO := &MinIO{
		s3Client:     test.DefaultMinIOS3Client(ctx, t),
		knownBuckets: make(map[string]bool),
	}
	return minIO
}

func (m *MinIO) CreatePublishBucket(ctx context.Context, t require.TestingT) string {
	randomName := uuid.NewString()
	bucketName := aws.String(randomName)
	createInput := &s3.CreateBucketInput{Bucket: bucketName}
	_, err := m.s3Client.CreateBucket(ctx, createInput)
	if err != nil {
		fmt.Printf("%v: %T\n", err, err)
		unwrapped := errors.Unwrap(err)
		for unwrapped != nil {
			fmt.Printf("%v: %T\n", unwrapped, unwrapped)
			unwrapped = errors.Unwrap(unwrapped)
		}
		fmt.Printf("%+v\n", m.s3Client.Options())
		fmt.Printf("BaseEndpoint: %s\n", *m.s3Client.Options().BaseEndpoint)
		fmt.Printf("Credentials: %+v: %T\n", m.s3Client.Options().Credentials, m.s3Client.Options().Credentials)
		fmt.Printf("EndpointResolver2: %+v: %T\n", m.s3Client.Options().EndpointResolverV2, m.s3Client.Options().EndpointResolverV2)
		fmt.Printf("Region: %s\n", m.s3Client.Options().Region)

	}
	require.NoError(t, err)

	require.NoError(t, s3.NewBucketExistsWaiter(m.s3Client).Wait(ctx, &s3.HeadBucketInput{Bucket: bucketName}, time.Minute), "publish bucket %s not created in time", randomName)

	m.knownBuckets[randomName] = true

	versioningInput := &s3.PutBucketVersioningInput{
		Bucket: bucketName,
		VersioningConfiguration: &types.VersioningConfiguration{
			Status: types.BucketVersioningStatusEnabled,
		},
	}
	_, err = m.s3Client.PutBucketVersioning(ctx, versioningInput)
	require.NoError(t, err, "error enabling versioning: bucket: %s, error: %v", randomName, err)
	return randomName
}

func (m *MinIO) RequireObjectExists(ctx context.Context, t require.TestingT, bucket, key string) *s3.HeadObjectOutput {
	headIn := &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	headOut, err := m.s3Client.HeadObject(ctx, headIn)
	require.NoError(t, err, "HEAD object returned an error for expected bucket: %s, key: %s", bucket, key)
	return headOut
}

// ListObjectVersions returns slices of all DeleteMarkers and Versions found in the given bucket under the given prefix if any.
// It takes care of pagination, so the returned slices are the entire listings. It is assumed that in a test situation that these
// will be small enough to hold in memory without any issues.
func (m *MinIO) ListObjectVersions(ctx context.Context, t require.TestingT, bucket string, prefix *string) struct {
	DeleteMarkers []types.DeleteMarkerEntry
	Versions      []types.ObjectVersion
} {
	var deleteMarkers []types.DeleteMarkerEntry
	var versions []types.ObjectVersion
	listInput := s3.ListObjectVersionsInput{Bucket: aws.String(bucket), Prefix: prefix}
	var isTruncated bool
	for makeRequest := true; makeRequest; makeRequest = isTruncated {
		listOutput, err := m.s3Client.ListObjectVersions(ctx, &listInput)
		require.NoError(t, err, "error listing test objects: bucket: %s, error: %v", bucket, err)

		deleteMarkers = append(deleteMarkers, listOutput.DeleteMarkers...)
		versions = append(versions, listOutput.Versions...)
		isTruncated = aws.ToBool(listOutput.IsTruncated)
		if isTruncated {
			listInput.KeyMarker = listOutput.NextKeyMarker
			listInput.VersionIdMarker = listOutput.NextVersionIdMarker
		}
	}
	return struct {
		DeleteMarkers []types.DeleteMarkerEntry
		Versions      []types.ObjectVersion
	}{
		deleteMarkers,
		versions,
	}
}

func awsErrorToString(bucket string, error types.Error) string {
	return fmt.Sprintf("AWS error: code: %s, message: %s, S3 Object: (%s, %s, %s)",
		aws.ToString(error.Code),
		aws.ToString(error.Message),
		bucket,
		aws.ToString(error.Key),
		aws.ToString(error.VersionId))
}

func (m *MinIO) CleanUp(ctx context.Context, t require.TestingT) {
	var waitInputs []s3.HeadBucketInput
	for name := range m.knownBuckets {
		listOutput := m.ListObjectVersions(ctx, t, name, nil)
		if len(listOutput.DeleteMarkers)+len(listOutput.Versions) > 0 {
			var objectIds []types.ObjectIdentifier
			for _, dm := range listOutput.DeleteMarkers {
				objectIds = append(objectIds, types.ObjectIdentifier{Key: dm.Key, VersionId: dm.VersionId})
			}
			for _, obj := range listOutput.Versions {
				objectIds = append(objectIds, types.ObjectIdentifier{Key: obj.Key, VersionId: obj.VersionId})
			}
			deleteObjectsInput := s3.DeleteObjectsInput{Bucket: aws.String(name), Delete: &types.Delete{Objects: objectIds}}
			if deleteObjectsOutput, err := m.s3Client.DeleteObjects(ctx, &deleteObjectsInput); err != nil {
				assert.FailNow(t, "error deleting test objects", "bucket: %s, error: %v", name, err)
			} else if len(deleteObjectsOutput.Errors) > 0 {
				// Convert AWS Errors to string so that all the pointers AWS uses become de-referenced and readable in the output
				var errs []string
				for _, err := range deleteObjectsOutput.Errors {
					errs = append(errs, awsErrorToString(name, err))
				}
				assert.FailNow(t, "errors deleting test objects", "bucket: %s, errors: %v", name, errs)
			}
		}
		deleteBucketInput := s3.DeleteBucketInput{Bucket: aws.String(name)}
		if _, err := m.s3Client.DeleteBucket(ctx, &deleteBucketInput); err != nil {
			assert.FailNow(t, "error deleting test bucket", "bucket: %s, error: %v", name, err)
		}
		waitInputs = append(waitInputs, s3.HeadBucketInput{Bucket: aws.String(name)})
	}
	if err := waitForEverything(waitInputs, func(i s3.HeadBucketInput) error {
		return s3.NewBucketNotExistsWaiter(m.s3Client).Wait(ctx, &i, time.Minute)
	}); err != nil {
		assert.FailNow(t, "test bucket not deleted", err)
	}
}

func waitForEverything[T any](inputs []T, waitFn func(T) error) error {
	var wg sync.WaitGroup
	waitErrors := make([]error, len(inputs))
	for index, input := range inputs {
		wg.Add(1)
		go func(i int, in T) {
			defer wg.Done()
			waitErrors[i] = waitFn(in)
		}(index, input)
	}
	wg.Wait()
	for _, we := range waitErrors {
		if we != nil {
			return we
		}
	}
	return nil
}
