package manifests_test

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/pennsieve/collections-service/internal/api/publishing"
	"github.com/pennsieve/collections-service/internal/api/store/manifests"
	"github.com/pennsieve/collections-service/internal/shared/logging"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/pennsieve/collections-service/internal/test/fixtures"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestS3Store(t *testing.T) {
	tests := []struct {
		scenario string
		tstFunc  func(t *testing.T, minio *fixtures.MinIO)
	}{
		{"SaveManifest should save the manifest correctly", testSaveManifest},
		{"SaveManifest should save manifest versions correctly", testSaveManifestVersions},
		{"DeleteManifestVersion should delete the manifest version correctly", testDeleteManifestVersion},
	}

	for _, tt := range tests {
		ctx := context.Background()
		minio := fixtures.NewMinIOWithDefaultClient(ctx, t)
		t.Run(tt.scenario, func(t *testing.T) {
			t.Cleanup(func() {
				minio.CleanUp(ctx, t)
			})
			tt.tstFunc(t, minio)
		})

	}
}

func testSaveManifest(t *testing.T, minio *fixtures.MinIO) {
	ctx := context.Background()

	bucket := minio.CreatePublishBucket(ctx, t)

	manifestStore := manifests.NewS3Store(test.DefaultMinIOS3Client(ctx, t), bucket, logging.Default)

	manifest := apitest.NewExpectedManifest(t,
		apitest.WithManifestPennsieveDatasetID(34),
		apitest.WithManifestVersion(3),
	)

	key := manifest.S3Key()

	response, err := manifestStore.SaveManifest(ctx, key, manifest)
	require.NoError(t, err)

	headOut := minio.RequireObjectExists(ctx, t, bucket, key)
	require.Equal(t, response.S3VersionID, aws.ToString(headOut.VersionId))

	var actualManifest publishing.ManifestV5
	minio.GetObject(ctx, t, bucket, key, headOut.VersionId).As(t, &actualManifest)

	apitest.RequireManifestsEqual(t, manifest, actualManifest)

}

func testSaveManifestVersions(t *testing.T, minio *fixtures.MinIO) {
	ctx := context.Background()

	bucket := minio.CreatePublishBucket(ctx, t)

	manifestStore := manifests.NewS3Store(test.DefaultMinIOS3Client(ctx, t), bucket, logging.Default)

	expectedDatasetID := int64(48)
	key := publishing.S3Key(expectedDatasetID)

	expectedManifestV1 := apitest.NewExpectedManifest(t,
		apitest.WithManifestPennsieveDatasetID(expectedDatasetID),
		apitest.WithManifestVersion(1),
	)
	response, err := manifestStore.SaveManifest(ctx, key, expectedManifestV1)
	require.NoError(t, err)

	s3VersionIDV1 := response.S3VersionID

	expectedManifestV2 := apitest.NewExpectedManifest(t,
		apitest.WithManifestPennsieveDatasetID(expectedDatasetID),
		apitest.WithManifestVersion(2),
	)

	response, err = manifestStore.SaveManifest(ctx, key, expectedManifestV2)
	require.NoError(t, err)

	s3VersionIDV2 := response.S3VersionID

	headOut := minio.RequireObjectExists(ctx, t, bucket, key)
	// Head with no versionID info should return the most recent version V2
	require.Equal(t, s3VersionIDV2, aws.ToString(headOut.VersionId))

	var actualManifestV1 publishing.ManifestV5
	minio.GetObject(ctx, t, bucket, key, &s3VersionIDV1).As(t, &actualManifestV1)

	apitest.RequireManifestsEqual(t, expectedManifestV1, actualManifestV1)

	var actualManifestV2 publishing.ManifestV5
	minio.GetObject(ctx, t, bucket, key, &s3VersionIDV2).As(t, &actualManifestV2)

	apitest.RequireManifestsEqual(t, expectedManifestV2, actualManifestV2)

}

func testDeleteManifestVersion(t *testing.T, minio *fixtures.MinIO) {
	ctx := context.Background()

	bucket := minio.CreatePublishBucket(ctx, t)

	manifestStore := manifests.NewS3Store(test.DefaultMinIOS3Client(ctx, t), bucket, logging.Default)

	expectedDatasetID := int64(48)
	key := publishing.S3Key(expectedDatasetID)

	expectedManifestV1 := apitest.NewExpectedManifest(t,
		apitest.WithManifestPennsieveDatasetID(expectedDatasetID),
		apitest.WithManifestVersion(1),
	)
	response, err := manifestStore.SaveManifest(ctx, key, expectedManifestV1)
	require.NoError(t, err)

	s3VersionIDV1 := response.S3VersionID

	expectedManifestV2 := apitest.NewExpectedManifest(t,
		apitest.WithManifestPennsieveDatasetID(expectedDatasetID),
		apitest.WithManifestVersion(2),
	)

	response, err = manifestStore.SaveManifest(ctx, key, expectedManifestV2)
	require.NoError(t, err)

	s3VersionIDV2 := response.S3VersionID

	// Head with no versionID info should return the most recent version V2
	headOut := minio.RequireObjectExists(ctx, t, bucket, key)
	require.Equal(t, s3VersionIDV2, aws.ToString(headOut.VersionId))

	// Delete most recent version
	require.NoError(t, manifestStore.DeleteManifestVersion(ctx, key, s3VersionIDV2))

	// Now Head should return V1 versionID
	headOut = minio.RequireObjectExists(ctx, t, bucket, key)
	require.Equal(t, s3VersionIDV1, aws.ToString(headOut.VersionId))

	// Delete the remaining version
	require.NoError(t, manifestStore.DeleteManifestVersion(ctx, key, s3VersionIDV1))

	minio.RequireNoObject(ctx, t, bucket, key)

}
