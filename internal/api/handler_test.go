package api

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/collections-service/internal/api/store"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/pennsieve/collections-service/internal/test/mocks"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"strings"
	"testing"
)

func TestAPILambdaHandler(t *testing.T) {
	for scenario, fn := range map[string]func(
		tt *testing.T,
	){
		"default not found response": testDefaultNotFound,
		"no claims":                  testNoClaims,
		"create collection bad request: external DOIs":        testCreateCollectionExternalDOIs,
		"create collection bad request: empty name":           testCreateCollectionEmptyName,
		"create collection bad request: name too long":        testCreateCollectionNameTooLong,
		"create collection bad request: description too long": testCreateCollectionDescriptionTooLong,
		"create collection bad request: unpublished DOIs":     testCreateCollectionUnpublishedDOIs,
		"create collection bad request: no body":              testCreateCollectionNoBody,
		"create collection bad request: malformed body":       testCreateCollectionMalformedBody,
		"create collection":                                   testCreateCollection,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func testDefaultNotFound(t *testing.T) {
	handler := CollectionsServiceAPIHandler(apitest.NewTestContainer(), apitest.NewConfigBuilder().Build())

	req := test.NewAPIGatewayRequestBuilder("GET /unknown").
		WithDefaultClaims(test.User2).
		Build()

	response, err := handler(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, response.StatusCode)
	assert.Contains(t, response.Body, "not found")
	assert.Contains(t, response.Body, `"error_id"`)
}

func testNoClaims(t *testing.T) {
	handler := CollectionsServiceAPIHandler(apitest.NewTestContainer(), apitest.NewConfigBuilder().Build())

	req := test.NewAPIGatewayRequestBuilder("POST /collections").Build()

	response, err := handler(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

func testCreateCollectionExternalDOIs(t *testing.T) {
	createCollectionRequest := dto.CreateCollectionRequest{
		Name:        uuid.NewString(),
		Description: uuid.NewString(),
		DOIs:        []string{test.NewExternalDOI(), test.NewPennsieveDOI(), test.NewExternalDOI()},
	}

	handler := CollectionsServiceAPIHandler(
		apitest.NewTestContainer(),
		apitest.NewConfigBuilder().
			WithPennsieveConfig(apitest.PennsieveConfigWithFakeHost()).
			Build())

	req := test.NewAPIGatewayRequestBuilder("POST /collections").
		WithDefaultClaims(test.User).
		WithBody(t, createCollectionRequest).
		Build()

	response, err := handler(context.Background(), req)

	assert.NoError(t, err)
	assert.Contains(t, response.Body, createCollectionRequest.DOIs[0])
	assert.NotContains(t, response.Body, createCollectionRequest.DOIs[1])
	assert.Contains(t, response.Body, createCollectionRequest.DOIs[2])
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func testCreateCollectionEmptyName(t *testing.T) {
	createCollectionRequest := dto.CreateCollectionRequest{
		Name:        "",
		Description: uuid.NewString(),
	}

	handler := CollectionsServiceAPIHandler(
		apitest.NewTestContainer(),
		apitest.NewConfigBuilder().
			WithPennsieveConfig(apitest.PennsieveConfigWithFakeHost()).
			Build())

	req := test.NewAPIGatewayRequestBuilder("POST /collections").
		WithDefaultClaims(test.User).
		WithBody(t, createCollectionRequest).
		Build()

	response, err := handler(context.Background(), req)

	assert.NoError(t, err)
	assert.Contains(t, response.Body, "collection name cannot be empty")
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func testCreateCollectionNameTooLong(t *testing.T) {
	createCollectionRequest := dto.CreateCollectionRequest{
		Name:        strings.Repeat("a", 256),
		Description: uuid.NewString(),
	}

	handler := CollectionsServiceAPIHandler(
		apitest.NewTestContainer(),
		apitest.NewConfigBuilder().
			WithPennsieveConfig(apitest.PennsieveConfigWithFakeHost()).
			Build())

	req := test.NewAPIGatewayRequestBuilder("POST /collections").
		WithDefaultClaims(test.User).
		WithBody(t, createCollectionRequest).
		Build()

	response, err := handler(context.Background(), req)

	assert.NoError(t, err)
	assert.Contains(t, response.Body, "collection name cannot have more than 255 characters")
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func testCreateCollectionDescriptionTooLong(t *testing.T) {
	createCollectionRequest := dto.CreateCollectionRequest{
		Name:        uuid.NewString(),
		Description: strings.Repeat("a", 256),
	}

	handler := CollectionsServiceAPIHandler(
		apitest.NewTestContainer(),
		apitest.NewConfigBuilder().
			WithPennsieveConfig(apitest.PennsieveConfigWithFakeHost()).
			Build())

	req := test.NewAPIGatewayRequestBuilder("POST /collections").
		WithDefaultClaims(test.User).
		WithBody(t, createCollectionRequest).
		Build()

	response, err := handler(context.Background(), req)

	assert.NoError(t, err)
	assert.Contains(t, response.Body, "collection description cannot have more than 255 characters")
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func testCreateCollectionUnpublishedDOIs(t *testing.T) {
	publishedDOI := test.NewPennsieveDOI()
	unpublishedDOI := test.NewPennsieveDOI()
	expectedStatus := "PublishFailed"
	createCollectionRequest := dto.CreateCollectionRequest{
		Name:        uuid.NewString(),
		Description: uuid.NewString(),
		DOIs:        []string{publishedDOI, unpublishedDOI},
	}

	mockDiscoverService := mocks.NewMockDiscover().
		WithGetDatasetsByDOIFunc(func(dois []string) (service.DatasetsByDOIResponse, error) {
			test.Helper(t)
			require.Equal(t, []string{publishedDOI, unpublishedDOI}, dois)
			return service.DatasetsByDOIResponse{
				Published:   map[string]dto.PublicDataset{publishedDOI: test.NewPublicDataset(publishedDOI, nil)},
				Unpublished: map[string]dto.Tombstone{unpublishedDOI: test.NewTombstone(unpublishedDOI, expectedStatus)},
			}, nil
		})

	handler := CollectionsServiceAPIHandler(
		apitest.NewTestContainer().WithDiscover(mockDiscoverService),
		apitest.NewConfigBuilder().
			WithPennsieveConfig(apitest.PennsieveConfigWithFakeHost()).
			Build())

	req := test.NewAPIGatewayRequestBuilder("POST /collections").
		WithDefaultClaims(test.User).
		WithBody(t, createCollectionRequest).
		Build()

	response, err := handler(context.Background(), req)

	assert.NoError(t, err)
	assert.Contains(t, response.Body, unpublishedDOI)
	assert.Contains(t, response.Body, expectedStatus)
	assert.NotContains(t, response.Body, publishedDOI)
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)

}

func testCreateCollectionNoBody(t *testing.T) {
	handler := CollectionsServiceAPIHandler(
		apitest.NewTestContainer(),
		apitest.NewConfigBuilder().Build())

	req := test.NewAPIGatewayRequestBuilder("POST /collections").WithDefaultClaims(test.User).Build()

	response, err := handler(context.Background(), req)
	require.NoError(t, err)

	assert.Contains(t, response.Body, "no request body")
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)

}

func testCreateCollectionMalformedBody(t *testing.T) {
	handler := CollectionsServiceAPIHandler(
		apitest.NewTestContainer(),
		apitest.NewConfigBuilder().Build())

	req := test.NewAPIGatewayRequestBuilder("POST /collections").
		WithDefaultClaims(test.User).
		Build()

	req.Body = "{]"

	response, err := handler(context.Background(), req)
	require.NoError(t, err)

	assert.Contains(t, response.Body, "error unmarshalling request body")
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)

}

func testCreateCollection(t *testing.T) {
	publishedDOI1 := test.NewPennsieveDOI()
	banner1 := test.NewBanner()

	publishedDOI2 := test.NewPennsieveDOI()
	banner2 := test.NewBanner()

	callingUser := test.User

	createCollectionRequest := dto.CreateCollectionRequest{
		Name:        uuid.NewString(),
		Description: uuid.NewString(),
		DOIs:        []string{publishedDOI1, publishedDOI2},
	}

	mockDiscoverService := mocks.NewMockDiscover().
		WithGetDatasetsByDOIFunc(func(dois []string) (service.DatasetsByDOIResponse, error) {
			t.Helper()
			require.Equal(t, []string{publishedDOI1, publishedDOI2}, dois)
			return service.DatasetsByDOIResponse{
				Published: map[string]dto.PublicDataset{
					publishedDOI1: test.NewPublicDataset(publishedDOI1, banner1),
					publishedDOI2: test.NewPublicDataset(publishedDOI2, banner2)},
			}, nil
		})

	var collectionNodeID string

	mockCollectionsStore := mocks.NewMockCollectionsStore().WithCreateCollectionsFunc(func(_ context.Context, userID int64, nodeID, name, description string, dois []string) (store.CreateCollectionResponse, error) {
		t.Helper()
		require.Equal(t, callingUser.ID, userID)
		require.NotEmpty(t, nodeID)
		collectionNodeID = nodeID
		require.Equal(t, createCollectionRequest.Name, name)
		require.Equal(t, createCollectionRequest.Description, description)
		require.Equal(t, createCollectionRequest.DOIs, dois)
		return store.CreateCollectionResponse{
			ID:          1,
			CreatorRole: role.Owner,
		}, nil
	})

	handler := CollectionsServiceAPIHandler(
		apitest.NewTestContainer().WithDiscover(mockDiscoverService).WithCollectionsStore(mockCollectionsStore),
		apitest.NewConfigBuilder().
			WithPennsieveConfig(apitest.PennsieveConfigWithFakeHost()).
			Build())

	req := test.NewAPIGatewayRequestBuilder("POST /collections").
		WithDefaultClaims(callingUser).
		WithBody(t, createCollectionRequest).
		Build()

	response, err := handler(context.Background(), req)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusCreated, response.StatusCode)

	var responseDTO dto.CollectionResponse
	require.NoError(t, json.Unmarshal([]byte(response.Body), &responseDTO))

	assert.Equal(t, collectionNodeID, responseDTO.NodeID)
	assert.Equal(t, createCollectionRequest.Name, responseDTO.Name)
	assert.Equal(t, createCollectionRequest.Description, responseDTO.Description)
	assert.Equal(t, len(createCollectionRequest.DOIs), responseDTO.Size)
	assert.Equal(t, role.Owner.String(), responseDTO.UserRole)
	assert.Equal(t, []string{*banner1, *banner2}, responseDTO.Banners)
}
