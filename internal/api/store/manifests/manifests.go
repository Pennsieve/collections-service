package manifests

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pennsieve/collections-service/internal/api/publishing"
)

type ManifestStore interface {
	SaveManifest(ctx context.Context, manifest publishing.ManifestV5) (SaveManifestResponse, error)
	DeleteManifest(ctx context.Context) error
}

type S3ManifestStore struct {
	s3            *s3.Client
	publishBucket string
}

func (s *S3ManifestStore) SaveManifest(ctx context.Context, key string, manifest publishing.ManifestV5) (SaveManifestResponse, error) {
	manifestBytes, err := manifest.Marshal()
	if err != nil {
		return SaveManifestResponse{}, fmt.Errorf("error marshalling manifest for uploading to %s/$%s: %w",
			s.publishBucket,
			key,
			err)
	}
	putIn := s3.PutObjectInput{
		Bucket: aws.String(s.publishBucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(manifestBytes),
	}
	putOut, err := s.s3.PutObject(ctx, &putIn)
	if err != nil {
		return SaveManifestResponse{}, fmt.Errorf("error writing manifest to %s/%s: %w", s.publishBucket, key, err)
	}
	return SaveManifestResponse{S3VersionID: aws.ToString(putOut.VersionId)}, err
}

func (s *S3ManifestStore) DeleteManifest(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}
