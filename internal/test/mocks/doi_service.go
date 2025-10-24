package mocks

import (
	"context"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
)

type GetLatestDOIFunc func(ctx context.Context, collectionID int64, collectionNodeID string, userRole role.Role) (dto.GetLatestDOIResponse, error)

type DOI struct {
	GetLatestDOIFunc
}

func NewDOI() *DOI { return &DOI{} }

func (d *DOI) WithGetLatestDOIFunc(f GetLatestDOIFunc) *DOI {
	d.GetLatestDOIFunc = f
	return d
}

func (d *DOI) GetLatestDOI(ctx context.Context, collectionID int64, collectionNodeID string, userRole role.Role) (dto.GetLatestDOIResponse, error) {
	if d.GetLatestDOIFunc == nil {
		panic("mock GetLatestDOI function not set")
	}
	return d.GetLatestDOIFunc(ctx, collectionID, collectionNodeID, userRole)
}
