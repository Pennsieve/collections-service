package dbmigrate_test

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pennsieve/collections-service/internal/dbmigrate"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/dbmigratetest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestCollectionsMigrator_Up(t *testing.T) {
	ctx := context.Background()

	migrateConfig := dbmigratetest.Config()

	migrator, err := dbmigrate.NewLocalCollectionsMigrator(ctx, migrateConfig)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, migrator.Drop())
	})

	require.NoError(t, migrator.Up())

	conn, err := test.NewPostgresDBFromConfig(t, migrateConfig.PostgresDB).Connect(ctx, migrateConfig.PostgresDB.CollectionsDatabase)
	require.NoError(t, err)

	defer test.CloseConnection(ctx, t, conn)

	var id int64
	var createdAt, updatedAt time.Time
	require.NoError(t,
		conn.QueryRow(ctx,
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
		conn.QueryRow(ctx,
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
func TestCollectionsMigrator_Down(t *testing.T) {
	ctx := context.Background()

	migrateConfig := dbmigratetest.Config()

	migrator, err := dbmigrate.NewLocalCollectionsMigrator(ctx, migrateConfig)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, migrator.Drop())
	})

	require.NoError(t, migrator.Up())

	require.NoError(t, migrator.Down())
}
