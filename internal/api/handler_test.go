package api

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/datasource"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/routes"
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/collections-service/internal/api/store"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/pennsieve/collections-service/internal/test/mocks"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"strings"
	"testing"
)

func TestAPILambdaHandler(t *testing.T) {
	tests := []struct {
		scenario string
		fn       func(t *testing.T)
	}{
		{"default not found response", testDefaultNotFound},
		{"no claims", testNoClaims},
		{"create collection bad request: external DOIs", testCreateCollectionExternalDOIs},
		{"create collection bad request: empty name", testCreateCollectionEmptyName},
		{"create collection bad request: name too long", testCreateCollectionNameTooLong},
		{"create collection bad request: description too long", testCreateCollectionDescriptionTooLong},
		{"create collection bad request: unpublished DOIs", testCreateCollectionUnpublishedDOIs},
		{"create collection bad request: no body", testCreateCollectionNoBody},
		{"create collection bad request: malformed body", testCreateCollectionMalformedBody},
		{"create collection", testCreateCollection},
		{"get collections", testGetCollections},
		{"get collection", testGetCollection},
		{"delete collection", testDeleteCollection},
		{"update collection", testUpdateCollection},
	}
	for _, tt := range tests {
		t.Run(tt.scenario, func(t *testing.T) {
			tt.fn(t)
		})
	}
}

func testDefaultNotFound(t *testing.T) {
	handler := CollectionsServiceAPIHandler(apitest.NewTestContainer(), apitest.NewConfigBuilder().Build())

	req := apitest.NewAPIGatewayRequestBuilder("GET /unknown").
		WithDefaultClaims(apitest.SeedUser2).
		Build()

	response, err := handler(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, response.StatusCode)
	assert.Contains(t, response.Body, "not found")
	assert.Contains(t, response.Body, `"errorId"`)
}

func testNoClaims(t *testing.T) {
	handler := CollectionsServiceAPIHandler(apitest.NewTestContainer(), apitest.NewConfigBuilder().Build())

	req := apitest.NewAPIGatewayRequestBuilder(routes.CreateCollectionRouteKey).Build()

	response, err := handler(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

func testCreateCollectionExternalDOIs(t *testing.T) {
	createCollectionRequest := dto.CreateCollectionRequest{
		Name:        uuid.NewString(),
		Description: uuid.NewString(),
		DOIs:        []string{apitest.NewExternalDOI().Value, apitest.NewPennsieveDOI().Value, apitest.NewExternalDOI().Value},
	}

	handler := CollectionsServiceAPIHandler(
		apitest.NewTestContainer(),
		apitest.NewConfigBuilder().
			WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).
			Build())

	req := apitest.NewAPIGatewayRequestBuilder(routes.CreateCollectionRouteKey).
		WithDefaultClaims(apitest.SeedUser1).
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
			WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).
			Build())

	req := apitest.NewAPIGatewayRequestBuilder(routes.CreateCollectionRouteKey).
		WithDefaultClaims(apitest.SeedUser1).
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
			WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).
			Build())

	req := apitest.NewAPIGatewayRequestBuilder(routes.CreateCollectionRouteKey).
		WithDefaultClaims(apitest.SeedUser1).
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
			WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).
			Build())

	req := apitest.NewAPIGatewayRequestBuilder(routes.CreateCollectionRouteKey).
		WithDefaultClaims(apitest.SeedUser1).
		WithBody(t, createCollectionRequest).
		Build()

	response, err := handler(context.Background(), req)

	assert.NoError(t, err)
	assert.Contains(t, response.Body, "collection description cannot have more than 255 characters")
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func testCreateCollectionUnpublishedDOIs(t *testing.T) {
	publishedDOI := apitest.NewPennsieveDOI()
	unpublishedDOI := apitest.NewPennsieveDOI()
	expectedStatus := "PublishFailed"
	createCollectionRequest := dto.CreateCollectionRequest{
		Name:        uuid.NewString(),
		Description: uuid.NewString(),
		DOIs:        []string{publishedDOI.Value, unpublishedDOI.Value},
	}

	mockDiscoverService := mocks.NewMockDiscover().
		WithGetDatasetsByDOIFunc(func(dois []string) (service.DatasetsByDOIResponse, error) {
			test.Helper(t)
			require.Equal(t, []string{publishedDOI.Value, unpublishedDOI.Value}, dois)
			return service.DatasetsByDOIResponse{
				Published:   map[string]dto.PublicDataset{publishedDOI.Value: apitest.NewPublicDataset(publishedDOI.Value, nil)},
				Unpublished: map[string]dto.Tombstone{unpublishedDOI.Value: apitest.NewTombstone(unpublishedDOI.Value, expectedStatus)},
			}, nil
		})

	handler := CollectionsServiceAPIHandler(
		apitest.NewTestContainer().WithDiscover(mockDiscoverService),
		apitest.NewConfigBuilder().
			WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).
			Build())

	req := apitest.NewAPIGatewayRequestBuilder(routes.CreateCollectionRouteKey).
		WithDefaultClaims(apitest.SeedUser1).
		WithBody(t, createCollectionRequest).
		Build()

	response, err := handler(context.Background(), req)

	assert.NoError(t, err)
	assert.Contains(t, response.Body, unpublishedDOI.Value)
	assert.Contains(t, response.Body, expectedStatus)
	assert.NotContains(t, response.Body, publishedDOI)
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)

}

func testCreateCollectionNoBody(t *testing.T) {
	handler := CollectionsServiceAPIHandler(
		apitest.NewTestContainer(),
		apitest.NewConfigBuilder().Build())

	req := apitest.NewAPIGatewayRequestBuilder(routes.CreateCollectionRouteKey).WithDefaultClaims(apitest.SeedUser1).Build()

	response, err := handler(context.Background(), req)
	require.NoError(t, err)

	assert.Contains(t, response.Body, "missing request body")
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)

}

func testCreateCollectionMalformedBody(t *testing.T) {
	handler := CollectionsServiceAPIHandler(
		apitest.NewTestContainer(),
		apitest.NewConfigBuilder().Build())

	req := apitest.NewAPIGatewayRequestBuilder(routes.CreateCollectionRouteKey).
		WithDefaultClaims(apitest.SeedUser1).
		Build()

	req.Body = "{]"

	response, err := handler(context.Background(), req)
	require.NoError(t, err)

	assert.Contains(t, response.Body, "error unmarshalling request body")
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)

}

func testCreateCollection(t *testing.T) {
	publishedDOI1 := apitest.NewPennsieveDOI()
	banner1 := apitest.NewBanner()

	publishedDOI2 := apitest.NewPennsieveDOI()
	banner2 := apitest.NewBanner()

	callingUser := apitest.SeedUser1

	createCollectionRequest := dto.CreateCollectionRequest{
		Name:        uuid.NewString(),
		Description: uuid.NewString(),
		DOIs:        []string{publishedDOI1.Value, publishedDOI2.Value},
	}

	mockDiscoverService := mocks.NewMockDiscover().
		WithGetDatasetsByDOIFunc(func(dois []string) (service.DatasetsByDOIResponse, error) {
			t.Helper()
			require.Equal(t, []string{publishedDOI1.Value, publishedDOI2.Value}, dois)
			return service.DatasetsByDOIResponse{
				Published: map[string]dto.PublicDataset{
					publishedDOI1.Value: apitest.NewPublicDataset(publishedDOI1.Value, banner1),
					publishedDOI2.Value: apitest.NewPublicDataset(publishedDOI2.Value, banner2)},
			}, nil
		})

	var collectionNodeID string

	mockCollectionsStore := mocks.NewMockCollectionsStore().WithCreateCollectionsFunc(func(_ context.Context, userID int64, nodeID, name, description string, dois []store.DOI) (store.CreateCollectionResponse, error) {
		t.Helper()
		require.Equal(t, callingUser.ID, userID)
		require.NotEmpty(t, nodeID)
		collectionNodeID = nodeID
		require.Equal(t, createCollectionRequest.Name, name)
		require.Equal(t, createCollectionRequest.Description, description)
		var expectedDOIs []store.DOI
		for _, doi := range createCollectionRequest.DOIs {
			expectedDOIs = append(expectedDOIs, store.DOI{
				Value:      doi,
				Datasource: datasource.Pennsieve,
			})
		}
		require.Equal(t, expectedDOIs, dois)
		return store.CreateCollectionResponse{
			ID:          1,
			CreatorRole: role.Owner,
		}, nil
	})

	handler := CollectionsServiceAPIHandler(
		apitest.NewTestContainer().WithDiscover(mockDiscoverService).WithCollectionsStore(mockCollectionsStore),
		apitest.NewConfigBuilder().
			WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).
			Build())

	req := apitest.NewAPIGatewayRequestBuilder(routes.CreateCollectionRouteKey).
		WithDefaultClaims(callingUser).
		WithBody(t, createCollectionRequest).
		Build()

	response, err := handler(context.Background(), req)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusCreated, response.StatusCode)

	var responseDTO dto.CollectionSummary
	require.NoError(t, json.Unmarshal([]byte(response.Body), &responseDTO))

	assert.Equal(t, collectionNodeID, responseDTO.NodeID)
	assert.Equal(t, createCollectionRequest.Name, responseDTO.Name)
	assert.Equal(t, createCollectionRequest.Description, responseDTO.Description)
	assert.Equal(t, len(createCollectionRequest.DOIs), responseDTO.Size)
	assert.Equal(t, role.Owner.String(), responseDTO.UserRole)
	assert.Equal(t, []string{*banner1, *banner2}, responseDTO.Banners)
}

func testGetCollections(t *testing.T) {
	callingUser := apitest.SeedUser1

	expectedDatasets := apitest.NewExpectedPennsieveDatasets()
	expectedDataset := expectedDatasets.NewPublished()

	expectedOffset := 100

	expectedBanner := expectedDataset.Banner

	expectedCollection := apitest.NewExpectedCollection().WithNodeID().WithUser(callingUser.ID, pgdb.Owner).WithPublicDatasets(expectedDataset)

	mockCollectionStore := mocks.NewMockCollectionsStore().
		WithGetCollectionsFunc(func(ctx context.Context, userID int64, limit int, offset int) (store.GetCollectionsResponse, error) {
			require.Equal(t, callingUser.ID, userID)
			require.Equal(t, routes.DefaultGetCollectionsLimit, limit)
			require.Equal(t, expectedOffset, offset)
			return store.GetCollectionsResponse{
				Limit:      routes.DefaultGetCollectionsLimit,
				Offset:     expectedOffset,
				TotalCount: 101,
				Collections: []store.CollectionSummary{{
					CollectionBase: store.CollectionBase{
						NodeID:      *expectedCollection.NodeID,
						Name:        expectedCollection.Name,
						Description: expectedCollection.Description,
						Size:        len(expectedCollection.DOIs),
						UserRole:    expectedCollection.Users[0].PermissionBit.ToRole(),
					},
					BannerDOIs: []string{expectedDataset.DOI},
				}},
			}, nil
		})

	mockDiscoverService := mocks.NewMockDiscover().WithGetDatasetsByDOIFunc(expectedDatasets.GetDatasetsByDOIFunc(t))

	handler := CollectionsServiceAPIHandler(
		apitest.NewTestContainer().
			WithCollectionsStore(mockCollectionStore).
			WithDiscover(mockDiscoverService),
		apitest.NewConfigBuilder().
			WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).
			Build(),
	)
	req := apitest.NewAPIGatewayRequestBuilder(routes.GetCollectionsRouteKey).
		WithDefaultClaims(callingUser).
		WithIntQueryParam("offset", expectedOffset).
		Build()

	response, err := handler(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, response.StatusCode)

	var responseDTO dto.GetCollectionsResponse
	require.NoError(t, json.Unmarshal([]byte(response.Body), &responseDTO))

	assert.Equal(t, routes.DefaultGetCollectionsLimit, responseDTO.Limit)
	assert.Equal(t, expectedOffset, responseDTO.Offset)
	assert.Equal(t, 101, responseDTO.TotalCount)

	assert.Len(t, responseDTO.Collections, 1)
	actualCollection := responseDTO.Collections[0]
	assert.Equal(t, expectedCollection.Name, actualCollection.Name)
	assert.Equal(t, *expectedCollection.NodeID, actualCollection.NodeID)
	assert.Equal(t, expectedCollection.Description, actualCollection.Description)
	assert.Equal(t, len(expectedCollection.DOIs), actualCollection.Size)
	assert.Equal(t, expectedCollection.Users[0].PermissionBit.ToRole().String(), actualCollection.UserRole)
	assert.Equal(t, []string{*expectedBanner}, actualCollection.Banners)

}

func testGetCollection(t *testing.T) {
	callingUser := apitest.SeedUser1

	expectedDatasets := apitest.NewExpectedPennsieveDatasets()
	expectedDataset := expectedDatasets.NewPublished(apitest.NewPublicContributor(), apitest.NewPublicContributor(apitest.WithOrcid()))

	expectedCollection := apitest.NewExpectedCollection().WithRandomID().WithNodeID().WithUser(callingUser.ID, pgdb.Owner).WithPublicDatasets(expectedDataset)

	mockCollectionStore := mocks.NewMockCollectionsStore().
		WithGetCollectionFunc(expectedCollection.GetCollectionFunc(t))

	mockDiscoverService := mocks.NewMockDiscover().WithGetDatasetsByDOIFunc(expectedDatasets.GetDatasetsByDOIFunc(t))

	handler := CollectionsServiceAPIHandler(
		apitest.NewTestContainer().
			WithCollectionsStore(mockCollectionStore).
			WithDiscover(mockDiscoverService),
		apitest.NewConfigBuilder().
			WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).
			Build(),
	)
	req := apitest.NewAPIGatewayRequestBuilder(routes.GetCollectionRouteKey).
		WithDefaultClaims(callingUser).
		WithPathParam(routes.NodeIDPathParamKey, *expectedCollection.NodeID).
		Build()

	response, err := handler(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, response.StatusCode)

	var responseDTO dto.GetCollectionResponse
	require.NoError(t, json.Unmarshal([]byte(response.Body), &responseDTO))

	assert.Equal(t, expectedCollection.Name, responseDTO.Name)
	assert.Equal(t, *expectedCollection.NodeID, responseDTO.NodeID)
	assert.Equal(t, expectedCollection.Description, responseDTO.Description)
	assert.Equal(t, len(expectedCollection.DOIs), responseDTO.Size)
	assert.Equal(t, role.Owner.String(), responseDTO.UserRole)
	assert.Equal(t, []string{*expectedDataset.Banner}, responseDTO.Banners)

	assert.Equal(t, expectedDataset.Contributors, responseDTO.DerivedContributors)

	require.Len(t, responseDTO.Datasets, 1)
	actualDataset := responseDTO.Datasets[0]
	var actualPennsieveDataset dto.PublicDataset
	apitest.RequireAsPennsieveDataset(t, actualDataset, &actualPennsieveDataset)
	assert.Equal(t, expectedDataset, actualPennsieveDataset)
}

func testDeleteCollection(t *testing.T) {
	callingUser := apitest.SeedUser1

	expectedDatasets := apitest.NewExpectedPennsieveDatasets()
	expectedDataset := expectedDatasets.NewPublished(apitest.NewPublicContributor(), apitest.NewPublicContributor(apitest.WithOrcid()))

	expectedCollection := apitest.NewExpectedCollection().WithRandomID().WithNodeID().WithUser(callingUser.ID, pgdb.Owner).WithPublicDatasets(expectedDataset)

	mockCollectionStore := mocks.NewMockCollectionsStore().
		WithGetCollectionFunc(expectedCollection.GetCollectionFunc(t)).
		WithDeleteCollectionFunc(func(ctx context.Context, collectionID int64) error {
			require.Equal(t, *expectedCollection.ID, collectionID)
			return nil
		})

	handler := CollectionsServiceAPIHandler(
		apitest.NewTestContainer().
			WithCollectionsStore(mockCollectionStore),
		apitest.NewConfigBuilder().
			WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).
			Build(),
	)
	req := apitest.NewAPIGatewayRequestBuilder(routes.DeleteCollectionRouteKey).
		WithDefaultClaims(callingUser).
		WithPathParam(routes.NodeIDPathParamKey, *expectedCollection.NodeID).
		Build()

	response, err := handler(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusNoContent, response.StatusCode)
	assert.Empty(t, response.Body)

}

func testUpdateCollection(t *testing.T) {
	callingUser := apitest.SeedUser1

	expectedDatasets := apitest.NewExpectedPennsieveDatasets()
	datasetToRemove := expectedDatasets.NewPublished(apitest.NewPublicContributor(), apitest.NewPublicContributor(apitest.WithOrcid()))
	datasetToAdd := expectedDatasets.NewPublished(apitest.NewPublicContributor(apitest.WithOrcid()))

	expectedCollection := apitest.NewExpectedCollection().WithRandomID().WithNodeID().WithUser(callingUser.ID, pgdb.Owner).WithPublicDatasets(datasetToRemove)

	mockCollectionStore := mocks.NewMockCollectionsStore().
		WithGetCollectionFunc(expectedCollection.GetCollectionFunc(t)).
		WithUpdateCollectionFunc(expectedCollection.UpdateCollectionFunc(t))

	newName := uuid.NewString()
	newDescription := uuid.NewString()
	update := dto.PatchCollectionRequest{
		Name:        &newName,
		Description: &newDescription,
		DOIs: &dto.PatchDOIs{
			Remove: []string{datasetToRemove.DOI},
			Add:    []string{datasetToAdd.DOI},
		},
	}

	handler := CollectionsServiceAPIHandler(
		apitest.NewTestContainer().
			WithCollectionsStore(mockCollectionStore).
			WithDiscover(mocks.NewMockDiscover().WithGetDatasetsByDOIFunc(expectedDatasets.GetDatasetsByDOIFunc(t))),
		apitest.NewConfigBuilder().
			WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).
			Build(),
	)
	req := apitest.NewAPIGatewayRequestBuilder(routes.PatchCollectionRouteKey).
		WithDefaultClaims(callingUser).
		WithPathParam(routes.NodeIDPathParamKey, *expectedCollection.NodeID).
		WithBody(t, update).
		Build()

	response, err := handler(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, response.StatusCode)

	var responseDTO dto.GetCollectionResponse
	require.NoError(t, json.Unmarshal([]byte(response.Body), &responseDTO))

	assert.Equal(t, newName, responseDTO.Name)
	assert.Equal(t, *expectedCollection.NodeID, responseDTO.NodeID)
	assert.Equal(t, newDescription, responseDTO.Description)
	assert.Equal(t, 1, responseDTO.Size)
	assert.Equal(t, role.Owner.String(), responseDTO.UserRole)
	assert.Equal(t, []string{*datasetToAdd.Banner}, responseDTO.Banners)

	assert.Equal(t, datasetToAdd.Contributors, responseDTO.DerivedContributors)

	require.Len(t, responseDTO.Datasets, 1)
	actualDataset := responseDTO.Datasets[0]
	var actualPennsieveDataset dto.PublicDataset
	apitest.RequireAsPennsieveDataset(t, actualDataset, &actualPennsieveDataset)
	assert.Equal(t, datasetToAdd, actualPennsieveDataset)

}
