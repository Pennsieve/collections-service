package manifests

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pennsieve/collections-service/internal/api/publishing"
	"log/slog"
	"time"
)

type Store interface {
	SaveManifest(ctx context.Context, key string, manifest publishing.ManifestV5) (SaveManifestResponse, error)
	DeleteManifestVersion(ctx context.Context, key string, s3VersionID string) error
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
	versionId := aws.ToString(putOut.VersionId)
	s.logger.Debug("wrote manifest",
		slog.String("logSource", "S3Store"),
		slog.String("bucket", s.publishBucket),
		slog.String("key", key),
		slog.String("s3VersionId", versionId),
	)
	//TODO Remove this!
	time.Sleep(time.Second)
	return SaveManifestResponse{S3VersionID: versionId}, nil
}

func (s *S3Store) DeleteManifestVersion(ctx context.Context, key string, s3VersionID string) error {
	deleteIn := s3.DeleteObjectInput{
		Bucket:    aws.String(s.publishBucket),
		Key:       aws.String(key),
		VersionId: aws.String(s3VersionID),
	}
	_, err := s.s3.DeleteObject(ctx, &deleteIn)
	if err != nil {
		return fmt.Errorf("error deleting manifest version %s at %s/%s: %w",
			s3VersionID, s.publishBucket, key, err)
	}
	return nil
}
