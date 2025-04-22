package dbmigrate_test

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pennsieve/collections-service/internal/dbmigrate"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/configtest"
	"github.com/pennsieve/collections-service/internal/test/dbmigratetest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"net"
	"strconv"
	"testing"
	"time"
)

func TestCollectionsMigrator(t *testing.T) {
	tests := []struct {
		scenario string
		tstFunc  func(t *testing.T, migrator *dbmigrate.CollectionsMigrator, verificationConn *pgx.Conn)
	}{
		{"test up and collections created_at and updated_at", testUp},
		{"Down runs without error", testDown},
		{"prevent empty name", testPreventEmptyName},
		{"prevent all white space name", testPreventWhiteSpaceName},
		{"prevent empty DOI", testPreventEmptyDOI},
	}

	// Set up testcontainer that will be used by all tests.
	// Using a self-contained container since we can't use the shared pennsievedb-collections container for these
	// migration tests.
	// Also, so that we don't have to start a pre-collections pennsievedb seed in docker compose only for these tests
	ctx := context.Background()

	containerReq := testcontainers.ContainerRequest{
		Image:        "pennsieve/pennsievedb:V20241120161735-seed",
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithStartupTimeout(5 * time.Second),
	}

	pennsievedb, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: containerReq,
		Started:          true,
	})
	testcontainers.CleanupContainer(t, pennsievedb)
	require.NoError(t, err)

	hostPort, err := pennsievedb.Endpoint(ctx, "")
	require.NoError(t, err)

	host, portStr, err := net.SplitHostPort(hostPort)
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)

	migrateConfig := dbmigratetest.Config(
		configtest.WithHost(host),
		configtest.WithPort(port),
	)

	for _, tt := range tests {
		t.Run(tt.scenario, func(t *testing.T) {
			// make a migrator for each test and pass it into the function so that
			// we can take care of cleaning it up here
			migrator, err := dbmigrate.NewLocalCollectionsMigrator(ctx, migrateConfig)
			require.NoError(t, err)

			// also pass in a plain pgx.Conn to let the test function run any verifications on the migrated schema
			verificationConn, err := test.NewPostgresDBFromConfig(t, migrateConfig.PostgresDB).Connect(ctx, migrateConfig.PostgresDB.CollectionsDatabase)
			require.NoError(t, err)

			t.Cleanup(func() {
				require.NoError(t, migrator.Drop())
				dbmigratetest.Close(t, migrator)
				test.CloseConnection(ctx, t, verificationConn)
			})

			tt.tstFunc(t, migrator, verificationConn)
		})
	}
}

func testUp(t *testing.T, migrator *dbmigrate.CollectionsMigrator, verificationConn *pgx.Conn) {

	require.NoError(t, migrator.Up())

	ctx := context.Background()
	var id int64
	var createdAt, updatedAt time.Time
	require.NoError(t,
		verificationConn.QueryRow(ctx,
			"INSERT INTO collections.collections (name, description, node_id) VALUES (@name, @description, @node_id) RETURNING id, created_at, updated_at",
			pgx.NamedArgs{
				"name":        uuid.NewString(),
				"description": uuid.NewString(),
				"node_id":     uuid.NewString()}).
			Scan(&id, &createdAt, &updatedAt),
	)
	assert.False(t, createdAt.IsZero())
	assert.False(t, updatedAt.IsZero())

	var updatedUpdatedAt time.Time
	require.NoError(t,
		verificationConn.QueryRow(ctx,
			"UPDATE collections.collections SET description = @description WHERE id = @id RETURNING updated_at",
			pgx.NamedArgs{
				"description": uuid.NewString(),
				"id":          id,
			}).
			Scan(&updatedUpdatedAt),
	)
	assert.False(t, updatedUpdatedAt.IsZero())
	assert.False(t, updatedAt.Equal(updatedUpdatedAt))
}

// We don't really use the Down() method for real. Test is here so that
// if we do write 'down' files something checks that they at least run
// without error.
func testDown(t *testing.T, migrator *dbmigrate.CollectionsMigrator, _ *pgx.Conn) {

	require.NoError(t, migrator.Up())

	require.NoError(t, migrator.Down())
}

func testPreventEmptyName(t *testing.T, migrator *dbmigrate.CollectionsMigrator, verificationConn *pgx.Conn) {
	require.NoError(t, migrator.Up())

	ctx := context.Background()

	_, err := verificationConn.Exec(ctx,
		"INSERT INTO collections.collections (name, description, node_id) VALUES (@name, @description, @node_id)",
		pgx.NamedArgs{
			"name":        "",
			"description": uuid.NewString(),
			"node_id":     uuid.NewString()},
	)
	require.Error(t, err)

	emptyNameRows, err := verificationConn.Query(ctx, "SELECT id FROM collections.collections WHERE name = ''")
	require.NoError(t, err)

	emptyNameIDs, err := pgx.CollectRows(emptyNameRows, pgx.RowTo[int64])
	require.NoError(t, err)
	assert.Empty(t, emptyNameIDs)

}

func testPreventWhiteSpaceName(t *testing.T, migrator *dbmigrate.CollectionsMigrator, verificationConn *pgx.Conn) {
	require.NoError(t, migrator.Up())

	ctx := context.Background()

	whiteSpaceName := "   "
	_, err := verificationConn.Exec(ctx,
		"INSERT INTO collections.collections (name, description, node_id) VALUES (@name, @description, @node_id)",
		pgx.NamedArgs{
			"name":        whiteSpaceName,
			"description": uuid.NewString(),
			"node_id":     uuid.NewString()},
	)
	require.Error(t, err)

	emptyNameRows, err := verificationConn.Query(ctx, "SELECT id FROM collections.collections WHERE name = @white_space_name",
		pgx.NamedArgs{"white_space_name": whiteSpaceName})
	require.NoError(t, err)

	emptyNameIDs, err := pgx.CollectRows(emptyNameRows, pgx.RowTo[int64])
	require.NoError(t, err)
	assert.Empty(t, emptyNameIDs)

}

func testPreventEmptyDOI(t *testing.T, migrator *dbmigrate.CollectionsMigrator, verificationConn *pgx.Conn) {
	require.NoError(t, migrator.Up())

	ctx := context.Background()

	var collectionID int64
	err := verificationConn.QueryRow(ctx,
		"INSERT INTO collections.collections (name, description, node_id) VALUES (@name, @description, @node_id) RETURNING id",
		pgx.NamedArgs{
			"name":        uuid.NewString(),
			"description": uuid.NewString(),
			"node_id":     uuid.NewString()},
	).Scan(&collectionID)
	require.NoError(t, err)

	_, err = verificationConn.Exec(ctx,
		"INSERT INTO collections.dois (collection_id, doi) VALUES (@collection_id, @doi)",
		pgx.NamedArgs{
			"collection_id": collectionID,
			"doi":           ""},
	)
	require.Error(t, err)

	emptyDOIRows, err := verificationConn.Query(ctx, "SELECT id FROM collections.dois WHERE doi = ''")
	require.NoError(t, err)

	emptyDOIIDs, err := pgx.CollectRows(emptyDOIRows, pgx.RowTo[int64])
	require.NoError(t, err)
	assert.Empty(t, emptyDOIIDs)

}
