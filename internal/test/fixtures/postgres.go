package fixtures

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/pennsieve/collections-service/internal/api/store"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/stretchr/testify/require"
)

func GetCollection(ctx context.Context, t require.TestingT, conn *pgx.Conn, collectionID int64) store.Collection {
	test.Helper(t)
	rows, err := conn.Query(ctx, "SELECT * from collections.collections where id = @id", pgx.NamedArgs{"id": collectionID})
	require.NoError(t, err)
	collection, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[store.Collection])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			require.FailNow(t, "no collection found with id", "id = %d", collectionID)
		} else if errors.Is(err, pgx.ErrTooManyRows) {
			require.FailNow(t, "more than one collection found with id", "id = %d", collectionID)
		} else {
			require.NoError(t, err)
		}
	}
	return collection
}
func GetCollectionByNodeID(ctx context.Context, t require.TestingT, conn *pgx.Conn, collectionNodeID string) store.Collection {
	test.Helper(t)
	rows, err := conn.Query(ctx, "SELECT * from collections.collections where node_id = @nodeId", pgx.NamedArgs{"nodeId": collectionNodeID})
	require.NoError(t, err)
	collection, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[store.Collection])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			require.FailNow(t, "no collection found with node id", "node id = %d", collectionNodeID)
		} else if errors.Is(err, pgx.ErrTooManyRows) {
			require.FailNow(t, "more than one collection found with node id", "node id = %d", collectionNodeID)
		} else {
			require.NoError(t, err)
		}
	}
	return collection

}

func GetCollectionUsers(ctx context.Context, t require.TestingT, conn *pgx.Conn, collectionID int64) (userIDToCollectionUser map[int64]store.CollectionUser) {
	test.Helper(t)
	rows, err := conn.Query(ctx,
		"SELECT * FROM collections.collection_user WHERE collection_id = @collection_id",
		pgx.NamedArgs{"collection_id": collectionID})
	require.NoError(t, err)
	collectionUsers, err := pgx.CollectRows(rows, pgx.RowToStructByName[store.CollectionUser])
	require.NoError(t, err)
	userIDToCollectionUser = make(map[int64]store.CollectionUser, len(collectionUsers))
	for _, collectionUser := range collectionUsers {
		userIDToCollectionUser[collectionUser.UserID] = collectionUser
	}
	return
}

func GetDOIs(ctx context.Context, t require.TestingT, conn *pgx.Conn, collectionID int64) (doiToDOI map[string]store.CollectionDOI) {
	test.Helper(t)
	rows, err := conn.Query(ctx,
		"SELECT * FROM collections.dois WHERE collection_id = @collection_id",
		pgx.NamedArgs{"collection_id": collectionID})
	require.NoError(t, err)
	dois, err := pgx.CollectRows(rows, pgx.RowToStructByName[store.CollectionDOI])
	require.NoError(t, err)
	doiToDOI = make(map[string]store.CollectionDOI, len(dois))
	for _, doi := range dois {
		doiToDOI[doi.DOI] = doi
	}
	return
}

func CreateTestUser(ctx context.Context, t require.TestingT, conn *pgx.Conn, testUser *apitest.TestUser) {
	test.Helper(t)
	require.Nil(t, testUser.ID, "cannot create new user from TestUser: id already set: %d", *testUser.ID)
	var userID int64
	err := conn.QueryRow(ctx,
		"INSERT INTO pennsieve.users (email, node_id, is_super_admin) VALUES (@email, @node_id, @is_super_admin) RETURNING id",
		pgx.NamedArgs{"email": testUser.Email, "node_id": testUser.NodeID, "is_super_admin": testUser.IsSuperAdmin}).
		Scan(&userID)
	require.NoError(t, err, "error creating user from TestUser %v", testUser)
	testUser.ID = &userID
}
