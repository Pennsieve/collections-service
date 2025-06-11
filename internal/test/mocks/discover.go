package mocks

import (
	"github.com/pennsieve/collections-service/internal/api/service"
)

type GetDatasetsByDOIFunc func(dois []string) (service.DatasetsByDOIResponse, error)

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

func (d *Discover) GetDatasetsByDOI(dois []string) (service.DatasetsByDOIResponse, error) {
	if d.GetDatasetsByDOIFunc == nil {
		panic("mock GetDatasetsByDOI function not set")
	}
	return d.GetDatasetsByDOIFunc(dois)
}
