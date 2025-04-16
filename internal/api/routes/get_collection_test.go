package routes

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/collections-service/internal/dbmigrate"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/pennsieve/collections-service/internal/test/configtest"
	"github.com/pennsieve/collections-service/internal/test/dbmigratetest"
	"github.com/pennsieve/collections-service/internal/test/fixtures"
	"github.com/pennsieve/collections-service/internal/test/mocks"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetCollection(t *testing.T) {
	tests := []struct {
		scenario string
		tstFunc  func(t *testing.T, expectationDB *fixtures.ExpectationDB)
	}{
		{"get collection, none", testGetCollectionNone},
		{"get collection", testGetCollection},
	}

	ctx := context.Background()
	postgresDBConfig := configtest.PostgresDBConfig()
	migrator, err := dbmigrate.NewLocalCollectionsMigrator(ctx, dbmigrate.Config{
		PostgresDB:     postgresDBConfig,
		VerboseLogging: true,
	})
	t.Cleanup(func() {
		dbmigratetest.Close(t, migrator)
	})
	require.NoError(t, err)
	require.NoError(t, migrator.Up())

	for _, tt := range tests {
		t.Run(tt.scenario, func(t *testing.T) {
			db := test.NewPostgresDBFromConfig(t, postgresDBConfig)

			t.Cleanup(func() {
				require.NoError(t, fixtures.TruncateCollectionsSchema(ctx, t, db, postgresDBConfig.CollectionsDatabase))
			})

			tt.tstFunc(t, fixtures.NewExpectationDB(db, postgresDBConfig.CollectionsDatabase))
		})
	}
}

func testGetCollectionNone(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	// Set up using the ExpectationDB

	user2ExpectedCollection := apitest.NewExpectedCollection().WithNodeID().WithUser(apitest.User2.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI())
	expectationDB.CreateCollection(ctx, t, user2ExpectedCollection)

	// Test route
	// use a different user with no collections
	callingUser := apitest.User
	nonExistentNodeID := uuid.NewString()

	claims := apitest.DefaultClaims(callingUser)

	apiConfig := apitest.NewConfigBuilder().
		WithDockerPostgresDBConfig().
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
		WithContainerStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(GetCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, nonExistentNodeID).
			Build(),
		Container: container,
		Config:    apiConfig,
		Claims:    &claims,
	}
	_, err := GetCollection(ctx, params)
	require.Error(t, err)

	var apiError *apierrors.Error
	if assert.ErrorAs(t, err, &apiError) {
		assert.Equal(t, http.StatusNotFound, apiError.StatusCode)
		assert.Contains(t, apiError.UserMessage, nonExistentNodeID)
	}

}

func testGetCollection(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	user1 := apitest.User
	user2 := apitest.User2

	expectedDatasets := apitest.NewExpectedPennsieveDatasets()

	// Set up using the ExpectationDB
	user1CollectionNoDOI := apitest.NewExpectedCollection().WithNodeID().WithUser(user1.ID, pgdb.Owner)
	expectationDB.CreateCollection(ctx, t, user1CollectionNoDOI)

	user1CollectionOneDOI := apitest.NewExpectedCollection().WithNodeID().WithUser(user1.ID, pgdb.Owner).
		WithDOIs(expectedDatasets.NewPublished(apitest.NewPublicContributor()).DOI)
	expectationDB.CreateCollection(ctx, t, user1CollectionOneDOI)

	user1CollectionFiveDOI := apitest.NewExpectedCollection().WithNodeID().WithUser(user1.ID, pgdb.Owner).
		WithDOIs(
			expectedDatasets.NewPublished(apitest.NewPublicContributor()).DOI,
			expectedDatasets.NewPublished(apitest.NewPublicContributor(apitest.WithMiddleInitial()), apitest.NewPublicContributor(apitest.WithOrcid())).DOI,
			expectedDatasets.NewPublished(apitest.NewPublicContributor(apitest.WithOrcid())).DOI,
			expectedDatasets.NewPublished(apitest.NewPublicContributor(apitest.WithOrcid()), apitest.NewPublicContributor(apitest.WithOrcid(), apitest.WithDegree())).DOI,
			expectedDatasets.NewPublished(apitest.NewPublicContributor(apitest.WithOrcid())).DOI,
		)
	expectationDB.CreateCollection(ctx, t, user1CollectionFiveDOI)

	user2Collection := apitest.NewExpectedCollection().WithNodeID().WithUser(user2.ID, pgdb.Owner).
		WithDOIs(expectedDatasets.NewPublished(apitest.NewPublicContributor()).DOI, expectedDatasets.NewPublished(apitest.NewPublicContributor(apitest.WithMiddleInitial(), apitest.WithDegree(), apitest.WithOrcid())).DOI)
	expectationDB.CreateCollection(ctx, t, user2Collection)

	mockDiscoverServer := httptest.NewServer(mocks.ToDiscoverHandlerFunc(t, expectedDatasets.GetDatasetsByDOIFunc(t)))
	defer mockDiscoverServer.Close()

	user1Claims := apitest.DefaultClaims(user1)
	user2Claims := apitest.DefaultClaims(user2)

	apiConfig := apitest.NewConfigBuilder().
		WithDockerPostgresDBConfig().
		WithPennsieveConfig(apitest.PennsieveConfig(mockDiscoverServer.URL)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
		WithContainerStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase).
		WithHTTPTestDiscover(mockDiscoverServer.URL)

	// user1NoDOIs
	paramsNoDOI := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(GetCollectionRouteKey).
			WithClaims(user1Claims).
			WithPathParam(NodeIDPathParamKey, *user1CollectionNoDOI.NodeID).
			Build(),
		Container: container,
		Config:    apiConfig,
		Claims:    &user1Claims,
	}
	user1NoDOIResp, err := GetCollection(ctx, paramsNoDOI)
	require.NoError(t, err)
	assert.NotNil(t, user1NoDOIResp)
	assertExpectedEqualCollectionResponse(t, user1CollectionNoDOI, user1NoDOIResp.CollectionResponse, expectedDatasets)
	assert.Empty(t, user1NoDOIResp.Datasets)
	assert.Empty(t, user1NoDOIResp.DerivedContributors)

	// user1OneDOI
	paramsOneDOI := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(GetCollectionRouteKey).
			WithClaims(user1Claims).
			WithPathParam(NodeIDPathParamKey, *user1CollectionOneDOI.NodeID).
			Build(),
		Container: container,
		Config:    apiConfig,
		Claims:    &user1Claims,
	}
	user1OneDOIResp, err := GetCollection(ctx, paramsOneDOI)
	assert.NoError(t, err)
	assert.NotNil(t, user1CollectionOneDOI)
	assertExpectedEqualCollectionResponse(t, user1CollectionOneDOI, user1OneDOIResp.CollectionResponse, expectedDatasets)
	assert.Len(t, user1OneDOIResp.Datasets, len(user1CollectionOneDOI.DOIs))
	for i := 0; i < len(user1CollectionOneDOI.DOIs); i++ {
		actualDataset := user1OneDOIResp.Datasets[i]
		expectedDOI := user1CollectionOneDOI.DOIs[i].DOI
		assert.False(t, actualDataset.Problem)
		require.Equal(t, dto.PennsieveSource, actualDataset.Source)
		var actualData dto.PublicDataset
		require.NoError(t, json.Unmarshal(actualDataset.Data, &actualData))
		assert.Equal(t, expectedDOI, actualData.DOI)
		var actualPublicDataset dto.PublicDataset
		apitest.RequireAsPennsieveDataset(t, actualDataset, &actualPublicDataset)
		assert.Equal(t, expectedDatasets.DOIToPublicDataset[expectedDOI], actualPublicDataset)
	}
	assert.Equal(t, expectedDatasets.ExpectedContributorsForDOIs(t, user1CollectionOneDOI.DOIs.Strings()), user1OneDOIResp.DerivedContributors)

	// user1FiveDOI
	paramsFiveDOI := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(GetCollectionRouteKey).
			WithClaims(user1Claims).
			WithPathParam(NodeIDPathParamKey, *user1CollectionFiveDOI.NodeID).
			Build(),
		Container: container,
		Config:    apiConfig,
		Claims:    &user1Claims,
	}
	user1FiveDOIResp, err := GetCollection(ctx, paramsFiveDOI)
	assert.NoError(t, err)
	assert.NotNil(t, user1CollectionFiveDOI)
	assertExpectedEqualCollectionResponse(t, user1CollectionFiveDOI, user1FiveDOIResp.CollectionResponse, expectedDatasets)
	assert.Len(t, user1FiveDOIResp.Datasets, len(user1CollectionFiveDOI.DOIs))
	for i := 0; i < len(user1CollectionFiveDOI.DOIs); i++ {
		actualDataset := user1FiveDOIResp.Datasets[i]
		expectedDOI := user1CollectionFiveDOI.DOIs[i].DOI
		assert.False(t, actualDataset.Problem)
		require.Equal(t, dto.PennsieveSource, actualDataset.Source)
		var actualData dto.PublicDataset
		require.NoError(t, json.Unmarshal(actualDataset.Data, &actualData))
		assert.Equal(t, expectedDOI, actualData.DOI)
		var actualPublicDataset dto.PublicDataset
		apitest.RequireAsPennsieveDataset(t, actualDataset, &actualPublicDataset)
		assert.Equal(t, expectedDatasets.DOIToPublicDataset[expectedDOI], actualPublicDataset)
	}
	// there should be no duplicates in the contributors since they contain UUIDs for any strings
	// So it's ok to use results straight from ExpectedContributorsForDOIs
	assert.Equal(t, expectedDatasets.ExpectedContributorsForDOIs(t, user1CollectionFiveDOI.DOIs.Strings()), user1FiveDOIResp.DerivedContributors)

	// try user2's collections
	paramsUser2 := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(GetCollectionRouteKey).
			WithClaims(user2Claims).
			WithPathParam(NodeIDPathParamKey, *user2Collection.NodeID).
			Build(),
		Container: container,
		Config:    apiConfig,
		Claims:    &user2Claims,
	}
	user2CollectionResp, err := GetCollection(ctx, paramsUser2)
	require.NoError(t, err)
	assert.NotNil(t, user2CollectionResp)
	assertExpectedEqualCollectionResponse(t, user2Collection, user2CollectionResp.CollectionResponse, expectedDatasets)
	assert.Len(t, user2CollectionResp.Datasets, len(user2Collection.DOIs))
	for i := 0; i < len(user2Collection.DOIs); i++ {
		actualDataset := user2CollectionResp.Datasets[i]
		expectedDOI := user2Collection.DOIs[i].DOI
		assert.False(t, actualDataset.Problem)
		require.Equal(t, dto.PennsieveSource, actualDataset.Source)
		var actualData dto.PublicDataset
		require.NoError(t, json.Unmarshal(actualDataset.Data, &actualData))
		assert.Equal(t, expectedDOI, actualData.DOI)
		var actualPublicDataset dto.PublicDataset
		apitest.RequireAsPennsieveDataset(t, actualDataset, &actualPublicDataset)
		assert.Equal(t, expectedDatasets.DOIToPublicDataset[expectedDOI], actualPublicDataset)
	}
	assert.Equal(t, expectedDatasets.ExpectedContributorsForDOIs(t, user2Collection.DOIs.Strings()), user2CollectionResp.DerivedContributors)

}

// TestHandleGetCollection tests that run the Handle wrapper around GetCollection
func TestHandleGetCollection(t *testing.T) {
	tests := []struct {
		name    string
		tstFunc func(t *testing.T)
	}{
		{
			"return empty datasets and collaborators arrays instead of null",
			testHandleGetCollectionEmptyArrays,
		},
		{
			"return empty arrays in PublicDatasets instead of nulls",
			testHandleGetCollectionEmptyArraysInPublicDataset,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.tstFunc(t)
		})
	}
}

func testHandleGetCollectionEmptyArrays(t *testing.T) {
	ctx := context.Background()
	callingUser := apitest.User

	expectedCollection := apitest.NewExpectedCollection().WithNodeID().WithUser(callingUser.ID, pgdb.Owner)

	mockCollectionStore := mocks.NewMockCollectionsStore().
		WithGetCollectionFunc(expectedCollection.GetCollectionFunc(t))

	claims := apitest.DefaultClaims(callingUser)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(GetCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
			Build(),
		Container: apitest.NewTestContainer().WithCollectionsStore(mockCollectionStore),
		Config:    apitest.NewConfigBuilder().WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).Build(),
		Claims:    &claims,
	}
	response, err := Handle(ctx, NewGetCollectionRouteHandler(), params)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, response.StatusCode)

	assert.NotContains(t, response.Body, `"banners":null`)
	assert.Contains(t, response.Body, `"banners":[]`)

	assert.NotContains(t, response.Body, `"derivedContributors":null`)
	assert.Contains(t, response.Body, `"derivedContributors":[]`)

	assert.NotContains(t, response.Body, `"datasets":null`)
	assert.Contains(t, response.Body, `"datasets":[]`)

}

func testHandleGetCollectionEmptyArraysInPublicDataset(t *testing.T) {
	ctx := context.Background()
	callingUser := apitest.User

	expectedDOI := apitest.NewPennsieveDOI()
	expectedCollection := apitest.NewExpectedCollection().WithNodeID().WithUser(callingUser.ID, pgdb.Owner).WithDOIs(expectedDOI)

	mockCollectionStore := mocks.NewMockCollectionsStore().
		WithGetCollectionFunc(expectedCollection.GetCollectionFunc(t))

	mockDiscover := mocks.NewMockDiscover().WithGetDatasetsByDOIFunc(func(dois []string) (service.DatasetsByDOIResponse, error) {
		return service.DatasetsByDOIResponse{Published: map[string]dto.PublicDataset{
			expectedDOI: apitest.NewPublicDataset(expectedDOI, apitest.NewBanner(), apitest.NewPublicContributor()),
		}}, nil
	})
	claims := apitest.DefaultClaims(callingUser)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(GetCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
			Build(),
		Container: apitest.NewTestContainer().WithCollectionsStore(mockCollectionStore).WithDiscover(mockDiscover),
		Config:    apitest.NewConfigBuilder().WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).Build(),
		Claims:    &claims,
	}
	response, err := Handle(ctx, NewGetCollectionRouteHandler(), params)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, response.StatusCode)

	assert.NotContains(t, response.Body, `"modelCount":null`)
	assert.Contains(t, response.Body, `"modelCount":[]`)

}
