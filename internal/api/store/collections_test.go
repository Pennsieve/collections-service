package store_test

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

	for scenario, tstFunc := range map[string]func(t *testing.T, collectionsStore *store.RDSCollectionsStore, expectationDB *ExpectationDB){
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

			tstFunc(t, collectionsStore, NewExpectationDB(db, config.CollectionsDatabase))
		})
	}
}

func createCollectionNilDOIs(t *testing.T, store *store.RDSCollectionsStore, expectationDB *ExpectationDB) {
	ctx := context.Background()
	expectedOwnerID := test.User.ID
	expectedCollection := NewExpectedCollection().WithUser(expectedOwnerID, pgdb.Owner)

	resp, err := store.CreateCollection(ctx, expectedOwnerID, expectedCollection.NodeID, expectedCollection.Name, expectedCollection.Description, nil)
	require.NoError(t, err)
	assert.Positive(t, resp.ID)
	assert.Equal(t, role.Owner, resp.CreatorRole)

	expectationDB.RequireCollection(ctx, t, expectedCollection, resp.ID)
}

func createCollectionEmptyDOIs(t *testing.T, store *store.RDSCollectionsStore, expectationDB *ExpectationDB) {
	ctx := context.Background()
	expectedOwnerID := test.User.ID
	expectedCollection := NewExpectedCollection().WithUser(expectedOwnerID, pgdb.Owner)

	resp, err := store.CreateCollection(ctx, expectedOwnerID, expectedCollection.NodeID, expectedCollection.Name, expectedCollection.Description, []string{})
	require.NoError(t, err)
	assert.Positive(t, resp.ID)
	assert.Equal(t, role.Owner, resp.CreatorRole)

	expectationDB.RequireCollection(ctx, t, expectedCollection, resp.ID)
}

func createCollectionOneDOI(t *testing.T, store *store.RDSCollectionsStore, expectationDB *ExpectationDB) {
	ctx := context.Background()
	expectedOwnerID := test.User.ID
	expectedCollection := NewExpectedCollection().WithUser(expectedOwnerID, pgdb.Owner)

	resp, err := store.CreateCollection(ctx, expectedOwnerID, expectedCollection.NodeID, expectedCollection.Name, expectedCollection.Description, []string{test.NewDOI()})
	require.NoError(t, err)
	assert.Positive(t, resp.ID)
	assert.Equal(t, role.Owner, resp.CreatorRole)

	expectationDB.RequireCollection(ctx, t, expectedCollection, resp.ID)

}

func createCollectionManyDOIs(t *testing.T, store *store.RDSCollectionsStore, expectationDB *ExpectationDB) {
	ctx := context.Background()
	expectedOwnerID := test.User.ID
	expectedCollection := NewExpectedCollection().WithUser(expectedOwnerID, pgdb.Owner)

	resp, err := store.CreateCollection(ctx, expectedOwnerID, expectedCollection.NodeID, expectedCollection.Name, expectedCollection.Description, []string{test.NewDOI(), test.NewDOI(), test.NewDOI()})
	require.NoError(t, err)
	assert.Positive(t, resp.ID)
	assert.Equal(t, role.Owner, resp.CreatorRole)

	expectationDB.RequireCollection(ctx, t, expectedCollection, resp.ID)

}

func createCollectionEmptyDescription(t *testing.T, store *store.RDSCollectionsStore, expectationDB *ExpectationDB) {
	ctx := context.Background()
	expectedOwnerID := test.User.ID
	expectedCollection := NewExpectedCollection().WithUser(expectedOwnerID, pgdb.Owner)
	expectedCollection.Description = ""

	resp, err := store.CreateCollection(ctx, expectedOwnerID, expectedCollection.NodeID, expectedCollection.Name, expectedCollection.Description, []string{test.NewDOI(), test.NewDOI(), test.NewDOI()})
	require.NoError(t, err)
	assert.Positive(t, resp.ID)
	assert.Equal(t, role.Owner, resp.CreatorRole)

	expectationDB.RequireCollection(ctx, t, expectedCollection, resp.ID)

}

type ExpectedCollection struct {
	Name        string
	Description string
	NodeID      string
	Users       []ExpectedUser
}

func NewExpectedCollection() *ExpectedCollection {
	return &ExpectedCollection{
		Name:        uuid.NewString(),
		Description: uuid.NewString(),
		NodeID:      uuid.NewString(),
	}
}

type ExpectedUser struct {
	UserID        int64
	PermissionBit pgdb.DbPermission
}

func (c *ExpectedCollection) WithUser(userID int64, permission pgdb.DbPermission) *ExpectedCollection {
	c.Users = append(c.Users, ExpectedUser{userID, permission})
	return c
}

type ExpectationDB struct {
	db     *test.PostgresDB
	dbName string
}

func NewExpectationDB(db *test.PostgresDB, dbName string) *ExpectationDB {
	return &ExpectationDB{
		db:     db,
		dbName: dbName,
	}
}

func (e *ExpectationDB) Connect(ctx context.Context, t require.TestingT) *pgx.Conn {
	test.Helper(t)
	conn, err := e.db.Connect(ctx, e.dbName)
	require.NoError(t, err)
	return conn
}

func (e *ExpectationDB) RequireCollection(ctx context.Context, t require.TestingT, expected *ExpectedCollection, expectedCollectionID int64) {
	test.Helper(t)
	conn := e.Connect(ctx, t)
	defer test.CloseConnection(ctx, t, conn)

	actual := fixtures.GetCollection(ctx, t, conn, expectedCollectionID)
	require.Equal(t, expected.Name, actual.Name)
	require.Equal(t, expected.Description, actual.Description)
	require.Equal(t, expected.NodeID, actual.NodeID)
	require.NotZero(t, actual.CreatedAt)
	require.NotZero(t, actual.UpdatedAt)

	actualUsers := fixtures.GetCollectionUsers(ctx, t, conn, expectedCollectionID)
	require.Len(t, actualUsers, len(expected.Users))
	for _, expectedUser := range expected.Users {
		require.Contains(t, actualUsers, expectedUser.UserID)
		actualUser := actualUsers[expectedUser.UserID]
		require.Equal(t, expectedUser.PermissionBit, actualUser.PermissionBit)
		require.Equal(t, expectedUser.PermissionBit.ToRole(), actualUser.Role.AsRole())
		require.NotZero(t, actualUser.CreatedAt)
		require.NotZero(t, actualUser.UpdatedAt)
	}
}
