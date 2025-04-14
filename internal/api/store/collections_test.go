package store_test

import (
	"context"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/store"
	"github.com/pennsieve/collections-service/internal/dbmigrate"
	"github.com/pennsieve/collections-service/internal/shared/logging"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/pennsieve/collections-service/internal/test/configtest"
	"github.com/pennsieve/collections-service/internal/test/dbmigratetest"
	"github.com/pennsieve/collections-service/internal/test/fixtures"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestStore(t *testing.T) {
	ctx := context.Background()
	config := configtest.PostgresDBConfig()
	migrator, err := dbmigrate.NewLocalCollectionsMigrator(ctx, dbmigrate.Config{
		PostgresDB:     config,
		VerboseLogging: true,
	})
	require.NoError(t, err)
	require.NoError(t, migrator.Up())
	dbmigratetest.Close(t, migrator)

	for _, tt := range []struct {
		scenario string
		tstFunc  func(t *testing.T, collectionsStore *store.PostgresCollectionsStore, expectationDB *fixtures.ExpectationDB)
	}{
		{"create collection, nil DOIs", testCreateCollectionNilDOIs},
		{"create collection, empty DOIs", testCreateCollectionEmptyDOIs},
		{"create collection, one DOI", testCreateCollectionOneDOI},
		{"create collection, many DOIs", testCreateCollectionManyDOIs},
		{"create collection, empty description", testCreateCollectionEmptyDescription},
		{"get collections, none", testGetCollectionsNone},
		{"get collections", testGetCollections},
		{"get collections, limit and offset", testGetCollectionsLimitOffset},
		{"get collection, none", testGetCollectionNone},
		{"get collection", testGetCollection},
	} {

		t.Run(tt.scenario, func(t *testing.T) {
			db := test.NewPostgresDBFromConfig(t, config)

			t.Cleanup(func() {
				require.NoError(t, fixtures.TruncateCollectionsSchema(ctx, t, db, config.CollectionsDatabase))
			})

			collectionsStore := store.NewPostgresCollectionsStore(db, config.CollectionsDatabase, logging.Default)

			tt.tstFunc(t, collectionsStore, fixtures.NewExpectationDB(db, config.CollectionsDatabase))
		})
	}
}

func testCreateCollectionNilDOIs(t *testing.T, store *store.PostgresCollectionsStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()
	expectedOwnerID := apitest.User.ID
	expectedCollection := fixtures.NewExpectedCollection().WithNodeID().WithUser(expectedOwnerID, pgdb.Owner)

	resp, err := store.CreateCollection(ctx, expectedOwnerID, *expectedCollection.NodeID, expectedCollection.Name, expectedCollection.Description, nil)
	require.NoError(t, err)
	assert.Positive(t, resp.ID)
	assert.Equal(t, role.Owner, resp.CreatorRole)

	expectationDB.RequireCollection(ctx, t, expectedCollection, resp.ID)
}

func testCreateCollectionEmptyDOIs(t *testing.T, store *store.PostgresCollectionsStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()
	expectedOwnerID := apitest.User.ID
	expectedCollection := fixtures.NewExpectedCollection().WithNodeID().WithUser(expectedOwnerID, pgdb.Owner)

	resp, err := store.CreateCollection(ctx, expectedOwnerID, *expectedCollection.NodeID, expectedCollection.Name, expectedCollection.Description, []string{})
	require.NoError(t, err)
	assert.Positive(t, resp.ID)
	assert.Equal(t, role.Owner, resp.CreatorRole)

	expectationDB.RequireCollection(ctx, t, expectedCollection, resp.ID)
}

func testCreateCollectionOneDOI(t *testing.T, store *store.PostgresCollectionsStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()
	expectedOwnerID := apitest.User.ID
	expectedCollection := fixtures.NewExpectedCollection().WithNodeID().WithUser(expectedOwnerID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI())

	resp, err := store.CreateCollection(ctx, expectedOwnerID, *expectedCollection.NodeID, expectedCollection.Name, expectedCollection.Description, expectedCollection.DOIs.Strings())
	require.NoError(t, err)
	assert.Positive(t, resp.ID)
	assert.Equal(t, role.Owner, resp.CreatorRole)

	expectationDB.RequireCollection(ctx, t, expectedCollection, resp.ID)

}

func testCreateCollectionManyDOIs(t *testing.T, store *store.PostgresCollectionsStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()
	expectedOwnerID := apitest.User.ID
	expectedCollection := fixtures.NewExpectedCollection().WithNodeID().WithUser(expectedOwnerID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI())

	resp, err := store.CreateCollection(ctx, expectedOwnerID, *expectedCollection.NodeID, expectedCollection.Name, expectedCollection.Description, expectedCollection.DOIs.Strings())
	require.NoError(t, err)
	assert.Positive(t, resp.ID)
	assert.Equal(t, role.Owner, resp.CreatorRole)

	expectationDB.RequireCollection(ctx, t, expectedCollection, resp.ID)

}

func testCreateCollectionEmptyDescription(t *testing.T, store *store.PostgresCollectionsStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()
	expectedOwnerID := apitest.User.ID
	expectedCollection := fixtures.NewExpectedCollection().WithNodeID().WithUser(expectedOwnerID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI())
	expectedCollection.Description = ""

	resp, err := store.CreateCollection(ctx, expectedOwnerID, *expectedCollection.NodeID, expectedCollection.Name, expectedCollection.Description, expectedCollection.DOIs.Strings())
	require.NoError(t, err)
	assert.Positive(t, resp.ID)
	assert.Equal(t, role.Owner, resp.CreatorRole)

	expectationDB.RequireCollection(ctx, t, expectedCollection, resp.ID)

}

func testGetCollectionsNone(t *testing.T, store *store.PostgresCollectionsStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	// Set up using the ExpectationDB

	user2ExpectedCollection := fixtures.NewExpectedCollection().WithNodeID().WithUser(apitest.User2.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI())
	expectationDB.CreateCollection(ctx, t, user2ExpectedCollection)

	// Test with store
	limit, offset := 10, 0
	// use a different user with no collections
	response, err := store.GetCollections(ctx, apitest.User.ID, limit, offset)
	require.NoError(t, err)

	assert.Equal(t, limit, response.Limit)
	assert.Equal(t, offset, response.Offset)
	assert.Equal(t, 0, response.TotalCount)

	assert.Len(t, response.Collections, 0)
}

func testGetCollections(t *testing.T, store *store.PostgresCollectionsStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	// Set up using the ExpectationDB
	user1CollectionNoDOI := fixtures.NewExpectedCollection().WithNodeID().WithUser(apitest.User.ID, pgdb.Owner)
	expectationDB.CreateCollection(ctx, t, user1CollectionNoDOI)
	user1CollectionOneDOI := fixtures.NewExpectedCollection().WithNodeID().WithUser(apitest.User.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI())
	expectationDB.CreateCollection(ctx, t, user1CollectionOneDOI)
	user1CollectionFiveDOI := fixtures.NewExpectedCollection().WithNodeID().WithUser(apitest.User.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI())
	expectationDB.CreateCollection(ctx, t, user1CollectionFiveDOI)

	user2Collection := fixtures.NewExpectedCollection().WithNodeID().WithUser(apitest.User2.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI())
	expectationDB.CreateCollection(ctx, t, user2Collection)

	// Test with store
	limit, offset := 10, 0
	response, err := store.GetCollections(ctx, apitest.User.ID, limit, offset)
	require.NoError(t, err)

	assert.Equal(t, limit, response.Limit)
	assert.Equal(t, offset, response.Offset)
	assert.Equal(t, 3, response.TotalCount)

	assert.Len(t, response.Collections, 3)

	// They should be returned in oldest first order
	actualCollection1 := response.Collections[0]
	assertExpectedEqualCollectionResponse(t, user1CollectionNoDOI, actualCollection1)

	actualCollection2 := response.Collections[1]
	assertExpectedEqualCollectionResponse(t, user1CollectionOneDOI, actualCollection2)

	actualCollection3 := response.Collections[2]
	assertExpectedEqualCollectionResponse(t, user1CollectionFiveDOI, actualCollection3)

	// try user2's collections
	user2CollectionResp, err := store.GetCollections(ctx, apitest.User2.ID, limit, offset)
	require.NoError(t, err)

	assert.Equal(t, limit, user2CollectionResp.Limit)
	assert.Equal(t, offset, user2CollectionResp.Offset)
	assert.Equal(t, 1, user2CollectionResp.TotalCount)
	assert.Len(t, user2CollectionResp.Collections, 1)

	actualUser2Collection := user2CollectionResp.Collections[0]
	assertExpectedEqualCollectionResponse(t, user2Collection, actualUser2Collection)

}

func testGetCollectionsLimitOffset(t *testing.T, store *store.PostgresCollectionsStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()
	totalCollections := 11
	var expectedCollections []*fixtures.ExpectedCollection
	for i := 0; i < totalCollections; i++ {
		expectedCollection := fixtures.NewExpectedCollection().WithNodeID().WithUser(apitest.User.ID, pgdb.Owner).WithNPennsieveDOIs(i)
		expectationDB.CreateCollection(ctx, t, expectedCollection)
		expectedCollections = append(expectedCollections, expectedCollection)
	}

	limit := 3
	// offsets:        0 3 6 9 12
	// response sizes: 3 3 3 2  0
	offset := 0

	for ; offset < totalCollections; offset += limit {
		resp, err := store.GetCollections(ctx, apitest.User.ID, limit, offset)
		require.NoError(t, err)

		assert.Equal(t, limit, resp.Limit)
		assert.Equal(t, offset, resp.Offset)
		assert.Equal(t, totalCollections, resp.TotalCount)

		expectedCollectionLen := min(limit, totalCollections-offset)
		if assert.Len(t, resp.Collections, expectedCollectionLen) {
			for i := 0; i < expectedCollectionLen; i++ {
				assertExpectedEqualCollectionResponse(t, expectedCollections[offset+i], resp.Collections[i])
			}
		}
	}

	// now offset >= totalCollections, so the response should have no collections
	// but still have the correct TotalCount.

	emptyResp, err := store.GetCollections(ctx, apitest.User.ID, limit, offset)
	require.NoError(t, err)

	assert.Equal(t, limit, emptyResp.Limit)
	assert.Equal(t, offset, emptyResp.Offset)
	assert.Equal(t, totalCollections, emptyResp.TotalCount)
	assert.Empty(t, emptyResp.Collections)

}

func testGetCollectionNone(t *testing.T, store *store.PostgresCollectionsStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	// Set up using the ExpectationDB

	user2ExpectedCollection := fixtures.NewExpectedCollection().WithNodeID().WithUser(apitest.User2.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI())
	expectationDB.CreateCollection(ctx, t, user2ExpectedCollection)

	// Test with store
	// use a different user with no collections
	response, err := store.GetCollection(ctx, apitest.User.ID, uuid.NewString())
	require.NoError(t, err)

	assert.Nil(t, response)
}

func testGetCollection(t *testing.T, store *store.PostgresCollectionsStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	// Set up using the ExpectationDB
	user1CollectionNoDOI := fixtures.NewExpectedCollection().WithNodeID().WithUser(apitest.User.ID, pgdb.Owner)
	expectationDB.CreateCollection(ctx, t, user1CollectionNoDOI)
	user1CollectionOneDOI := fixtures.NewExpectedCollection().WithNodeID().WithUser(apitest.User.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI())
	expectationDB.CreateCollection(ctx, t, user1CollectionOneDOI)
	user1CollectionFiveDOI := fixtures.NewExpectedCollection().WithNodeID().WithUser(apitest.User.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI())
	expectationDB.CreateCollection(ctx, t, user1CollectionFiveDOI)

	user2Collection := fixtures.NewExpectedCollection().WithNodeID().WithUser(apitest.User2.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI())
	expectationDB.CreateCollection(ctx, t, user2Collection)

	// Test with store
	// user1NoDOIs
	user1NoDOIResp, err := store.GetCollection(ctx, apitest.User.ID, *user1CollectionNoDOI.NodeID)
	require.NoError(t, err)
	assert.NotNil(t, user1NoDOIResp)
	assertExpectedEqualCollectionResponse(t, user1CollectionNoDOI, user1NoDOIResp.CollectionResponse)
	assert.Empty(t, user1NoDOIResp.DOIs)

	// user1OneDOI
	user1OneDOIResp, err := store.GetCollection(ctx, apitest.User.ID, *user1CollectionOneDOI.NodeID)
	assert.NoError(t, err)
	assert.NotNil(t, user1CollectionOneDOI)
	assertExpectedEqualCollectionResponse(t, user1CollectionOneDOI, user1OneDOIResp.CollectionResponse)
	assert.Equal(t, user1CollectionOneDOI.DOIs.Strings(), user1OneDOIResp.DOIs)

	// user1FiveDOI
	user1FiveDOIResp, err := store.GetCollection(ctx, apitest.User.ID, *user1CollectionFiveDOI.NodeID)
	assert.NoError(t, err)
	assert.NotNil(t, user1CollectionFiveDOI)
	assertExpectedEqualCollectionResponse(t, user1CollectionFiveDOI, user1FiveDOIResp.CollectionResponse)
	assert.Equal(t, user1CollectionFiveDOI.DOIs.Strings(), user1FiveDOIResp.DOIs)

	// try user2's collections
	user2CollectionResp, err := store.GetCollection(ctx, apitest.User2.ID, *user2Collection.NodeID)
	require.NoError(t, err)
	assert.NotNil(t, user2CollectionResp)
	assertExpectedEqualCollectionResponse(t, user2Collection, user2CollectionResp.CollectionResponse)
	assert.Equal(t, user2Collection.DOIs.Strings(), user2CollectionResp.DOIs)

}

func assertExpectedEqualCollectionResponse(t *testing.T, expected *fixtures.ExpectedCollection, actual store.CollectionResponse) {
	assert.Equal(t, *expected.NodeID, actual.NodeID)
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.Description, actual.Description)
	assert.Equal(t, expected.Users[0].PermissionBit.ToRole().String(), actual.UserRole)
	assert.Len(t, expected.DOIs, actual.Size)
	bannerLen := min(store.MaxDOIsPerCollection, len(expected.DOIs))
	assert.Equal(t, expected.DOIs.Strings()[:bannerLen], actual.BannerDOIs)
}
