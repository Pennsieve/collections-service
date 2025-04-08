package mocks

import (
	"github.com/pennsieve/collections-service/internal/api/dto"
)

type GetDatasetsByDOIFunc func(dois []string) (dto.DatasetsByDOIResponse, error)

type Discover struct {
	GetDatasetsByDOIFunc
}

func NewMockDiscover() *Discover {
	return &Discover{}
}

func (d *Discover) WithGetDatasetsByDOIFunc(f GetDatasetsByDOIFunc) *Discover {
	d.GetDatasetsByDOIFunc = f
	return d
}

func (d *Discover) GetDatasetsByDOI(dois []string) (dto.DatasetsByDOIResponse, error) {
	if d.GetDatasetsByDOIFunc == nil {
		panic("mock GetDatasetsByDOI function not set")
	}
	return d.GetDatasetsByDOIFunc(dois)
}
