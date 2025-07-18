package fixtures

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/pennsieve/collections-service/internal/api/publishing"
	"github.com/pennsieve/collections-service/internal/api/store/collections"
	"github.com/pennsieve/collections-service/internal/shared/logging"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/pennsieve/collections-service/internal/test/userstest"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"github.com/stretchr/testify/require"
	"log/slog"
	"maps"
	"slices"
	"time"
)

type ExpectationDB struct {
	db                     *test.PostgresDB
	dbName                 string
	internalStore          *collections.PostgresStore
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

func (e *ExpectationDB) collectionsStore() collections.Store {
	if e.internalStore == nil {
		e.internalStore = collections.NewPostgresStore(e.db, e.dbName, logging.Default.With(slog.String("source", "ExpectationDB")))
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

func (e *ExpectationDB) RequirePublishStatus(ctx context.Context, t require.TestingT, expected *apitest.ExpectedPublishStatus) {
	test.Helper(t)
	require.NotNil(t, expected.CollectionID, "expected collectionID not set")
	conn := e.connect(ctx, t)
	defer test.CloseConnection(ctx, t, conn)

	actual := GetPublishStatus(ctx, t, conn, *expected.CollectionID)
	require.Equal(t, expected.ExpectedStatus, actual.Status)
	require.Equal(t, expected.ExpectedType, actual.Type)
	require.Equal(t, expected.ExpectedUserID, *actual.UserID)
	require.NotZero(t, actual.StartedAt)
	if expected.ExpectedStatus == publishing.InProgressStatus {
		require.Nil(t, actual.FinishedAt)
		if preCondition := expected.PreCondition; preCondition != nil {
			switch preCondition.Status {
			// If we expected InProgress with an InProgress pre-condition, then we should expect that there are no changes to the pre-condition
			case publishing.InProgressStatus:
				require.Equal(t, *preCondition.UserID, expected.ExpectedUserID)
				requireTimeWithinEpsilon(t, preCondition.StartedAt, actual.StartedAt, time.Second)
			default:
				// but if the pre-condition is not InProgress, then StartedAt should have been reset
				require.True(t, actual.StartedAt.After(preCondition.StartedAt), "updated started_at %v <= previous started_at %v", actual.StartedAt, preCondition.StartedAt)
			}
		}
	} else {
		require.NotNil(t, actual.FinishedAt)
		require.False(t, (*actual.FinishedAt).Before(actual.StartedAt))
		if preCondition := expected.PreCondition; preCondition != nil {
			requireTimeWithinEpsilon(t, preCondition.StartedAt, actual.StartedAt, time.Second)
		}
	}
}

func (e *ExpectationDB) RequireNoPublishStatus(ctx context.Context, t require.TestingT, expectedCollectionID int64) {
	test.Helper(t)
	e.knownCollectionIDs[expectedCollectionID] = true
	conn := e.connect(ctx, t)
	defer test.CloseConnection(ctx, t, conn)

	rows, _ := conn.Query(ctx, "SELECT * from collections.publish_status where collection_id = @collection_id", pgx.NamedArgs{"collection_id": expectedCollectionID})
	unexpected, err := pgx.CollectOneRow(rows, func(row pgx.CollectableRow) (map[string]any, error) {
		return pgx.RowToMap(row)
	})
	require.ErrorIs(t, err, pgx.ErrNoRows, "expected no row, got %v", unexpected)
}

func (e *ExpectationDB) CreateCollection(ctx context.Context, t require.TestingT, expected *apitest.ExpectedCollection) collections.CreateCollectionResponse {
	test.Helper(t)
	ownerIdx := slices.IndexFunc(expected.Users, func(user apitest.ExpectedUser) bool {
		return user.PermissionBit == pgdb.Owner
	})
	require.True(t, ownerIdx > -1, "ExpectationDB.CreateCollection can only be called with at least one expected owner")
	expectedOwner := expected.Users[ownerIdx]
	require.NotNil(t, expected.NodeID, "ExpectationDB.CreateCollection can only be called with a non-nil node id; call WithNodeID() on ExpectedCollection")

	response, err := e.collectionsStore().CreateCollection(ctx, expectedOwner.UserID, *expected.NodeID, expected.Name, expected.Description, expected.DOIs.AsDOIs())
	require.NoError(t, err)
	expected.ID = &response.ID
	e.knownCollectionIDs[response.ID] = true

	// return if only the owner is expected
	if len(expected.Users) == 1 {
		return response
	}

	// otherwise, add other users
	otherUsers := map[int64]pgdb.DbPermission{}
	for i, user := range expected.Users {
		if i != ownerIdx {
			otherUsers[user.UserID] = user.PermissionBit
		}
	}
	conn := e.connect(ctx, t)
	defer test.CloseConnection(ctx, t, conn)
	AddCollectionUsers(ctx, t, conn, response.ID, otherUsers)

	return response
}

func (e *ExpectationDB) CreateTestUser(ctx context.Context, t require.TestingT, testUser *userstest.TestUser) {
	test.Helper(t)
	conn := e.connect(ctx, t)
	defer test.CloseConnection(ctx, t, conn)
	CreateTestUser(ctx, t, conn, testUser)
	e.createdUsers[*testUser.ID] = true
}

func (e *ExpectationDB) CreatePublishStatusPreCondition(ctx context.Context, t require.TestingT, expectedPublishStatus *apitest.ExpectedPublishStatus) {
	test.Helper(t)
	require.NotNil(t, expectedPublishStatus.PreCondition, "the given ExpectedPublishStatus does not have a precondition")
	require.NotNil(t, expectedPublishStatus.PreCondition.CollectionID, "collectionID not set on PreCondition")
	require.Equal(t, expectedPublishStatus.PreCondition.CollectionID, *expectedPublishStatus.CollectionID,
		"PreCondition.CollectionID %d does not match CollectionID %d",
		expectedPublishStatus.PreCondition.CollectionID,
		*expectedPublishStatus.CollectionID)
	conn := e.connect(ctx, t)
	defer test.CloseConnection(ctx, t, conn)

	AddPublishStatus(ctx, t, conn, *expectedPublishStatus.PreCondition)
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

func requireCollection(ctx context.Context, t require.TestingT, conn *pgx.Conn, expected *apitest.ExpectedCollection, actual collections.Collection) {
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
		require.Equal(t, expectedDOI.Datasource, actualDOI.Datasource)
		require.NotZero(t, actualDOI.CreatedAt)
		require.NotZero(t, actualDOI.UpdatedAt)
	}
}

// requireTimeWithinEpsilon will fail test if absolute value of diff between expected and actual is more
// than epsilon.
// For occasions when one cannot use time.Equal because one value goes through a deserialization process that
// creates small differences, for example, coming out of the DB.
func requireTimeWithinEpsilon(t require.TestingT, expected, actual time.Time, epsilon time.Duration) {
	delta := expected.Sub(actual)
	require.LessOrEqual(t, delta.Abs(), epsilon,
		"actual %v more than %v away from expected %v: %v",
		actual,
		epsilon,
		expected,
		delta.Abs())
}
