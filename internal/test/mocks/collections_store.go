package mocks

import (
	"context"
	"github.com/pennsieve/collections-service/internal/api/store"
)

type CreateCollectionsFunc func(ctx context.Context, userID int64, nodeID, name, description string, dois []string) (store.CreateCollectionResponse, error)

type GetCollectionsFunc func(ctx context.Context, userID int64, limit int, offset int) (store.GetCollectionsResponse, error)

type GetCollectionFunc func(ctx context.Context, userID int64, nodeID string) (*store.GetCollectionResponse, error)

type DeleteCollectionFunc func(ctx context.Context, id int64) error

type CollectionsStore struct {
	CreateCollectionsFunc
	GetCollectionsFunc
	GetCollectionFunc
	DeleteCollectionFunc
}

func NewMockCollectionsStore() *CollectionsStore {
	return &CollectionsStore{}
}

func (c *CollectionsStore) WithCreateCollectionsFunc(f CreateCollectionsFunc) *CollectionsStore {
	c.CreateCollectionsFunc = f
	return c
}

func (c *CollectionsStore) WithGetCollectionsFunc(f GetCollectionsFunc) *CollectionsStore {
	c.GetCollectionsFunc = f
	return c
}

func (c *CollectionsStore) WithGetCollectionFunc(f GetCollectionFunc) *CollectionsStore {
	c.GetCollectionFunc = f
	return c
}

func (c *CollectionsStore) WithDeleteCollectionFunc(f DeleteCollectionFunc) *CollectionsStore {
	c.DeleteCollectionFunc = f
	return c
}

func (c *CollectionsStore) CreateCollection(ctx context.Context, userID int64, nodeID, name, description string, dois []string) (store.CreateCollectionResponse, error) {
	if c.CreateCollectionsFunc == nil {
		panic("mock CreateCollections function not set")
	}
	return c.CreateCollectionsFunc(ctx, userID, nodeID, name, description, dois)
}

func (c *CollectionsStore) GetCollections(ctx context.Context, userID int64, limit int, offset int) (store.GetCollectionsResponse, error) {
	if c.GetCollectionsFunc == nil {
		panic("mock GetCollections function not set")
	}
	return c.GetCollectionsFunc(ctx, userID, limit, offset)
}

func (c *CollectionsStore) GetCollection(ctx context.Context, userID int64, nodeID string) (*store.GetCollectionResponse, error) {
	if c.GetCollectionFunc == nil {
		panic("mock GetCollection function not set")
	}
	return c.GetCollectionFunc(ctx, userID, nodeID)
}

func (c *CollectionsStore) DeleteCollection(ctx context.Context, id int64) error {
	if c.DeleteCollectionFunc == nil {
		panic("mock DeleteCollection function not set")
	}
	return c.DeleteCollection(ctx, id)
}
