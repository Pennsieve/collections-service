package mocks

import (
	"context"
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
)

type PublishCollectionFunc func(ctx context.Context, collectionID int64, userRole role.Role, request service.PublishDOICollectionRequest) (service.PublishDOICollectionResponse, error)
type FinalizeCollectionPublishFunc func(ctx context.Context, collectionID int64, collectionNodeID string, userRole role.Role, request service.FinalizeDOICollectionPublishRequest) (service.FinalizeDOICollectionPublishResponse, error)
type GetCollectionPublishStatusFunc func(ctx context.Context, collectionID int64, collectionNodeID string, userRole role.Role) (service.DatasetPublishStatusResponse, error)
type UnpublishCollectionFunc func(ctx context.Context, collectionID int64, collectionNodeID string, userRole role.Role) (service.DatasetPublishStatusResponse, error)

type InternalDiscover struct {
	PublishCollectionFunc
	FinalizeCollectionPublishFunc
	GetCollectionPublishStatusFunc
	UnpublishCollectionFunc
}

func NewInternalDiscover() *InternalDiscover {
	return &InternalDiscover{}
}

func (i *InternalDiscover) WithPublishCollectionFunc(publishCollectionFunc PublishCollectionFunc) *InternalDiscover {
	i.PublishCollectionFunc = publishCollectionFunc
	return i
}

func (i *InternalDiscover) WithFinalizeCollectionPublishFunc(f FinalizeCollectionPublishFunc) *InternalDiscover {
	i.FinalizeCollectionPublishFunc = f
	return i
}

func (i *InternalDiscover) WithGetCollectionPublishStatusFunc(f GetCollectionPublishStatusFunc) *InternalDiscover {
	i.GetCollectionPublishStatusFunc = f
	return i
}

func (i *InternalDiscover) WithUnpublishCollectionFunc(f UnpublishCollectionFunc) *InternalDiscover {
	i.UnpublishCollectionFunc = f
	return i
}

func (i *InternalDiscover) PublishCollection(ctx context.Context, collectionID int64, userRole role.Role, request service.PublishDOICollectionRequest) (service.PublishDOICollectionResponse, error) {
	if i.PublishCollectionFunc == nil {
		panic("mock PublishCollection function not set")
	}
	return i.PublishCollectionFunc(ctx, collectionID, userRole, request)
}

func (i *InternalDiscover) FinalizeCollectionPublish(ctx context.Context, collectionID int64, collectionNodeID string, userRole role.Role, request service.FinalizeDOICollectionPublishRequest) (service.FinalizeDOICollectionPublishResponse, error) {
	if i.FinalizeCollectionPublishFunc == nil {
		panic("mock FinalizeCollectionPublish function not set")
	}
	return i.FinalizeCollectionPublishFunc(ctx, collectionID, collectionNodeID, userRole, request)
}

func (i *InternalDiscover) GetCollectionPublishStatus(ctx context.Context, collectionID int64, collectionNodeID string, userRole role.Role) (service.DatasetPublishStatusResponse, error) {
	if i.GetCollectionPublishStatusFunc == nil {
		panic("mock GetCollectionPublishStatus function not set")
	}
	return i.GetCollectionPublishStatusFunc(ctx, collectionID, collectionNodeID, userRole)
}

func (i *InternalDiscover) UnpublishCollection(ctx context.Context, collectionID int64, collectionNodeID string, userRole role.Role) (service.DatasetPublishStatusResponse, error) {
	if i.UnpublishCollectionFunc == nil {
		panic("mock UnpublishCollection function not set")
	}
	return i.UnpublishCollectionFunc(ctx, collectionID, collectionNodeID, userRole)
}
