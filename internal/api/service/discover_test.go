package service_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/collections-service/internal/shared/logging"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/pennsieve/collections-service/internal/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPDiscover_GetDatasetsByDOI(t *testing.T) {
	ctx := context.Background()
	publishedDOI := apitest.NewPennsieveDOI()
	publishedDTO := apitest.NewPublicDataset(publishedDOI.Value, apitest.NewBanner())
	expectedResponse := service.DatasetsByDOIResponse{
		Published:   map[string]dto.PublicDataset{publishedDOI.Value: publishedDTO},
		Unpublished: nil,
	}
	discoverServer := httptest.NewServer(mocks.ToDiscoverHandlerFunc(ctx, t, func(ctx context.Context, dois []string) (service.DatasetsByDOIResponse, error) {
		return expectedResponse, nil
	}))
	defer discoverServer.Close()

	discover := service.NewHTTPDiscover(discoverServer.URL, logging.Default)

	response, err := discover.GetDatasetsByDOI(ctx, []string{publishedDOI.Value})
	require.NoError(t, err)
	assert.Equal(t, expectedResponse, response)

}

// TestHTTPDiscover_GetDatasetsByDOI_Empty verifies that calling with no DOIs
// short-circuits without making any HTTP requests to discover.
func TestHTTPDiscover_GetDatasetsByDOI_Empty(t *testing.T) {
	ctx := context.Background()

	var requestCount atomic.Int32
	discoverServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
	}))
	defer discoverServer.Close()

	discover := service.NewHTTPDiscover(discoverServer.URL, logging.Default)

	response, err := discover.GetDatasetsByDOI(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, response.Published)
	assert.Empty(t, response.Unpublished)

	response, err = discover.GetDatasetsByDOI(ctx, []string{})
	require.NoError(t, err)
	assert.Empty(t, response.Published)
	assert.Empty(t, response.Unpublished)

	assert.Zero(t, requestCount.Load(), "no HTTP requests should be made for an empty DOI list")
}

// TestHTTPDiscover_GetDatasetsByDOI_Batches verifies that a large DOI set is
// split across multiple discover requests so that no single request exceeds
// the discover front-end's URI length limit, and that the per-batch responses
// are merged correctly.
func TestHTTPDiscover_GetDatasetsByDOI_Batches(t *testing.T) {

	const largeCollectionSize = 500
	const discoverURILimit = 2048

	ctx := context.Background()
	expectedDatasets := apitest.NewExpectedPennsieveDatasets()
	var dois []string
	for range largeCollectionSize {
		dois = append(dois, expectedDatasets.NewPublished().DOI)
	}

	var requestCount atomic.Int32
	var maxURILen atomic.Int32
	discoverServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		if int32(len(r.URL.RequestURI())) > maxURILen.Load() {
			maxURILen.Store(int32(len(r.URL.RequestURI())))
		}
		mocks.ToDiscoverHandlerFunc(ctx, t, expectedDatasets.GetDatasetsByDOIFunc(t)).ServeHTTP(w, r)
	}))
	defer discoverServer.Close()

	discover := service.NewHTTPDiscover(discoverServer.URL, logging.Default)

	response, err := discover.GetDatasetsByDOI(ctx, dois)
	require.NoError(t, err)

	assert.Len(t, response.Published, largeCollectionSize)
	for _, doi := range dois {
		assert.Contains(t, response.Published, doi)
	}
	assert.Greater(t, requestCount.Load(), int32(1), "expected multiple batched requests for %d DOIs", largeCollectionSize)
	assert.LessOrEqual(t, maxURILen.Load(), int32(discoverURILimit), "no batch request URI should exceed the discover URI limit")
}
