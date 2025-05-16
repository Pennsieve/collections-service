package fixtures

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/pennsieve/collections-service/internal/api/store"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"github.com/stretchr/testify/require"
	"strings"
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

func AddCollectionUser(ctx context.Context, t require.TestingT, conn *pgx.Conn, collectionID int64, userID int64, permission pgdb.DbPermission) {
	test.Helper(t)
	args := pgx.NamedArgs{
		"collection_id":  collectionID,
		"user_id":        userID,
		"permission_bit": permission,
		"role":           store.PgxRole(permission.ToRole()),
	}
	tag, err := conn.Exec(ctx,
		`INSERT INTO collections.collection_user (collection_id, user_id, permission_bit, role) 
											  VALUES (@collection_id, @user_id, @permission_bit, @role)`,
		args)
	require.NoError(t, err, "error adding user %d to collection %d", collectionID, userID)
	require.Equal(t, int64(1), tag.RowsAffected())
}

func AddCollectionUsers(ctx context.Context, t require.TestingT, conn *pgx.Conn, collectionID int64, userIDToPermission map[int64]pgdb.DbPermission) {
	test.Helper(t)
	require.NotEmpty(t, userIDToPermission)
	collectionKey := "collection_id"
	args := pgx.NamedArgs{collectionKey: collectionID}
	var values []string
	for userID, permission := range userIDToPermission {
		i := len(values)
		userKey := fmt.Sprintf("user_id_%d", i)
		permKey := fmt.Sprintf("permission_bit_%d", i)
		roleKey := fmt.Sprintf("role_%d", i)
		values = append(values, fmt.Sprintf("(@%s, @%s, @%s, @%s)", collectionKey, userKey, permKey, roleKey))
		args[userKey] = userID
		args[permKey] = permission
		args[roleKey] = store.PgxRole(permission.ToRole())
	}

	tag, err := conn.Exec(ctx,
		fmt.Sprintf(`INSERT INTO collections.collection_user (collection_id, user_id, permission_bit, role) VALUES %s`, strings.Join(values, ",")),
		args)
	require.NoError(t, err, "error adding users to collection %d", collectionID)
	require.Equal(t, int64(len(userIDToPermission)), tag.RowsAffected(), "expected to add %d users, but added %d", len(userIDToPermission), tag.RowsAffected())
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
	require.Nil(t, testUser.ID, "cannot create new user from TestUser: id already set")

	var userID, returnedID int64

	// This is a hack to get around the problem
	// that the test DB already has users with ids 1, 2, & 3, but
	// they were added with those ids specified instead of letting the id sequence for the users table
	// handle things, so the sequence will restart at
	// 1 everytime the container is started.
	require.NoError(t, pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, "SELECT nextval('pennsieve.users_id_seq') + @max_seed_user_id",
			pgx.NamedArgs{"max_seed_user_id": apitest.SeedSuperUser.ID}).Scan(&userID); err != nil {
			return err
		}

		if err := conn.QueryRow(ctx,
			"INSERT INTO pennsieve.users (id, email, node_id, is_super_admin) VALUES (@id, @email, @node_id, @is_super_admin) RETURNING id",
			pgx.NamedArgs{"id": userID, "email": testUser.Email, "node_id": testUser.NodeID, "is_super_admin": testUser.IsSuperAdmin}).
			Scan(&returnedID); err != nil {
			return err
		}
		return nil
	}))

	require.Equal(t, userID, returnedID)
	testUser.ID = &returnedID
}
