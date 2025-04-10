package fixtures

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pennsieve/collections-service/internal/api/store"
	"github.com/pennsieve/collections-service/internal/shared/logging"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"github.com/stretchr/testify/require"
	"log/slog"
)

// ExpectedCollection is what we expect the collection to look like
// in Postgres, so it doesn't include things not persisted there. Like banners for
// example.
type ExpectedCollection struct {
	Name        string
	Description string
	// NodeID is optional since it may not be known depending
	// on the level we are testing. We can have an expected nodeID
	// if testing collection creation at the store level, but not at the route handling level
	// for example
	NodeID *string
	Users  []ExpectedUser
	DOIs   ExpectedDOIs
}

func NewExpectedCollection() *ExpectedCollection {
	return &ExpectedCollection{
		Name:        uuid.NewString(),
		Description: uuid.NewString(),
	}
}

func (c *ExpectedCollection) WithNodeID() *ExpectedCollection {
	nodeID := uuid.NewString()
	c.NodeID = &nodeID
	return c
}

type ExpectedUser struct {
	UserID        int64
	PermissionBit pgdb.DbPermission
}

func (c *ExpectedCollection) WithUser(userID int64, permission pgdb.DbPermission) *ExpectedCollection {
	c.Users = append(c.Users, ExpectedUser{userID, permission})
	return c
}

type ExpectedDOI struct {
	DOI string
}

func (c *ExpectedCollection) WithDOIs(dois ...string) *ExpectedCollection {
	for _, doi := range dois {
		c.DOIs = append(c.DOIs, ExpectedDOI{DOI: doi})
	}
	return c
}

func (c *ExpectedCollection) WithNPennsieveDOIs(n int) *ExpectedCollection {
	var dois []string
	for i := 0; i < n; i++ {
		dois = append(dois, test.NewPennsieveDOI())
	}
	return c.WithDOIs(dois...)
}

type ExpectedDOIs []ExpectedDOI

func (d ExpectedDOIs) Strings() []string {
	if len(d) == 0 {
		return nil
	}
	strs := make([]string, len(d))
	for i, doi := range d {
		strs[i] = doi.DOI
	}
	return strs
}

func (d ExpectedDOIs) Len64() int64 {
	return int64(len(d))
}

type ExpectationDB struct {
	db            *test.PostgresDB
	dbName        string
	internalStore *store.PostgresCollectionsStore
}

func NewExpectationDB(db *test.PostgresDB, dbName string) *ExpectationDB {
	return &ExpectationDB{
		db:     db,
		dbName: dbName,
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

func (e *ExpectationDB) RequireCollection(ctx context.Context, t require.TestingT, expected *ExpectedCollection, expectedCollectionID int64) {
	test.Helper(t)
	conn := e.connect(ctx, t)
	defer test.CloseConnection(ctx, t, conn)

	actual := GetCollection(ctx, t, conn, expectedCollectionID)
	requireCollection(ctx, t, conn, expected, actual)
}

func (e *ExpectationDB) RequireCollectionByNodeID(ctx context.Context, t require.TestingT, expected *ExpectedCollection, expectedNodeID string) {
	test.Helper(t)
	conn := e.connect(ctx, t)
	defer test.CloseConnection(ctx, t, conn)

	actual := GetCollectionByNodeID(ctx, t, conn, expectedNodeID)
	requireCollection(ctx, t, conn, expected, actual)
}

func (e *ExpectationDB) CreateCollection(ctx context.Context, t require.TestingT, expected *ExpectedCollection) store.CreateCollectionResponse {
	test.Helper(t)
	require.Len(t, expected.Users, 1, "ExpectationDB.CreateCollection can only be called with one expected user: an owner")
	user := expected.Users[0]
	require.Equal(t, pgdb.Owner, user.PermissionBit, "ExpectationDB.CreateCollection can only be called with one expected user: an owner")
	require.NotNil(t, expected.NodeID, "ExpectationDB.CreateCollection can only be called with a non-nil node id; call WithNodeID() on ExpectedCollection")

	response, err := e.collectionsStore().CreateCollection(ctx, user.UserID, *expected.NodeID, expected.Name, expected.Description, expected.DOIs.Strings())
	require.NoError(t, err)
	return response
}

func requireCollection(ctx context.Context, t require.TestingT, conn *pgx.Conn, expected *ExpectedCollection, actual store.Collection) {
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
