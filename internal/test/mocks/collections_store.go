package mocks

import (
	"context"
	"github.com/pennsieve/collections-service/internal/api/publishing"
	"github.com/pennsieve/collections-service/internal/api/store/collections"
)

type CreateCollectionsFunc func(ctx context.Context, request collections.CreateCollectionRequest) (collections.CreateCollectionResponse, error)

type GetCollectionsFunc func(ctx context.Context, userID int64, limit int, offset int) (collections.GetCollectionsResponse, error)

type GetCollectionFunc func(ctx context.Context, userID int64, nodeID string) (collections.GetCollectionResponse, error)

type DeleteCollectionFunc func(ctx context.Context, collectionID int64) error

type UpdateCollectionFunc func(ctx context.Context, userID int64, collectionID int64, update collections.UpdateCollectionRequest) (collections.GetCollectionResponse, error)

type StartPublishFunc func(ctx context.Context, collectionID int64, userID int64, publishingType publishing.Type) error

type FinishPublishFunc func(ctx context.Context, collectionID int64, publishingStatus publishing.Status, strict bool) error

type CollectionsStore struct {
	CreateCollectionsFunc
	GetCollectionsFunc
	GetCollectionFunc
	DeleteCollectionFunc
	UpdateCollectionFunc
	StartPublishFunc
	FinishPublishFunc
}

func NewCollectionsStore() *CollectionsStore {
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

func (c *CollectionsStore) WithUpdateCollectionFunc(f UpdateCollectionFunc) *CollectionsStore {
	c.UpdateCollectionFunc = f
	return c
}

func (c *CollectionsStore) WithStartPublishFunc(f StartPublishFunc) *CollectionsStore {
	c.StartPublishFunc = f
	return c
}

func (c *CollectionsStore) WithFinishPublishFunc(f FinishPublishFunc) *CollectionsStore {
	c.FinishPublishFunc = f
	return c
}

func (c *CollectionsStore) CreateCollection(ctx context.Context, request collections.CreateCollectionRequest) (collections.CreateCollectionResponse, error) {
	if c.CreateCollectionsFunc == nil {
		panic("mock CreateCollections function not set")
	}
	return c.CreateCollectionsFunc(ctx, request)
}

func (c *CollectionsStore) GetCollections(ctx context.Context, userID int64, limit int, offset int) (collections.GetCollectionsResponse, error) {
	if c.GetCollectionsFunc == nil {
		panic("mock GetCollections function not set")
	}
	return c.GetCollectionsFunc(ctx, userID, limit, offset)
}

func (c *CollectionsStore) GetCollection(ctx context.Context, userID int64, nodeID string) (collections.GetCollectionResponse, error) {
	if c.GetCollectionFunc == nil {
		panic("mock GetCollection function not set")
	}
	return c.GetCollectionFunc(ctx, userID, nodeID)
}

func (c *CollectionsStore) DeleteCollection(ctx context.Context, collectionID int64) error {
	if c.DeleteCollectionFunc == nil {
		panic("mock DeleteCollection function not set")
	}
	return c.DeleteCollectionFunc(ctx, collectionID)
}

func (c *CollectionsStore) UpdateCollection(ctx context.Context, userID, collectionID int64, update collections.UpdateCollectionRequest) (collections.GetCollectionResponse, error) {
	if c.UpdateCollectionFunc == nil {
		panic("mock UpdateCollection function not set")
	}
	return c.UpdateCollectionFunc(ctx, userID, collectionID, update)
}

func (c *CollectionsStore) StartPublish(ctx context.Context, collectionID int64, userID int64, publishingType publishing.Type) error {
	if c.StartPublishFunc == nil {
		panic("mock StartPublish function not set")
	}
	return c.StartPublishFunc(ctx, collectionID, userID, publishingType)
}

func (c *CollectionsStore) FinishPublish(ctx context.Context, collectionID int64, publishingStatus publishing.Status, strict bool) error {
	if c.FinishPublishFunc == nil {
		panic("mock FinishPublish function not set")
	}
	return c.FinishPublishFunc(ctx, collectionID, publishingStatus, strict)
}
