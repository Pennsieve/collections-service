package routes

import (
	"context"
	"github.com/pennsieve/collections-service/internal/api/config"
	"github.com/pennsieve/collections-service/internal/api/dto"
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

func testGetCollectionsNone(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	// Set up using the ExpectationDB
	user2ExpectedCollection := fixtures.NewExpectedCollection().WithNodeID().WithUser(apitest.User2.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI())
	expectationDB.CreateCollection(ctx, t, user2ExpectedCollection)

	// Test route
	// use a different user with no collections
	callingUser := apitest.User

	claims := apitest.DefaultClaims(callingUser)

	apiConfig := apitest.NewConfigBuilder().
		WithDockerPostgresDBConfig().
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
		WithContainerStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase)

	limit, offset := 100, 10

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder("GET /").
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

	user1 := apitest.User

	testBanners := apitest.TestBanners{}

	// Set up using the ExpectationDB
	user1CollectionNoDOI := fixtures.NewExpectedCollection().WithNodeID().WithUser(user1.ID, pgdb.Owner)
	expectationDB.CreateCollection(ctx, t, user1CollectionNoDOI)

	user1CollectionOneDOI := fixtures.NewExpectedCollection().WithNodeID().WithUser(user1.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI())
	expectationDB.CreateCollection(ctx, t, user1CollectionOneDOI)
	testBanners.WithExpectedPennsieveBanners(user1CollectionOneDOI.DOIs.Strings())

	user1CollectionFiveDOI := fixtures.NewExpectedCollection().WithNodeID().WithUser(user1.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI())
	expectationDB.CreateCollection(ctx, t, user1CollectionFiveDOI)
	testBanners.WithExpectedPennsieveBanners(user1CollectionFiveDOI.DOIs.Strings())

	user2 := apitest.User2
	user2Collection := fixtures.NewExpectedCollection().WithNodeID().WithUser(user2.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI())
	expectationDB.CreateCollection(ctx, t, user2Collection)
	testBanners.WithExpectedPennsieveBanners(user2Collection.DOIs.Strings())

	// Test route
	user1Claims := apitest.DefaultClaims(user1)

	mockDiscoverServer := httptest.NewServer(mocks.ToDiscoverHandlerFunc(t, testBanners.ToDiscoverGetDatasetsByDOIFunc()))
	defer mockDiscoverServer.Close()

	apiConfig := apitest.NewConfigBuilder().
		WithDockerPostgresDBConfig().
		WithPennsieveConfig(apitest.PennsieveConfig(mockDiscoverServer.URL)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
		WithContainerStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase).
		WithHTTPTestDiscover(mockDiscoverServer.URL)

	user1Params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder("GET /").
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
	assertExpectedEqualCollectionResponse(t, user1CollectionNoDOI, actualCollection1, testBanners)

	actualCollection2 := response.Collections[1]
	assertExpectedEqualCollectionResponse(t, user1CollectionOneDOI, actualCollection2, testBanners)

	actualCollection3 := response.Collections[2]
	assertExpectedEqualCollectionResponse(t, user1CollectionFiveDOI, actualCollection3, testBanners)

	// try user2's collections
	user2Claims := apitest.DefaultClaims(user2)
	user2Params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder("GET /").
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
	assertExpectedEqualCollectionResponse(t, user2Collection, actualUser2Collection, testBanners)
}

func testGetCollectionsLimitOffset(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()
	totalCollections := 12
	testBanners := apitest.TestBanners{}
	var expectedCollections []*fixtures.ExpectedCollection
	for i := 0; i < totalCollections; i++ {
		expectedCollection := fixtures.NewExpectedCollection().WithNodeID().WithUser(apitest.User.ID, pgdb.Owner).WithNPennsieveDOIs(i)
		expectationDB.CreateCollection(ctx, t, expectedCollection)
		expectedCollections = append(expectedCollections, expectedCollection)
		testBanners.WithExpectedPennsieveBanners(expectedCollection.DOIs.Strings())
	}

	limit := 6
	// offsets:        0 6 12
	// response sizes: 6 6  0
	offset := 0

	userClaims := apitest.DefaultClaims(apitest.User)
	mockDiscoverServer := httptest.NewServer(mocks.ToDiscoverHandlerFunc(t, testBanners.ToDiscoverGetDatasetsByDOIFunc()))
	defer mockDiscoverServer.Close()

	apiConfig := apitest.NewConfigBuilder().
		WithDockerPostgresDBConfig().
		WithPennsieveConfig(apitest.PennsieveConfig(mockDiscoverServer.URL)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
		WithContainerStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase).
		WithHTTPTestDiscover(mockDiscoverServer.URL)

	for ; offset < totalCollections; offset += limit {
		params := Params{
			Request: apitest.NewAPIGatewayRequestBuilder("GET /").
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
				assertExpectedEqualCollectionResponse(t, expectedCollections[offset+i], resp.Collections[i], testBanners)
			}
		}
	}

	// now offset >= totalCollections, so the response should have no collections
	// but still have the correct TotalCount.
	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder("GET /").
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

func assertExpectedEqualCollectionResponse(t *testing.T, expected *fixtures.ExpectedCollection, actual dto.CollectionResponse, banners apitest.TestBanners) {
	assert.Equal(t, *expected.NodeID, actual.NodeID)
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.Description, actual.Description)
	assert.Equal(t, expected.Users[0].PermissionBit.ToRole().String(), actual.UserRole)
	assert.Len(t, expected.DOIs, actual.Size)
	bannerLen := min(config.MaxBannersPerCollection, len(expected.DOIs))
	expectedBanners := banners.GetExpectedBannersForDOIs(expected.DOIs.Strings()[:bannerLen])
	assert.Equal(t, expectedBanners, actual.Banners)
}
