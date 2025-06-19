package mocks

import (
	"context"
	"github.com/pennsieve/collections-service/internal/api/publishing"
	"github.com/pennsieve/collections-service/internal/api/store/manifests"
)

type SaveManifestFunc func(ctx context.Context, key string, manifest publishing.ManifestV5) (manifests.SaveManifestResponse, error)
type DeleteManifestFunc func(ctx context.Context, key string) error
type ManifestStore struct {
	SaveManifestFunc
	DeleteManifestFunc
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

func (m *ManifestStore) DeleteManifest(ctx context.Context, key string) error {
	if m.DeleteManifestFunc == nil {
		panic("mock DeleteManifest function not set")
	}
	return m.DeleteManifest(ctx, key)
}

func (m *ManifestStore) WithSaveManifestFunc(saveManifestFunc SaveManifestFunc) *ManifestStore {
	m.SaveManifestFunc = saveManifestFunc
	return m
}

func (m *ManifestStore) WithDeleteManifestFunc(deleteManifestFunc DeleteManifestFunc) *ManifestStore {
	m.DeleteManifestFunc = deleteManifestFunc
	return m
}
