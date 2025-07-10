package mocks

import (
	"context"
	"github.com/pennsieve/collections-service/internal/api/publishing"
	"github.com/pennsieve/collections-service/internal/api/store/manifests"
)

type SaveManifestFunc func(ctx context.Context, key string, manifest publishing.ManifestV5) (manifests.SaveManifestResponse, error)
type DeleteManifestVersionFunc func(ctx context.Context, key string, s3VersionID string) error
type ManifestStore struct {
	SaveManifestFunc
	DeleteManifestVersionFunc
}

func NewManifestStore() *ManifestStore {
	return &ManifestStore{}
}

func (m *ManifestStore) SaveManifest(ctx context.Context, key string, manifest publishing.ManifestV5) (manifests.SaveManifestResponse, error) {
	if m.SaveManifestFunc == nil {
		panic("mock SaveManifest function not set")
	}
	return m.SaveManifestFunc(ctx, key, manifest)
}

func (m *ManifestStore) DeleteManifestVersion(ctx context.Context, key string, s3VersionID string) error {
	if m.DeleteManifestVersionFunc == nil {
		panic("mock DeleteManifest function not set")
	}
	return m.DeleteManifestVersion(ctx, key, s3VersionID)
}

func (m *ManifestStore) WithSaveManifestFunc(saveManifestFunc SaveManifestFunc) *ManifestStore {
	m.SaveManifestFunc = saveManifestFunc
	return m
}

func (m *ManifestStore) WithDeleteManifestVersionFunc(deleteManifestFunc DeleteManifestVersionFunc) *ManifestStore {
	m.DeleteManifestVersionFunc = deleteManifestFunc
	return m
}
