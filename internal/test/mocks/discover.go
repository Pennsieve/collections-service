package mocks

import (
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/stretchr/testify/require"
)

type GetDatasetsByDOIFunc func(dois []string) (dto.DatasetsByDOIResponse, error)

type Discover struct {
	t require.TestingT
	GetDatasetsByDOIFunc
}

func NewMockDiscover(t require.TestingT) *Discover {
	test.Helper(t)
	return &Discover{t: t}
}

func (d *Discover) GetDatasetsByDOI(dois []string) (dto.DatasetsByDOIResponse, error) {
	require.NotNil(d.t, d.GetDatasetsByDOIFunc, "mock GetDatasetsByDOI function not set")
	return d.GetDatasetsByDOIFunc(dois)
}
