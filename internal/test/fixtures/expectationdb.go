package fixtures

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/pennsieve/collections-service/internal/api/store"
	"github.com/pennsieve/collections-service/internal/shared/logging"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"github.com/stretchr/testify/require"
	"log/slog"
	"maps"
	"slices"
)

type ExpectationDB struct {
	db                     *test.PostgresDB
	dbName                 string
	internalStore          *store.PostgresCollectionsStore
	createdUsers           map[int64]bool
	knownCollectionIDs     map[int64]bool
	knownCollectionNodeIDs map[string]bool
}

func NewExpectationDB(db *test.PostgresDB, dbName string) *ExpectationDB {
	return &ExpectationDB{
		db:                     db,
		dbName:                 dbName,
		createdUsers:           map[int64]bool{},
		knownCollectionIDs:     map[int64]bool{},
		knownCollectionNodeIDs: map[string]bool{},
	}
}

func (e *ExpectationDB) collectionsStore() store.CollectionsStore {
	if e.internalStore == nil {
		e.internalStore = store.NewPostgresCollectionsStore(e.db, e.dbName, logging.Default.With(slog.String("source", "ExpectationDB")))
	}
	return e.internalStore
}

func (e *ExpectationDB) connect(ctx context.Context, t require.TestingT) *pgx.Conn {
	test.Helper(t)
	conn, err := e.db.Connect(ctx, e.dbName)
	require.NoError(t, err)
	return conn
}

func (e *ExpectationDB) RequireCollection(ctx context.Context, t require.TestingT, expected *apitest.ExpectedCollection, expectedCollectionID int64) {
	test.Helper(t)
	e.knownCollectionIDs[expectedCollectionID] = true
	conn := e.connect(ctx, t)
	defer test.CloseConnection(ctx, t, conn)

	actual := GetCollection(ctx, t, conn, expectedCollectionID)
	requireCollection(ctx, t, conn, expected, actual)
}

func (e *ExpectationDB) RequireNoCollection(ctx context.Context, t require.TestingT, expectedCollectionID int64) {
	test.Helper(t)
	e.knownCollectionIDs[expectedCollectionID] = true
	conn := e.connect(ctx, t)
	defer test.CloseConnection(ctx, t, conn)

	rows, _ := conn.Query(ctx, "SELECT * from collections.collections where id = @id", pgx.NamedArgs{"id": expectedCollectionID})
	unexpected, err := pgx.CollectOneRow(rows, func(row pgx.CollectableRow) (map[string]any, error) {
		return pgx.RowToMap(row)
	})
	require.ErrorIs(t, err, pgx.ErrNoRows, "expected no row, got %v", unexpected)
}

func (e *ExpectationDB) RequireCollectionByNodeID(ctx context.Context, t require.TestingT, expected *apitest.ExpectedCollection, expectedNodeID string) {
	test.Helper(t)
	e.knownCollectionNodeIDs[expectedNodeID] = true
	conn := e.connect(ctx, t)
	defer test.CloseConnection(ctx, t, conn)

	actual := GetCollectionByNodeID(ctx, t, conn, expectedNodeID)
	requireCollection(ctx, t, conn, expected, actual)
}

func (e *ExpectationDB) CreateCollection(ctx context.Context, t require.TestingT, expected *apitest.ExpectedCollection) store.CreateCollectionResponse {
	test.Helper(t)
	require.Len(t, expected.Users, 1, "ExpectationDB.CreateCollection can only be called with one expected user: an owner")
	user := expected.Users[0]
	require.Equal(t, pgdb.Owner, user.PermissionBit, "ExpectationDB.CreateCollection can only be called with one expected user: an owner")
	require.NotNil(t, expected.NodeID, "ExpectationDB.CreateCollection can only be called with a non-nil node id; call WithNodeID() on ExpectedCollection")

	response, err := e.collectionsStore().CreateCollection(ctx, user.UserID, *expected.NodeID, expected.Name, expected.Description, expected.DOIs.Strings())
	require.NoError(t, err)
	e.knownCollectionIDs[response.ID] = true
	return response
}

func (e *ExpectationDB) CreateTestUser(ctx context.Context, t require.TestingT, testUser *apitest.TestUser) {
	test.Helper(t)
	conn := e.connect(ctx, t)
	defer test.CloseConnection(ctx, t, conn)
	CreateTestUser(ctx, t, conn, testUser)
	e.createdUsers[*testUser.ID] = true
}

func (e *ExpectationDB) CleanUp(ctx context.Context, t require.TestingT) {
	test.Helper(t)
	conn := e.connect(ctx, t)
	defer test.CloseConnection(ctx, t, conn)

	if len(e.knownCollectionIDs) > 0 {
		_, err := conn.Exec(
			ctx,
			"DELETE FROM collections.collections WHERE id = ANY(@collection_ids)",
			pgx.NamedArgs{"collection_ids": slices.AppendSeq([]int64{}, maps.Keys(e.knownCollectionIDs))},
		)
		require.NoError(t, err, "error deleting collections by id in CleanUp")
	}

	if len(e.knownCollectionNodeIDs) > 0 {
		_, err := conn.Exec(
			ctx,
			"DELETE FROM collections.collections WHERE node_id = ANY(@collection_node_ids)",
			pgx.NamedArgs{"collection_node_ids": slices.AppendSeq([]string{}, maps.Keys(e.knownCollectionNodeIDs))},
		)
		require.NoError(t, err, "error deleting collections by node id in CleanUp")
	}

	if len(e.createdUsers) > 0 {
		_, err := conn.Exec(
			ctx,
			"DELETE FROM pennsieve.users WHERE id = ANY(@user_ids)",
			pgx.NamedArgs{"user_ids": slices.AppendSeq([]int64{}, maps.Keys(e.createdUsers))},
		)
		require.NoError(t, err, "error deleting test users in CleanUp")
	}
}

func requireCollection(ctx context.Context, t require.TestingT, conn *pgx.Conn, expected *apitest.ExpectedCollection, actual store.Collection) {
	require.Equal(t, expected.Name, actual.Name)
	require.Equal(t, expected.Description, actual.Description)
	if expected.NodeID != nil {
		require.Equal(t, *expected.NodeID, actual.NodeID)
	}
	require.NotZero(t, actual.CreatedAt)
	require.NotZero(t, actual.UpdatedAt)

	actualUsers := GetCollectionUsers(ctx, t, conn, actual.ID)
	require.Len(t, actualUsers, len(expected.Users))
	for _, expectedUser := range expected.Users {
		require.Contains(t, actualUsers, expectedUser.UserID)
		actualUser := actualUsers[expectedUser.UserID]
		require.Equal(t, expectedUser.PermissionBit, actualUser.PermissionBit)
		require.Equal(t, expectedUser.PermissionBit.ToRole(), actualUser.Role.AsRole())
		require.NotZero(t, actualUser.CreatedAt)
		require.NotZero(t, actualUser.UpdatedAt)
	}

	actualDOIs := GetDOIs(ctx, t, conn, actual.ID)
	require.Len(t, actualDOIs, len(expected.DOIs))
	for _, expectedDOI := range expected.DOIs {
		require.Contains(t, actualDOIs, expectedDOI.DOI)
		actualDOI := actualDOIs[expectedDOI.DOI]
		require.Equal(t, expectedDOI.DOI, actualDOI.DOI)
		require.NotZero(t, actualDOI.CreatedAt)
		require.NotZero(t, actualDOI.UpdatedAt)
	}
}
