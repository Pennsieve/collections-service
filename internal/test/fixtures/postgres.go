package fixtures

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/pennsieve/collections-service/internal/api/publishing"
	"github.com/pennsieve/collections-service/internal/api/store/collections"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/userstest"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"github.com/stretchr/testify/require"
	"strings"
)

func GetCollection(ctx context.Context, t require.TestingT, conn *pgx.Conn, collectionID int64) collections.Collection {
	test.Helper(t)
	rows, err := conn.Query(ctx, "SELECT * from collections.collections where id = @id", pgx.NamedArgs{"id": collectionID})
	require.NoError(t, err)
	collection, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[collections.Collection])
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
func GetCollectionByNodeID(ctx context.Context, t require.TestingT, conn *pgx.Conn, collectionNodeID string) collections.Collection {
	test.Helper(t)
	rows, err := conn.Query(ctx, "SELECT * from collections.collections where node_id = @nodeId", pgx.NamedArgs{"nodeId": collectionNodeID})
	require.NoError(t, err)
	collection, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[collections.Collection])
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

func GetCollectionUsers(ctx context.Context, t require.TestingT, conn *pgx.Conn, collectionID int64) (userIDToCollectionUser map[int64]collections.CollectionUser) {
	test.Helper(t)
	rows, err := conn.Query(ctx,
		"SELECT * FROM collections.collection_user WHERE collection_id = @collection_id",
		pgx.NamedArgs{"collection_id": collectionID})
	require.NoError(t, err)
	collectionUsers, err := pgx.CollectRows(rows, pgx.RowToStructByName[collections.CollectionUser])
	require.NoError(t, err)
	userIDToCollectionUser = make(map[int64]collections.CollectionUser, len(collectionUsers))
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
		"role":           collections.PgxRole(permission.ToRole()),
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
		args[roleKey] = collections.PgxRole(permission.ToRole())
	}

	tag, err := conn.Exec(ctx,
		fmt.Sprintf(`INSERT INTO collections.collection_user (collection_id, user_id, permission_bit, role) VALUES %s`, strings.Join(values, ",")),
		args)
	require.NoError(t, err, "error adding users to collection %d", collectionID)
	require.Equal(t, int64(len(userIDToPermission)), tag.RowsAffected(), "expected to add %d users, but added %d", len(userIDToPermission), tag.RowsAffected())
}

func GetDOIs(ctx context.Context, t require.TestingT, conn *pgx.Conn, collectionID int64) (doiToDOI map[string]collections.CollectionDOI) {
	test.Helper(t)
	rows, err := conn.Query(ctx,
		"SELECT * FROM collections.dois WHERE collection_id = @collection_id",
		pgx.NamedArgs{"collection_id": collectionID})
	require.NoError(t, err)
	dois, err := pgx.CollectRows(rows, pgx.RowToStructByName[collections.CollectionDOI])
	require.NoError(t, err)
	doiToDOI = make(map[string]collections.CollectionDOI, len(dois))
	for _, doi := range dois {
		doiToDOI[doi.DOI] = doi
	}
	return
}

func CreateTestUser(ctx context.Context, t require.TestingT, conn *pgx.Conn, testUser *userstest.TestUser) {
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
			pgx.NamedArgs{"max_seed_user_id": userstest.SeedSuperUser.ID}).Scan(&userID); err != nil {
			return err
		}

		userArgs := pgx.NamedArgs{
			"id":                  userID,
			"email":               testUser.Email,
			"node_id":             testUser.NodeID,
			"is_super_admin":      testUser.IsSuperAdmin,
			"first_name":          testUser.FirstName,
			"last_name":           testUser.LastName,
			"orcid_authorization": testUser.ORCIDAuthorization,
			"middle_initial":      testUser.MiddleInitial,
			"degree":              testUser.Degree,
		}

		if err := conn.QueryRow(ctx,
			`INSERT INTO pennsieve.users (id, email, node_id, is_super_admin, first_name, last_name, orcid_authorization, middle_initial, degree) 
                                      VALUES (@id, @email, @node_id, @is_super_admin, @first_name, @last_name, @orcid_authorization, @middle_initial, @degree) RETURNING id`,
			userArgs).
			Scan(&returnedID); err != nil {
			return err
		}
		return nil
	}))

	require.Equal(t, userID, returnedID)
	testUser.ID = &returnedID
}

func AddPublishStatus(ctx context.Context, t require.TestingT, conn *pgx.Conn, status publishing.PublishStatus) {
	query := `INSERT INTO collections.publish_status (collection_id, status, type, started_at, finished_at, user_id) 
                                              VALUES (@collection_id, @status, @type, @started_at, @finished_at, @user_id)`
	args := pgx.NamedArgs{
		"collection_id": status.CollectionID,
		"status":        status.Status,
		"type":          status.Type,
		"started_at":    status.StartedAt,
		"finished_at":   status.FinishedAt,
		"user_id":       status.UserID,
	}
	tag, err := conn.Exec(ctx, query, args)
	require.NoError(t, err, "error inserting publish_status row: %+v", status)
	require.Equal(t, int64(1), tag.RowsAffected())
}

func GetPublishStatus(ctx context.Context, t require.TestingT, conn *pgx.Conn, collectionID int64) publishing.PublishStatus {
	query := `SELECT collection_id, status, type, started_at, finished_at, user_id
                FROM collections.publish_status WHERE collection_id = @collection_id`
	args := pgx.NamedArgs{"collection_id": collectionID}

	rows, _ := conn.Query(ctx, query, args)
	publishStatus, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[publishing.PublishStatus])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			require.FailNow(t, "no publish status found for collection with id", "id = %d", collectionID)
		} else if errors.Is(err, pgx.ErrTooManyRows) {
			require.FailNow(t, "more than one publish status found for collection with id", "id = %d", collectionID)
		} else {
			require.NoError(t, err)
		}
	}
	return publishStatus
}
