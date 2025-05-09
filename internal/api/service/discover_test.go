package service_test

import (
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/collections-service/internal/shared/logging"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/pennsieve/collections-service/internal/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"testing"
)

func TestHTTPDiscover_GetDatasetsByDOI(t *testing.T) {
	publishedDOI := apitest.NewPennsieveDOI()
	publishedDTO := apitest.NewPublicDataset(publishedDOI.Value, apitest.NewBanner())
	expectedResponse := service.DatasetsByDOIResponse{
		Published:   map[string]dto.PublicDataset{publishedDOI.Value: publishedDTO},
		Unpublished: nil,
	}
	discoverServer := httptest.NewServer(mocks.ToDiscoverHandlerFunc(t, func(dois []string) (service.DatasetsByDOIResponse, error) {
		return expectedResponse, nil
	}))
	defer discoverServer.Close()

	discover := service.NewHTTPDiscover(discoverServer.URL, logging.Default)

	response, err := discover.GetDatasetsByDOI([]string{publishedDOI.Value})
	require.NoError(t, err)
	assert.Equal(t, expectedResponse, response)

}
