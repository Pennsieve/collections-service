package mocks

import (
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
)

type PublishCollectionFunc func(collectionID int64, userRole role.Role, request service.PublishDOICollectionRequest) (service.PublishDOICollectionResponse, error)

type InternalDiscover struct {
	PublishCollectionFunc
}

func NewInternalDiscover() *InternalDiscover {
	return &InternalDiscover{}
}

func (i *InternalDiscover) WithPublishCollectionFunc(publishCollectionFunc PublishCollectionFunc) *InternalDiscover {
	i.PublishCollectionFunc = publishCollectionFunc
	return i
}

func (i *InternalDiscover) PublishCollection(collectionID int64, userRole role.Role, request service.PublishDOICollectionRequest) (service.PublishDOICollectionResponse, error) {
	if i.PublishCollectionFunc == nil {
		panic("mock PublishCollection function not set")
	}
	return i.PublishCollectionFunc(collectionID, userRole, request)
}
