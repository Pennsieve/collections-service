package routes

import (
	"context"
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

func TestGetCollections(t *testing.T) {

	tests := []struct {
		scenario string
		tstFunc  func(t *testing.T, expectationDB *fixtures.ExpectationDB)
	}{
		{"get collections, none", testGetCollectionsNone},
		{"get collections", testGetCollections},
		{"get collections, limit and offset", testGetCollectionsLimitOffset},
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

func testGetCollectionsNone(t *testing.T, _ *fixtures.ExpectationDB) {
	ctx := context.Background()

	// Test route
	// use a different user with no collections
	callingUser := apitest.SeedUser1

	claims := apitest.DefaultClaims(callingUser)

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
		WithContainerStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase)

	limit, offset := 100, 10

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(GetCollectionsRouteKey).
			WithClaims(claims).
			WithIntQueryParam("limit", limit).
			WithIntQueryParam("offset", offset).
			Build(),
		Container: container,
		Config:    apiConfig,
		Claims:    &claims,
	}

	response, err := GetCollections(ctx, params)
	require.NoError(t, err)

	assert.Equal(t, limit, response.Limit)
	assert.Equal(t, offset, response.Offset)
	assert.Equal(t, 0, response.TotalCount)

	assert.Len(t, response.Collections, 0)
}

func testGetCollections(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	user1 := apitest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user1)

	expectedDatasets := apitest.NewExpectedPennsieveDatasets()

	// Set up using the ExpectationDB
	user1CollectionNoDOI := apitest.NewExpectedCollection().WithNodeID().WithUser(*user1.ID, pgdb.Owner)
	expectationDB.CreateCollection(ctx, t, user1CollectionNoDOI)

	user1CollectionOneDOI := apitest.NewExpectedCollection().WithNodeID().WithUser(*user1.ID, pgdb.Owner).
		WithDOIs(expectedDatasets.NewPublished().DOI)
	expectationDB.CreateCollection(ctx, t, user1CollectionOneDOI)

	user1CollectionFiveDOI := apitest.NewExpectedCollection().WithNodeID().WithUser(*user1.ID, pgdb.Owner).
		WithDOIs(expectedDatasets.NewPublished().DOI, expectedDatasets.NewPublished().DOI, expectedDatasets.NewPublished().DOI, expectedDatasets.NewPublished().DOI, expectedDatasets.NewPublished().DOI)
	expectationDB.CreateCollection(ctx, t, user1CollectionFiveDOI)

	user2 := apitest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user2)
	user2Collection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user2.ID, pgdb.Owner).
		WithDOIs(expectedDatasets.NewPublished().DOI, expectedDatasets.NewPublished().DOI)
	expectationDB.CreateCollection(ctx, t, user2Collection)

	// Test route
	user1Claims := apitest.DefaultClaims(user1)

	mockDiscoverServer := httptest.NewServer(mocks.ToDiscoverHandlerFunc(t, expectedDatasets.GetDatasetsByDOIFunc(t)))
	defer mockDiscoverServer.Close()

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(apitest.PennsieveConfig(mockDiscoverServer.URL)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
		WithContainerStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase).
		WithHTTPTestDiscover(mockDiscoverServer.URL)

	user1Params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(GetCollectionsRouteKey).
			WithClaims(user1Claims).
			Build(),
		Container: container,
		Config:    apiConfig,
		Claims:    &user1Claims,
	}

	response, err := GetCollections(ctx, user1Params)
	require.NoError(t, err)

	assert.Equal(t, DefaultGetCollectionsLimit, response.Limit)
	assert.Equal(t, DefaultGetCollectionsOffset, response.Offset)
	assert.Equal(t, 3, response.TotalCount)

	assert.Len(t, response.Collections, 3)

	// They should be returned in oldest first order
	actualCollection1 := response.Collections[0]
	assertExpectedEqualCollectionSummary(t, user1CollectionNoDOI, actualCollection1, expectedDatasets)

	actualCollection2 := response.Collections[1]
	assertExpectedEqualCollectionSummary(t, user1CollectionOneDOI, actualCollection2, expectedDatasets)

	actualCollection3 := response.Collections[2]
	assertExpectedEqualCollectionSummary(t, user1CollectionFiveDOI, actualCollection3, expectedDatasets)

	// try user2's collections
	user2Claims := apitest.DefaultClaims(user2)
	user2Params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(GetCollectionsRouteKey).
			WithClaims(user2Claims).
			Build(),
		Container: container,
		Config:    apiConfig,
		Claims:    &user2Claims}
	user2CollectionResp, err := GetCollections(ctx, user2Params)
	require.NoError(t, err)

	assert.Equal(t, DefaultGetCollectionsLimit, user2CollectionResp.Limit)
	assert.Equal(t, DefaultGetCollectionsOffset, user2CollectionResp.Offset)
	assert.Equal(t, 1, user2CollectionResp.TotalCount)
	assert.Len(t, user2CollectionResp.Collections, 1)

	actualUser2Collection := user2CollectionResp.Collections[0]
	assertExpectedEqualCollectionSummary(t, user2Collection, actualUser2Collection, expectedDatasets)
}

func testGetCollectionsLimitOffset(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	user := apitest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)
	totalCollections := 12
	expectedDatasets := apitest.NewExpectedPennsieveDatasets()
	var expectedCollections []*apitest.ExpectedCollection
	for i := 0; i < totalCollections; i++ {
		expectedCollection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user.ID, pgdb.Owner)
		for j := 0; j < i; j++ {
			expectedCollection = expectedCollection.WithDOIs(expectedDatasets.NewPublished().DOI)
		}
		expectationDB.CreateCollection(ctx, t, expectedCollection)
		expectedCollections = append(expectedCollections, expectedCollection)
	}

	limit := 6
	// offsets:        0 6 12
	// response sizes: 6 6  0
	offset := 0

	userClaims := apitest.DefaultClaims(user)
	mockDiscoverServer := httptest.NewServer(mocks.ToDiscoverHandlerFunc(t, expectedDatasets.GetDatasetsByDOIFunc(t)))
	defer mockDiscoverServer.Close()

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(apitest.PennsieveConfig(mockDiscoverServer.URL)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
		WithContainerStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase).
		WithHTTPTestDiscover(mockDiscoverServer.URL)

	for ; offset < totalCollections; offset += limit {
		params := Params{
			Request: apitest.NewAPIGatewayRequestBuilder(GetCollectionsRouteKey).
				WithClaims(userClaims).
				WithIntQueryParam("limit", limit).
				WithIntQueryParam("offset", offset).
				Build(),
			Container: container,
			Config:    apiConfig,
			Claims:    &userClaims,
		}
		resp, err := GetCollections(ctx, params)
		require.NoError(t, err)

		assert.Equal(t, limit, resp.Limit)
		assert.Equal(t, offset, resp.Offset)
		assert.Equal(t, totalCollections, resp.TotalCount)

		expectedCollectionLen := min(limit, totalCollections-offset)
		if assert.Len(t, resp.Collections, expectedCollectionLen) {
			for i := 0; i < expectedCollectionLen; i++ {
				assertExpectedEqualCollectionSummary(t, expectedCollections[offset+i], resp.Collections[i], expectedDatasets)
			}
		}
	}

	// now offset >= totalCollections, so the response should have no collections
	// but still have the correct TotalCount.
	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(GetCollectionsRouteKey).
			WithClaims(userClaims).
			WithIntQueryParam("limit", limit).
			WithIntQueryParam("offset", offset).
			Build(),
		Container: container,
		Config:    apiConfig,
		Claims:    &userClaims,
	}
	emptyResp, err := GetCollections(ctx, params)
	require.NoError(t, err)

	assert.Equal(t, limit, emptyResp.Limit)
	assert.Equal(t, offset, emptyResp.Offset)
	assert.Equal(t, totalCollections, emptyResp.TotalCount)
	assert.Empty(t, emptyResp.Collections)
}

// TestHandleGetCollections tests that run the Handle wrapper around GetCollections
func TestHandleGetCollections(t *testing.T) {
	tests := []struct {
		name    string
		tstFunc func(t *testing.T)
	}{
		{
			"return empty collections array instead of null",
			testHandleGetCollectionsEmptyCollectionsArray,
		},
		{
			"return empty banners array instead of null",
			testHandleGetCollectionsEmptyBannersArray,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.tstFunc(t)
		})
	}
}

func testHandleGetCollectionsEmptyCollectionsArray(t *testing.T) {
	ctx := context.Background()
	callingUser := apitest.SeedUser1

	mockCollectionStore := mocks.NewMockCollectionsStore().
		WithGetCollectionsFunc(func(ctx context.Context, userID int64, limit int, offset int) (store.GetCollectionsResponse, error) {
			return store.GetCollectionsResponse{
				Limit:  DefaultGetCollectionsLimit,
				Offset: DefaultGetCollectionsOffset,
			}, nil
		})

	claims := apitest.DefaultClaims(callingUser)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(GetCollectionsRouteKey).
			WithClaims(claims).
			Build(),
		Container: apitest.NewTestContainer().WithCollectionsStore(mockCollectionStore),
		Config:    apitest.NewConfigBuilder().WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).Build(),
		Claims:    &claims,
	}
	response, err := Handle(ctx, NewGetCollectionsRouteHandler(), params)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, response.StatusCode)

	assert.NotContains(t, response.Body, `"collections":null`)
	assert.Contains(t, response.Body, `"collections":[]`)

}

func testHandleGetCollectionsEmptyBannersArray(t *testing.T) {
	ctx := context.Background()
	callingUser := apitest.SeedUser1

	expectedCollection := apitest.NewExpectedCollection().WithNodeID().WithUser(callingUser.ID, pgdb.Owner)

	mockCollectionStore := mocks.NewMockCollectionsStore().
		WithGetCollectionsFunc(func(ctx context.Context, userID int64, limit int, offset int) (store.GetCollectionsResponse, error) {
			return store.GetCollectionsResponse{
				Limit:      DefaultGetCollectionsLimit,
				Offset:     DefaultGetCollectionsOffset,
				TotalCount: 1,
				Collections: []store.CollectionSummary{{
					CollectionBase: store.CollectionBase{
						NodeID:      *expectedCollection.NodeID,
						Name:        expectedCollection.Name,
						Description: expectedCollection.Description,
						Size:        0,
						UserRole:    role.Owner,
					},
				}},
			}, nil
		})

	claims := apitest.DefaultClaims(callingUser)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(GetCollectionsRouteKey).
			WithClaims(claims).
			Build(),
		Container: apitest.NewTestContainer().WithCollectionsStore(mockCollectionStore),
		Config:    apitest.NewConfigBuilder().WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).Build(),
		Claims:    &claims,
	}
	response, err := Handle(ctx, NewGetCollectionsRouteHandler(), params)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, response.StatusCode)

	assert.NotContains(t, response.Body, `"banners":null`)
	assert.Contains(t, response.Body, `"banners":[]`)

}
