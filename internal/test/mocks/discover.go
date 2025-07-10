package mocks

import (
	"context"
	"github.com/pennsieve/collections-service/internal/api/service"
)

type GetDatasetsByDOIFunc func(ctx context.Context, dois []string) (service.DatasetsByDOIResponse, error)

type Discover struct {
	GetDatasetsByDOIFunc
}

func NewDiscover() *Discover {
	return &Discover{}
}

func (d *Discover) WithGetDatasetsByDOIFunc(f GetDatasetsByDOIFunc) *Discover {
	d.GetDatasetsByDOIFunc = f
	return d
}

func (d *Discover) GetDatasetsByDOI(ctx context.Context, dois []string) (service.DatasetsByDOIResponse, error) {
	if d.GetDatasetsByDOIFunc == nil {
		panic("mock GetDatasetsByDOI function not set")
	}
	return d.GetDatasetsByDOIFunc(ctx, dois)
}
