package manifests

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pennsieve/collections-service/internal/api/publishing"
	"log/slog"
)

type Store interface {
	SaveManifest(ctx context.Context, key string, manifest publishing.ManifestV5) (SaveManifestResponse, error)
	DeleteManifest(ctx context.Context, key string) error
}

type S3Store struct {
	s3            *s3.Client
	publishBucket string
	logger        *slog.Logger
}

func NewS3Store(s3Client *s3.Client, publishBucket string, logger *slog.Logger) *S3Store {
	return &S3Store{
		s3:            s3Client,
		publishBucket: publishBucket,
		logger:        logger,
	}
}

func (s *S3Store) SaveManifest(ctx context.Context, key string, manifest publishing.ManifestV5) (SaveManifestResponse, error) {
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

func (s *S3Store) DeleteManifest(ctx context.Context, key string) error {
	//TODO implement me
	panic("implement me")
}
