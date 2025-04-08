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

	for scenario, tstFunc := range map[string]func(t *testing.T, collectionsStore *store.RDSCollectionsStore, expectationDB *fixtures.ExpectationDB){
		"create collection, nil DOIs":          createCollectionNilDOIs,
		"create collection, empty DOIs":        createCollectionEmptyDOIs,
		"create collection, one DOI":           createCollectionOneDOI,
		"create collection, many DOIs":         createCollectionManyDOIs,
		"create collection, empty description": createCollectionEmptyDescription,
	} {

		t.Run(scenario, func(t *testing.T) {
			db := test.NewPostgresDBFromConfig(t, config)

			t.Cleanup(func() {
				require.NoError(t, fixtures.TruncateCollectionsSchema(ctx, db, config.CollectionsDatabase))
			})

			collectionsStore := store.NewRDSCollectionsStore(db, config.CollectionsDatabase, logging.Default)

			tstFunc(t, collectionsStore, fixtures.NewExpectationDB(db, config.CollectionsDatabase))
		})
	}
}

func createCollectionNilDOIs(t *testing.T, store *store.RDSCollectionsStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()
	expectedOwnerID := test.User.ID
	expectedCollection := fixtures.NewExpectedCollection().WithNodeID().WithUser(expectedOwnerID, pgdb.Owner)

	resp, err := store.CreateCollection(ctx, expectedOwnerID, *expectedCollection.NodeID, expectedCollection.Name, expectedCollection.Description, nil)
	require.NoError(t, err)
	assert.Positive(t, resp.ID)
	assert.Equal(t, role.Owner, resp.CreatorRole)

	expectationDB.RequireCollection(ctx, t, expectedCollection, resp.ID)
}

func createCollectionEmptyDOIs(t *testing.T, store *store.RDSCollectionsStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()
	expectedOwnerID := test.User.ID
	expectedCollection := fixtures.NewExpectedCollection().WithNodeID().WithUser(expectedOwnerID, pgdb.Owner)

	resp, err := store.CreateCollection(ctx, expectedOwnerID, *expectedCollection.NodeID, expectedCollection.Name, expectedCollection.Description, []string{})
	require.NoError(t, err)
	assert.Positive(t, resp.ID)
	assert.Equal(t, role.Owner, resp.CreatorRole)

	expectationDB.RequireCollection(ctx, t, expectedCollection, resp.ID)
}

func createCollectionOneDOI(t *testing.T, store *store.RDSCollectionsStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()
	expectedOwnerID := test.User.ID
	expectedCollection := fixtures.NewExpectedCollection().WithNodeID().WithUser(expectedOwnerID, pgdb.Owner).WithDOIs(test.NewPennsieveDOI())

	resp, err := store.CreateCollection(ctx, expectedOwnerID, *expectedCollection.NodeID, expectedCollection.Name, expectedCollection.Description, expectedCollection.DOIs.Strings())
	require.NoError(t, err)
	assert.Positive(t, resp.ID)
	assert.Equal(t, role.Owner, resp.CreatorRole)

	expectationDB.RequireCollection(ctx, t, expectedCollection, resp.ID)

}

func createCollectionManyDOIs(t *testing.T, store *store.RDSCollectionsStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()
	expectedOwnerID := test.User.ID
	expectedCollection := fixtures.NewExpectedCollection().WithNodeID().WithUser(expectedOwnerID, pgdb.Owner).WithDOIs(test.NewPennsieveDOI(), test.NewPennsieveDOI(), test.NewPennsieveDOI())

	resp, err := store.CreateCollection(ctx, expectedOwnerID, *expectedCollection.NodeID, expectedCollection.Name, expectedCollection.Description, expectedCollection.DOIs.Strings())
	require.NoError(t, err)
	assert.Positive(t, resp.ID)
	assert.Equal(t, role.Owner, resp.CreatorRole)

	expectationDB.RequireCollection(ctx, t, expectedCollection, resp.ID)

}

func createCollectionEmptyDescription(t *testing.T, store *store.RDSCollectionsStore, expectationDB *fixtures.ExpectationDB) {
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
