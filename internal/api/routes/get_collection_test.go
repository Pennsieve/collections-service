package routes

import (
	"context"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/pennsieve/collections-service/internal/test/apitest/builders/stores/collectionstest"
	"github.com/pennsieve/collections-service/internal/test/fixtures"
	"github.com/pennsieve/collections-service/internal/test/mocks"
	"github.com/pennsieve/collections-service/internal/test/userstest"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetCollection(t *testing.T) {
	tests := []struct {
		scenario string
		tstFunc  func(t *testing.T, expectationDB *fixtures.ExpectationDB)
	}{
		{"get collection, none", testGetCollectionNone},
		{"get collection", testGetCollection},
		{"get collection with tombstone", testGetCollectionTombstone},
		{"get collection should return Publication if a publish status exists", testGetCollectionPublishStatus},
		{"get collection on draft collection should return correct Publication field if includePublishedDataset=true", testGetCollectionIncludePublishedDatasetDraft},
		{"get collection on published collection should return correct Publication field if includePublishedDataset=true", testGetCollectionIncludePublishedDatasetPublished},
	}

	ctx := context.Background()
	postgresDBConfig := test.PostgresDBConfig(t)

	for _, tt := range tests {
		t.Run(tt.scenario, func(t *testing.T) {
			db := test.NewPostgresDBFromConfig(t, postgresDBConfig)
			expectationDB := fixtures.NewExpectationDB(db, postgresDBConfig.CollectionsDatabase)

			t.Cleanup(func() {
				expectationDB.CleanUp(ctx, t)
			})

			tt.tstFunc(t, expectationDB)
		})
	}
}

func testGetCollectionNone(t *testing.T, _ *fixtures.ExpectationDB) {
	ctx := context.Background()

	// Test route
	// use a user with no collections
	callingUser := userstest.SeedUser1
	nonExistentNodeID := uuid.NewString()

	claims := apitest.DefaultClaims(callingUser)

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
		WithCollectionsStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase)

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

	user1 := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user1)
	user2 := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user2)

	expectedDatasets := apitest.NewExpectedPennsieveDatasets()

	// Set up using the ExpectationDB
	user1CollectionNoDOI := apitest.NewExpectedCollection().WithNodeID().WithUser(*user1.ID, pgdb.Owner).WithRandomLicense()
	expectationDB.CreateCollection(ctx, t, user1CollectionNoDOI)

	user1CollectionOneDOI := apitest.NewExpectedCollection().WithNodeID().WithUser(*user1.ID, pgdb.Owner).
		WithPublicDatasets(expectedDatasets.NewPublished(apitest.NewPublicContributor())).WithNTags(2)
	expectationDB.CreateCollection(ctx, t, user1CollectionOneDOI)

	user1CollectionFiveDOI := apitest.NewExpectedCollection().WithNodeID().WithUser(*user1.ID, pgdb.Owner).
		WithPublicDatasets(
			expectedDatasets.NewPublished(apitest.NewPublicContributor()),
			expectedDatasets.NewPublished(apitest.NewPublicContributor(apitest.WithMiddleInitial()), apitest.NewPublicContributor(apitest.WithOrcid())),
			expectedDatasets.NewPublished(apitest.NewPublicContributor(apitest.WithOrcid())),
			expectedDatasets.NewPublished(apitest.NewPublicContributor(apitest.WithOrcid()), apitest.NewPublicContributor(apitest.WithOrcid(), apitest.WithDegree())),
			expectedDatasets.NewPublished(apitest.NewPublicContributor(apitest.WithOrcid())),
		).WithRandomLicense().WithNTags(6)
	expectationDB.CreateCollection(ctx, t, user1CollectionFiveDOI)

	user2Collection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user2.ID, pgdb.Owner).
		WithPublicDatasets(
			expectedDatasets.NewPublished(apitest.NewPublicContributor()),
			expectedDatasets.NewPublished(apitest.NewPublicContributor(apitest.WithMiddleInitial(), apitest.WithDegree(), apitest.WithOrcid())),
		).
		WithRandomLicense()
	expectationDB.CreateCollection(ctx, t, user2Collection)

	mockDiscoverServer := httptest.NewServer(mocks.ToDiscoverHandlerFunc(ctx, t, expectedDatasets.GetDatasetsByDOIFunc(t)))
	defer mockDiscoverServer.Close()

	user1Claims := apitest.DefaultClaims(user1)
	user2Claims := apitest.DefaultClaims(user2)

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(apitest.PennsieveConfig(mockDiscoverServer.URL)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
		WithCollectionsStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase).
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
	assertEqualExpectedCollectionSummary(t, user1CollectionNoDOI, user1NoDOIResp.CollectionSummary, expectedDatasets)
	assert.Empty(t, user1NoDOIResp.Datasets)
	assert.Empty(t, user1NoDOIResp.DerivedContributors)
	assertDraftPublication(t, user1NoDOIResp.CollectionSummary)

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
	assertEqualExpectedGetCollectionResponse(t, user1CollectionOneDOI, user1OneDOIResp, expectedDatasets)
	assertDraftPublication(t, user1OneDOIResp.CollectionSummary)

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
	assertEqualExpectedGetCollectionResponse(t, user1CollectionFiveDOI, user1FiveDOIResp, expectedDatasets)
	assertDraftPublication(t, user1FiveDOIResp.CollectionSummary)

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
	assertEqualExpectedGetCollectionResponse(t, user2Collection, user2CollectionResp, expectedDatasets)
	assertDraftPublication(t, user2CollectionResp.CollectionSummary)

}

func testGetCollectionTombstone(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	callingUser := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, callingUser)
	expectedDatasets := apitest.NewExpectedPennsieveDatasets()

	expectedPublicDataset := expectedDatasets.NewPublished(apitest.NewPublicContributor(apitest.WithDegree()), apitest.NewPublicContributor())
	expectedTombstone := expectedDatasets.NewUnpublished()

	expectedCollection := apitest.NewExpectedCollection().WithNodeID().WithUser(*callingUser.ID, pgdb.Owner).WithPublicDatasets(expectedPublicDataset).WithTombstones(expectedTombstone)
	expectationDB.CreateCollection(ctx, t, expectedCollection)

	mockDiscoverServer := httptest.NewServer(mocks.ToDiscoverHandlerFunc(ctx, t, expectedDatasets.GetDatasetsByDOIFunc(t)))
	defer mockDiscoverServer.Close()

	userClaims := apitest.DefaultClaims(callingUser)

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(apitest.PennsieveConfig(mockDiscoverServer.URL)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
		WithCollectionsStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase).
		WithHTTPTestDiscover(mockDiscoverServer.URL)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(GetCollectionRouteKey).
			WithClaims(userClaims).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
			Build(),
		Container: container,
		Config:    apiConfig,
		Claims:    &userClaims,
	}
	resp, err := GetCollection(ctx, params)
	require.NoError(t, err)

	assertEqualExpectedCollectionSummary(t, expectedCollection, resp.CollectionSummary, expectedDatasets)
	// Only the public dataset will add to the derived contributors
	assert.Equal(t, expectedDatasets.ExpectedContributorsForDOI(t, expectedPublicDataset.DOI), resp.DerivedContributors)

	require.Len(t, resp.Datasets, 2)
	// should be in same order that the DOIs were added to the ExpectedCollection
	var actualPublicDataset dto.PublicDataset
	apitest.RequireAsPennsieveDataset(t, resp.Datasets[0], &actualPublicDataset)
	assert.Equal(t, expectedPublicDataset, actualPublicDataset)

	var actualTombstone dto.Tombstone
	apitest.RequireAsPennsieveTombstone(t, resp.Datasets[1], &actualTombstone)
	assert.Equal(t, expectedTombstone, actualTombstone)

}

func testGetCollectionPublishStatus(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	user := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	expectedDatasets := apitest.NewExpectedPennsieveDatasets()

	// Set up using the ExpectationDB
	collection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user.ID, pgdb.Owner).
		WithPublicDatasets(expectedDatasets.NewPublished(apitest.NewPublicContributor()))
	expectationDB.CreateCollection(ctx, t, collection)

	expectedPublishStatus := collectionstest.NewCompletedPublishStatus(*collection.ID, *user.ID)
	expectationDB.CreatePublishStatus(ctx, t, expectedPublishStatus)

	mockDiscoverServer := httptest.NewServer(mocks.ToDiscoverHandlerFunc(ctx, t, expectedDatasets.GetDatasetsByDOIFunc(t)))
	defer mockDiscoverServer.Close()

	claims := apitest.DefaultClaims(user)

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(apitest.PennsieveConfig(mockDiscoverServer.URL)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
		WithCollectionsStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase).
		WithHTTPTestDiscover(mockDiscoverServer.URL)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(GetCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *collection.NodeID).
			Build(),
		Container: container,
		Config:    apiConfig,
		Claims:    &claims,
	}
	resp, err := GetCollection(ctx, params)
	assert.NoError(t, err)
	assertEqualExpectedPublishStatus(t, expectedPublishStatus, resp.CollectionSummary)

}

func testGetCollectionIncludePublishedDatasetDraft(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	user := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	expectedDatasets := apitest.NewExpectedPennsieveDatasets()

	// Set up using the ExpectationDB
	collection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user.ID, pgdb.Owner).
		WithPublicDatasets(expectedDatasets.NewPublished(apitest.NewPublicContributor()))
	expectationDB.CreateCollection(ctx, t, collection)

	mockDiscoverServer := httptest.NewServer(mocks.ToDiscoverHandlerFunc(ctx, t, expectedDatasets.GetDatasetsByDOIFunc(t)))
	defer mockDiscoverServer.Close()

	claims := apitest.DefaultClaims(user)

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(apitest.PennsieveConfig(mockDiscoverServer.URL)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
		WithCollectionsStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase).
		WithHTTPTestDiscover(mockDiscoverServer.URL)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(GetCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *collection.NodeID).
			WithQueryParam(IncludePublishedDatasetQueryParamKey, "true").
			Build(),
		Container: container,
		Config:    apiConfig,
		Claims:    &claims,
	}
	resp, err := GetCollection(ctx, params)
	assert.NoError(t, err)
	assertDraftPublication(t, resp.CollectionSummary)

}

func testGetCollectionIncludePublishedDatasetPublished(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	user := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	expectedDatasets := apitest.NewExpectedPennsieveDatasets()

	// Set up using the ExpectationDB
	collection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user.ID, pgdb.Owner).
		WithPublicDatasets(expectedDatasets.NewPublished(apitest.NewPublicContributor()))
	expectationDB.CreateCollection(ctx, t, collection)

	expectedPublishStatus := collectionstest.NewCompletedPublishStatus(*collection.ID, *user.ID)
	expectationDB.CreatePublishStatus(ctx, t, expectedPublishStatus)

	pennsieveConfig := apitest.PennsieveConfigWithOptions()

	expectedOrgServiceRole := apitest.ExpectedOrgServiceRole(pennsieveConfig.CollectionsIDSpace.ID)
	expectedDatasetServiceRole := collection.DatasetServiceRole(role.Owner)

	expectedDiscoverLastPublishedDate := time.Now().UTC().Add(-72 * time.Hour)
	expectedDiscoverPublishStatus := service.DatasetPublishStatusResponse{
		Name:                  collection.Name,
		SourceOrganizationID:  int32(pennsieveConfig.CollectionsIDSpace.ID),
		SourceDatasetID:       int32(*collection.ID),
		PublishedDatasetID:    rand.Int31n(1000) + 1,
		PublishedVersionCount: rand.Int31n(10),
		Status:                dto.PublishSucceeded,
		LastPublishedDate:     &expectedDiscoverLastPublishedDate,
	}
	mockDiscoverMux := mocks.NewDiscoverMux(*pennsieveConfig.JWTSecretKey.Value).
		WithGetDatasetsByDOIFunc(ctx, t, expectedDatasets.GetDatasetsByDOIFunc(t)).
		WithGetCollectionPublishStatusFunc(ctx, t, collection.GetCollectionPublishStatusFunc(t, expectedDiscoverPublishStatus), *collection.NodeID, expectedOrgServiceRole, expectedDatasetServiceRole)
	mockDiscoverServer := httptest.NewServer(mockDiscoverMux)
	defer mockDiscoverServer.Close()
	pennsieveConfig.DiscoverServiceURL = mockDiscoverServer.URL

	claims := apitest.DefaultClaims(user)

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(pennsieveConfig).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
		WithCollectionsStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase).
		WithHTTPTestDiscover(mockDiscoverServer.URL).
		WithHTTPTestInternalDiscover(pennsieveConfig)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(GetCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *collection.NodeID).
			WithQueryParam(IncludePublishedDatasetQueryParamKey, "true").
			Build(),
		Container: container,
		Config:    apiConfig,
		Claims:    &claims,
	}
	resp, err := GetCollection(ctx, params)
	assert.NoError(t, err)
	assertEqualExpectedPublishStatus(t, expectedPublishStatus, resp.CollectionSummary)
	require.NotNil(t, resp.Publication)
	require.NotNil(t, resp.Publication.PublishedDataset)
	actual := resp.Publication.PublishedDataset
	assert.Equal(t, expectedDiscoverPublishStatus.PublishedDatasetID, actual.ID)
	assert.Equal(t, expectedDiscoverPublishStatus.PublishedVersionCount, actual.Version)
	assert.Equal(t, expectedDiscoverPublishStatus.LastPublishedDate, actual.LastPublishedDate)

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
	callingUser := userstest.SeedUser1

	expectedCollection := apitest.NewExpectedCollection().WithRandomID().WithNodeID().WithUser(callingUser.ID, pgdb.Owner)

	mockCollectionStore := mocks.NewCollectionsStore().
		WithGetCollectionFunc(expectedCollection.GetCollectionFunc(t, nil))

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
	callingUser := userstest.SeedUser1

	expectedDOI := apitest.NewPennsieveDOI()
	expectedCollection := apitest.NewExpectedCollection().WithRandomID().WithNodeID().WithUser(callingUser.ID, pgdb.Owner).WithDOIs(expectedDOI)

	mockCollectionStore := mocks.NewCollectionsStore().
		WithGetCollectionFunc(expectedCollection.GetCollectionFunc(t, nil))

	mockDiscover := mocks.NewDiscover().WithGetDatasetsByDOIFunc(func(ctx context.Context, dois []string) (service.DatasetsByDOIResponse, error) {
		return service.DatasetsByDOIResponse{Published: map[string]dto.PublicDataset{
			expectedDOI.Value: apitest.NewPublicDataset(expectedDOI.Value, apitest.NewBanner(), apitest.NewPublicContributor()),
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
