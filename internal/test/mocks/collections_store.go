package mocks

import (
	"context"
	"github.com/pennsieve/collections-service/internal/api/store"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/stretchr/testify/require"
)

type CreateCollectionsFunc func(ctx context.Context, userID int64, nodeID, name, description string, dois []string) (*store.CreateCollectionResponse, error)
type CollectionsStore struct {
	t require.TestingT
	CreateCollectionsFunc
}

func NewMockCollectionsStore(t require.TestingT) *CollectionsStore {
	test.Helper(t)
	return &CollectionsStore{t: t}
}

func (c *CollectionsStore) CreateCollection(ctx context.Context, userID int64, nodeID, name, description string, dois []string) (*store.CreateCollectionResponse, error) {
	require.NotNil(c.t, c.CreateCollectionsFunc, "mock CreateCollections function not set")
	return c.CreateCollectionsFunc(ctx, userID, nodeID, name, description, dois)
}
