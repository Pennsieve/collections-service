package routes

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/collections-service/internal/api/store"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/pennsieve/collections-service/internal/test/fixtures"
	"github.com/pennsieve/collections-service/internal/test/mocks"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateCollection(t *testing.T) {
	ctx := context.Background()
	config := test.PostgresDBConfig(t)

	for scenario, tstFunc := range map[string]func(t *testing.T, expectationDB *fixtures.ExpectationDB){
		"create collection; no DTOs":              testCreateCollectionNoDTOs,
		"create collection; two DTOs":             testCreateCollectionTwoDTOs,
		"create collection; five DTOs":            testCreateCollectionFiveDTOs,
		"create collection; some missing banners": testCreateCollectionSomeMissingBanners,
		"create collection; remove whitespace":    testCreateCollectionRemoveWhitespace,
	} {
		t.Run(scenario, func(t *testing.T) {
			db := test.NewPostgresDBFromConfig(t, config)
			expectationDB := fixtures.NewExpectationDB(db, config.CollectionsDatabase)

			t.Cleanup(func() {
				expectationDB.CleanUp(ctx, t)
			})

			tstFunc(t, expectationDB)
		})
	}

}

func testCreateCollectionNoDTOs(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	callingUser := apitest.SeedUser1

	expectedCollection := apitest.NewExpectedCollection().
		WithUser(callingUser.ID, pgdb.Owner)

	createCollectionRequest := dto.CreateCollectionRequest{
		Name:        expectedCollection.Name,
		Description: expectedCollection.Description,
	}

	claims := apitest.DefaultClaims(callingUser)

	config := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, config.PostgresDB)).
		WithContainerStoreFromPostgresDB(config.PostgresDB.CollectionsDatabase)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(CreateCollectionRouteKey).
			WithClaims(claims).
			WithBody(t, createCollectionRequest).
			Build(),
		Container: container,
		Config:    config,
		Claims:    &claims,
	}

	response, err := CreateCollection(ctx, params)
	require.NoError(t, err)

	assert.NotEmpty(t, t, response.NodeID)
	assert.Equal(t, createCollectionRequest.Name, response.Name)
	assert.Equal(t, createCollectionRequest.Description, response.Description)
	assert.Zero(t, response.Size)
	assert.Equal(t, role.Owner.String(), response.UserRole)

	expectationDB.RequireCollectionByNodeID(ctx, t, expectedCollection, response.NodeID)
}

func testCreateCollectionTwoDTOs(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	callingUser := apitest.SeedUser1

	publishedDOI1 := apitest.NewPennsieveDOI()
	banner1 := apitest.NewBanner()

	publishedDOI2 := apitest.NewPennsieveDOI()
	banner2 := apitest.NewBanner()

	expectedCollection := apitest.NewExpectedCollection().
		WithUser(callingUser.ID, pgdb.Owner).
		WithDOIs(publishedDOI1, publishedDOI2)

	createCollectionRequest := dto.CreateCollectionRequest{
		Name:        expectedCollection.Name,
		Description: expectedCollection.Description,
		DOIs:        expectedCollection.DOIs.Strings(),
	}

	mockDiscoverServer := httptest.NewServer(mocks.ToDiscoverHandlerFunc(t, func(dois []string) (service.DatasetsByDOIResponse, error) {
		t.Helper()
		require.Equal(t, []string{publishedDOI1, publishedDOI2}, dois)
		return service.DatasetsByDOIResponse{
			Published: map[string]dto.PublicDataset{
				publishedDOI1: apitest.NewPublicDataset(publishedDOI1, banner1),
				publishedDOI2: apitest.NewPublicDataset(publishedDOI2, banner2)},
		}, nil
	}))
	defer mockDiscoverServer.Close()

	claims := apitest.DefaultClaims(callingUser)

	config := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(apitest.PennsieveConfig(mockDiscoverServer.URL)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, config.PostgresDB)).
		WithHTTPTestDiscover(mockDiscoverServer.URL).
		WithContainerStoreFromPostgresDB(config.PostgresDB.CollectionsDatabase)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(CreateCollectionRouteKey).
			WithClaims(claims).
			WithBody(t, createCollectionRequest).
			Build(),
		Container: container,
		Config:    config,
		Claims:    &claims,
	}

	response, err := CreateCollection(ctx, params)
	require.NoError(t, err)

	assert.NotEmpty(t, t, response.NodeID)
	assert.Equal(t, createCollectionRequest.Name, response.Name)
	assert.Equal(t, createCollectionRequest.Description, response.Description)
	assert.Equal(t, len(createCollectionRequest.DOIs), response.Size)
	assert.Equal(t, []string{*banner1, *banner2}, response.Banners)
	assert.Equal(t, role.Owner.String(), response.UserRole)

	expectationDB.RequireCollectionByNodeID(ctx, t, expectedCollection, response.NodeID)
}

func testCreateCollectionFiveDTOs(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	callingUser := apitest.SeedUser1

	publishedDOI1 := apitest.NewPennsieveDOI()
	banner1 := apitest.NewBanner()

	publishedDOI2 := apitest.NewPennsieveDOI()
	banner2 := apitest.NewBanner()

	publishedDOI3 := apitest.NewPennsieveDOI()
	banner3 := apitest.NewBanner()

	publishedDTO4 := apitest.NewPennsieveDOI()
	banner4 := apitest.NewBanner()

	publishedDTO5 := apitest.NewPennsieveDOI()
	banner5 := apitest.NewBanner()

	expectedCollection := apitest.NewExpectedCollection().
		WithUser(callingUser.ID, pgdb.Owner).
		WithDOIs(publishedDOI1, publishedDOI2, publishedDOI3, publishedDTO4, publishedDTO5)

	createCollectionRequest := dto.CreateCollectionRequest{
		Name:        expectedCollection.Name,
		Description: expectedCollection.Description,
		DOIs:        expectedCollection.DOIs.Strings(),
	}

	mockDiscoverServer := httptest.NewServer(mocks.ToDiscoverHandlerFunc(t, func(dois []string) (service.DatasetsByDOIResponse, error) {
		t.Helper()
		require.Equal(t, []string{publishedDOI1, publishedDOI2, publishedDOI3, publishedDTO4, publishedDTO5}, dois)
		return service.DatasetsByDOIResponse{
			Published: map[string]dto.PublicDataset{
				publishedDOI1: apitest.NewPublicDataset(publishedDOI1, banner1),
				publishedDOI2: apitest.NewPublicDataset(publishedDOI2, banner2),
				publishedDOI3: apitest.NewPublicDataset(publishedDOI3, banner3),
				publishedDTO4: apitest.NewPublicDataset(publishedDTO4, banner4),
				publishedDTO5: apitest.NewPublicDataset(publishedDTO5, banner5),
			},
		}, nil
	}))
	defer mockDiscoverServer.Close()

	claims := apitest.DefaultClaims(callingUser)

	config := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(apitest.PennsieveConfig(mockDiscoverServer.URL)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, config.PostgresDB)).
		WithHTTPTestDiscover(mockDiscoverServer.URL).
		WithContainerStoreFromPostgresDB(config.PostgresDB.CollectionsDatabase)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(CreateCollectionRouteKey).
			WithClaims(claims).
			WithBody(t, createCollectionRequest).
			Build(),
		Container: container,
		Config:    config,
		Claims:    &claims,
	}

	response, err := CreateCollection(ctx, params)
	require.NoError(t, err)

	assert.NotEmpty(t, t, response.NodeID)
	assert.Equal(t, createCollectionRequest.Name, response.Name)
	assert.Equal(t, createCollectionRequest.Description, response.Description)
	assert.Equal(t, len(createCollectionRequest.DOIs), response.Size)
	assert.Equal(t, []string{*banner1, *banner2, *banner3, *banner4}, response.Banners)
	assert.Equal(t, role.Owner.String(), response.UserRole)

	expectationDB.RequireCollectionByNodeID(ctx, t, expectedCollection, response.NodeID)

}

func testCreateCollectionSomeMissingBanners(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	callingUser := apitest.SeedUser1

	publishedDOI1 := apitest.NewPennsieveDOI()
	var banner1 *string = nil

	publishedDOI2 := apitest.NewPennsieveDOI()
	banner2 := apitest.NewBanner()

	publishedDOI3 := apitest.NewPennsieveDOI()
	var banner3 *string = nil

	publishedDTO4 := apitest.NewPennsieveDOI()
	banner4 := apitest.NewBanner()

	publishedDTO5 := apitest.NewPennsieveDOI()
	var banner5 *string = nil

	expectedCollection := apitest.NewExpectedCollection().
		WithUser(callingUser.ID, pgdb.Owner).
		WithDOIs(publishedDOI1, publishedDOI2, publishedDOI3, publishedDTO4, publishedDTO5)

	createCollectionRequest := dto.CreateCollectionRequest{
		Name:        expectedCollection.Name,
		Description: expectedCollection.Description,
		DOIs:        expectedCollection.DOIs.Strings(),
	}

	mockDiscoverServer := httptest.NewServer(mocks.ToDiscoverHandlerFunc(t, func(dois []string) (service.DatasetsByDOIResponse, error) {
		t.Helper()
		require.Equal(t, []string{publishedDOI1, publishedDOI2, publishedDOI3, publishedDTO4, publishedDTO5}, dois)
		return service.DatasetsByDOIResponse{
			Published: map[string]dto.PublicDataset{
				publishedDOI1: apitest.NewPublicDataset(publishedDOI1, banner1),
				publishedDOI2: apitest.NewPublicDataset(publishedDOI2, banner2),
				publishedDOI3: apitest.NewPublicDataset(publishedDOI3, banner3),
				publishedDTO4: apitest.NewPublicDataset(publishedDTO4, banner4),
				publishedDTO5: apitest.NewPublicDataset(publishedDTO5, banner5),
			},
		}, nil
	}))
	defer mockDiscoverServer.Close()

	claims := apitest.DefaultClaims(callingUser)

	config := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(apitest.PennsieveConfig(mockDiscoverServer.URL)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, config.PostgresDB)).
		WithHTTPTestDiscover(mockDiscoverServer.URL).
		WithContainerStoreFromPostgresDB(config.PostgresDB.CollectionsDatabase)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(CreateCollectionRouteKey).
			WithClaims(claims).
			WithBody(t, createCollectionRequest).
			Build(),
		Container: container,
		Config:    config,
		Claims:    &claims,
	}

	response, err := CreateCollection(ctx, params)
	require.NoError(t, err)

	assert.NotEmpty(t, t, response.NodeID)
	assert.Equal(t, createCollectionRequest.Name, response.Name)
	assert.Equal(t, createCollectionRequest.Description, response.Description)
	assert.Equal(t, len(createCollectionRequest.DOIs), response.Size)
	assert.Equal(t, []string{"", *banner2, "", *banner4}, response.Banners)
	assert.Equal(t, role.Owner.String(), response.UserRole)

	expectationDB.RequireCollectionByNodeID(ctx, t, expectedCollection, response.NodeID)

}

func testCreateCollectionRemoveWhitespace(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	callingUser := apitest.SeedUser1

	publishedDOI1 := apitest.NewPennsieveDOI()
	banner1 := apitest.NewBanner()

	publishedDOI2 := apitest.NewPennsieveDOI()
	banner2 := apitest.NewBanner()

	expectedCollection := apitest.NewExpectedCollection().
		WithUser(callingUser.ID, pgdb.Owner).
		WithDOIs(publishedDOI1, publishedDOI2)

	// Add some whitespace to vales in the create request.
	// Server should trim it off before creation and return the trimmed values.
	createCollectionRequest := dto.CreateCollectionRequest{
		Name:        fmt.Sprintf("   %s ", expectedCollection.Name),
		Description: fmt.Sprintf("%s  ", expectedCollection.Description),
		DOIs:        []string{fmt.Sprintf(" %s", publishedDOI1), fmt.Sprintf("%s ", publishedDOI2)},
	}

	mockDiscoverServer := httptest.NewServer(mocks.ToDiscoverHandlerFunc(t, func(dois []string) (service.DatasetsByDOIResponse, error) {
		t.Helper()
		require.Equal(t, []string{publishedDOI1, publishedDOI2}, dois)
		return service.DatasetsByDOIResponse{
			Published: map[string]dto.PublicDataset{
				publishedDOI1: apitest.NewPublicDataset(publishedDOI1, banner1),
				publishedDOI2: apitest.NewPublicDataset(publishedDOI2, banner2)},
		}, nil
	}))
	defer mockDiscoverServer.Close()

	claims := apitest.DefaultClaims(callingUser)

	config := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(apitest.PennsieveConfig(mockDiscoverServer.URL)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, config.PostgresDB)).
		WithHTTPTestDiscover(mockDiscoverServer.URL).
		WithContainerStoreFromPostgresDB(config.PostgresDB.CollectionsDatabase)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(CreateCollectionRouteKey).
			WithClaims(claims).
			WithBody(t, createCollectionRequest).
			Build(),
		Container: container,
		Config:    config,
		Claims:    &claims,
	}

	response, err := CreateCollection(ctx, params)
	require.NoError(t, err)

	assert.NotEmpty(t, t, response.NodeID)
	assert.Equal(t, expectedCollection.Name, response.Name)
	assert.Equal(t, expectedCollection.Description, response.Description)
	assert.Equal(t, len(expectedCollection.DOIs), response.Size)
	assert.Equal(t, []string{*banner1, *banner2}, response.Banners)
	assert.Equal(t, role.Owner.String(), response.UserRole)

	expectationDB.RequireCollectionByNodeID(ctx, t, expectedCollection, response.NodeID)

}

// TestHandleCreateCollection tests that run the Handle wrapper around CreateCollection
func TestHandleCreateCollection(t *testing.T) {
	tests := []struct {
		name    string
		tstFunc func(t *testing.T)
	}{
		{
			"return empty arrays instead of null",
			testHandleCreateCollectionEmptyBannerArray,
		},
		{
			"return Bad Request when given unknown fields", testRejectUnknownFields,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.tstFunc(t)
		})
	}
}

func testHandleCreateCollectionEmptyBannerArray(t *testing.T) {
	ctx := context.Background()

	callingUser := apitest.SeedUser1

	expectedCollection := apitest.NewExpectedCollection().
		WithUser(callingUser.ID, pgdb.Owner)

	createCollectionRequest := dto.CreateCollectionRequest{
		Name:        expectedCollection.Name,
		Description: expectedCollection.Description,
	}

	claims := apitest.DefaultClaims(callingUser)

	var collectionNodeID string

	mockCollectionsStore := mocks.NewMockCollectionsStore().WithCreateCollectionsFunc(func(_ context.Context, userID int64, nodeID, name, description string, dois []string) (store.CreateCollectionResponse, error) {
		t.Helper()
		collectionNodeID = nodeID
		return store.CreateCollectionResponse{
			ID:          1,
			CreatorRole: role.Owner,
		}, nil
	})

	config := apitest.NewConfigBuilder().
		WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).
		Build()

	container := apitest.NewTestContainer().
		WithCollectionsStore(mockCollectionsStore)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(CreateCollectionRouteKey).
			WithClaims(claims).
			WithBody(t, createCollectionRequest).
			Build(),
		Container: container,
		Config:    config,
		Claims:    &claims,
	}

	response, err := Handle(ctx, NewCreateCollectionRouteHandler(), params)
	require.NoError(t, err)

	assert.Equal(t, http.StatusCreated, response.StatusCode)

	assert.Contains(t, response.Body, collectionNodeID)

	// Want the banner url array to be empty and not null
	assert.NotContains(t, response.Body, "null")
	assert.Contains(t, response.Body, "[]")
}

func testRejectUnknownFields(t *testing.T) {
	ctx := context.Background()

	callingUser := apitest.SeedUser1

	unknownFieldName := uuid.NewString()
	badRequest := fmt.Sprintf(`{"name": "bad request collection", %q: ["test-doi"]}`, unknownFieldName)

	claims := apitest.DefaultClaims(callingUser)

	config := apitest.NewConfigBuilder().
		WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).
		Build()

	container := apitest.NewTestContainer()

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(CreateCollectionRouteKey).
			WithClaims(claims).
			WithBody(t, badRequest).
			Build(),
		Container: container,
		Config:    config,
		Claims:    &claims,
	}

	response, err := Handle(ctx, NewCreateCollectionRouteHandler(), params)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, response.StatusCode)

	assert.Contains(t, response.Body, unknownFieldName)
	assert.Contains(t, response.Body, "unknown field")
}
