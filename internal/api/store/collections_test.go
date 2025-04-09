package store_test

import (
	"context"
	"github.com/pennsieve/collections-service/internal/api/store"
	"github.com/pennsieve/collections-service/internal/dbmigrate"
	"github.com/pennsieve/collections-service/internal/shared/logging"
	"github.com/pennsieve/collections-service/internal/test"
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

	for scenario, tstFunc := range map[string]func(t *testing.T, collectionsStore *store.PostgresCollectionsStore, expectationDB *fixtures.ExpectationDB){
		"create collection, nil DOIs":          testCreateCollectionNilDOIs,
		"create collection, empty DOIs":        testCreateCollectionEmptyDOIs,
		"create collection, one DOI":           testCreateCollectionOneDOI,
		"create collection, many DOIs":         testCreateCollectionManyDOIs,
		"create collection, empty description": testCreateCollectionEmptyDescription,
		"get collections, none":                testGetCollectionsNone,
		"get collections":                      testGetCollections,
	} {

		t.Run(scenario, func(t *testing.T) {
			db := test.NewPostgresDBFromConfig(t, config)

			t.Cleanup(func() {
				require.NoError(t, fixtures.TruncateCollectionsSchema(ctx, t, db, config.CollectionsDatabase))
			})

			collectionsStore := store.NewPostgresCollectionsStore(db, config.CollectionsDatabase, logging.Default)

			tstFunc(t, collectionsStore, fixtures.NewExpectationDB(db, config.CollectionsDatabase))
		})
	}
}

func testCreateCollectionNilDOIs(t *testing.T, store *store.PostgresCollectionsStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()
	expectedOwnerID := test.User.ID
	expectedCollection := fixtures.NewExpectedCollection().WithNodeID().WithUser(expectedOwnerID, pgdb.Owner)

	resp, err := store.CreateCollection(ctx, expectedOwnerID, *expectedCollection.NodeID, expectedCollection.Name, expectedCollection.Description, nil)
	require.NoError(t, err)
	assert.Positive(t, resp.ID)
	assert.Equal(t, role.Owner, resp.CreatorRole)

	expectationDB.RequireCollection(ctx, t, expectedCollection, resp.ID)
}

func testCreateCollectionEmptyDOIs(t *testing.T, store *store.PostgresCollectionsStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()
	expectedOwnerID := test.User.ID
	expectedCollection := fixtures.NewExpectedCollection().WithNodeID().WithUser(expectedOwnerID, pgdb.Owner)

	resp, err := store.CreateCollection(ctx, expectedOwnerID, *expectedCollection.NodeID, expectedCollection.Name, expectedCollection.Description, []string{})
	require.NoError(t, err)
	assert.Positive(t, resp.ID)
	assert.Equal(t, role.Owner, resp.CreatorRole)

	expectationDB.RequireCollection(ctx, t, expectedCollection, resp.ID)
}

func testCreateCollectionOneDOI(t *testing.T, store *store.PostgresCollectionsStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()
	expectedOwnerID := test.User.ID
	expectedCollection := fixtures.NewExpectedCollection().WithNodeID().WithUser(expectedOwnerID, pgdb.Owner).WithDOIs(test.NewPennsieveDOI())

	resp, err := store.CreateCollection(ctx, expectedOwnerID, *expectedCollection.NodeID, expectedCollection.Name, expectedCollection.Description, expectedCollection.DOIs.Strings())
	require.NoError(t, err)
	assert.Positive(t, resp.ID)
	assert.Equal(t, role.Owner, resp.CreatorRole)

	expectationDB.RequireCollection(ctx, t, expectedCollection, resp.ID)

}

func testCreateCollectionManyDOIs(t *testing.T, store *store.PostgresCollectionsStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()
	expectedOwnerID := test.User.ID
	expectedCollection := fixtures.NewExpectedCollection().WithNodeID().WithUser(expectedOwnerID, pgdb.Owner).WithDOIs(test.NewPennsieveDOI(), test.NewPennsieveDOI(), test.NewPennsieveDOI())

	resp, err := store.CreateCollection(ctx, expectedOwnerID, *expectedCollection.NodeID, expectedCollection.Name, expectedCollection.Description, expectedCollection.DOIs.Strings())
	require.NoError(t, err)
	assert.Positive(t, resp.ID)
	assert.Equal(t, role.Owner, resp.CreatorRole)

	expectationDB.RequireCollection(ctx, t, expectedCollection, resp.ID)

}

func testCreateCollectionEmptyDescription(t *testing.T, store *store.PostgresCollectionsStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()
	expectedOwnerID := test.User.ID
	expectedCollection := fixtures.NewExpectedCollection().WithNodeID().WithUser(expectedOwnerID, pgdb.Owner).WithDOIs(test.NewPennsieveDOI(), test.NewPennsieveDOI(), test.NewPennsieveDOI())
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

	user2ExpectedCollection := fixtures.NewExpectedCollection().WithNodeID().WithUser(test.User2.ID, pgdb.Owner).WithDOIs(test.NewPennsieveDOI(), test.NewPennsieveDOI())
	expectationDB.CreateCollection(ctx, t, user2ExpectedCollection)

	// Test with store
	limit, offset := 10, 0
	// use a different user with no collections
	response, err := store.GetCollections(ctx, test.User.ID, limit, offset)
	require.NoError(t, err)

	assert.Equal(t, limit, response.Limit)
	assert.Equal(t, offset, response.Offset)
	assert.Equal(t, int64(0), response.TotalCount)

	assert.Len(t, response.Collections, 0)
}

func testGetCollections(t *testing.T, store *store.PostgresCollectionsStore, expectationDB *fixtures.ExpectationDB) {
	t.Skip("need a second query to get dois for each collection")
	ctx := context.Background()

	// Set up using the ExpectationDB
	user1CollectionNoDOI := fixtures.NewExpectedCollection().WithNodeID().WithUser(test.User.ID, pgdb.Owner)
	expectationDB.CreateCollection(ctx, t, user1CollectionNoDOI)
	user1CollectionOneDOI := fixtures.NewExpectedCollection().WithNodeID().WithUser(test.User.ID, pgdb.Owner).WithDOIs(test.NewPennsieveDOI())
	expectationDB.CreateCollection(ctx, t, user1CollectionOneDOI)
	user1CollectionFiveDOI := fixtures.NewExpectedCollection().WithNodeID().WithUser(test.User.ID, pgdb.Owner).WithDOIs(test.NewPennsieveDOI(), test.NewPennsieveDOI(), test.NewPennsieveDOI(), test.NewPennsieveDOI(), test.NewPennsieveDOI())
	expectationDB.CreateCollection(ctx, t, user1CollectionFiveDOI)

	user2Collection := fixtures.NewExpectedCollection().WithNodeID().WithUser(test.User2.ID, pgdb.Owner).WithDOIs(test.NewPennsieveDOI(), test.NewPennsieveDOI())
	expectationDB.CreateCollection(ctx, t, user2Collection)

	// Test with store
	limit, offset := 10, 0
	response, err := store.GetCollections(ctx, test.User.ID, limit, offset)
	require.NoError(t, err)

	assert.Equal(t, limit, response.Limit)
	assert.Equal(t, offset, response.Offset)
	assert.Equal(t, int64(3), response.TotalCount)

	assert.Len(t, response.Collections, 3)

	// They should be returned in oldest first order
	actualCollection1 := response.Collections[0]
	assert.Equal(t, *user1CollectionNoDOI.NodeID, actualCollection1.NodeID)
	assert.Equal(t, user1CollectionNoDOI.Name, actualCollection1.Name)
	assert.Equal(t, user1CollectionNoDOI.Description, actualCollection1.Description)
	assert.Equal(t, user1CollectionNoDOI.Users[0].PermissionBit.ToRole().String(), actualCollection1.UserRole)
	assert.Equal(t, len(user1CollectionNoDOI.DOIs), actualCollection1.Size)
	assert.Equal(t, user1CollectionNoDOI.DOIs.Strings(), actualCollection1.BannerDOIs)

	actualCollection2 := response.Collections[1]
	assert.Equal(t, *user1CollectionOneDOI.NodeID, actualCollection2.NodeID)
	assert.Equal(t, user1CollectionOneDOI.Name, actualCollection2.Name)
	assert.Equal(t, user1CollectionOneDOI.Description, actualCollection2.Description)
	assert.Equal(t, user1CollectionOneDOI.Users[0].PermissionBit.ToRole().String(), actualCollection2.UserRole)
	assert.Equal(t, len(user1CollectionOneDOI.DOIs), actualCollection2.Size)
	assert.Equal(t, user1CollectionOneDOI.DOIs.Strings(), actualCollection2.BannerDOIs)

	actualCollection3 := response.Collections[2]
	assert.Equal(t, *user1CollectionFiveDOI.NodeID, actualCollection3.NodeID)
	assert.Equal(t, user1CollectionFiveDOI.Name, actualCollection3.Name)
	assert.Equal(t, user1CollectionFiveDOI.Description, actualCollection3.Description)
	assert.Equal(t, user1CollectionFiveDOI.Users[0].PermissionBit.ToRole().String(), actualCollection3.UserRole)
	assert.Equal(t, len(user1CollectionFiveDOI.DOIs), actualCollection3.Size)
	assert.Equal(t, user1CollectionFiveDOI.DOIs.Strings()[:4], actualCollection3.BannerDOIs)

}
