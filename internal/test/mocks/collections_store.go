package mocks

import (
	"context"
	"github.com/pennsieve/collections-service/internal/api/store"
)

type CreateCollectionsFunc func(ctx context.Context, userID int64, nodeID, name, description string, dois []string) (*store.CreateCollectionResponse, error)
type CollectionsStore struct {
	CreateCollectionsFunc
}

func NewMockCollectionsStore() *CollectionsStore {
	return &CollectionsStore{}
}

func (c *CollectionsStore) WithCreateCollectionsFunc(f CreateCollectionsFunc) *CollectionsStore {
	c.CreateCollectionsFunc = f
	return c
}

func (c *CollectionsStore) CreateCollection(ctx context.Context, userID int64, nodeID, name, description string, dois []string) (*store.CreateCollectionResponse, error) {
	if c.CreateCollectionsFunc == nil {
		panic("mock CreateCollections function not set")
	}
	return c.CreateCollectionsFunc(ctx, userID, nodeID, name, description, dois)
}
