package dbmigrate_test

import (
	"context"
	"github.com/pennsieve/collections-service/internal/dbmigrate"
	"github.com/pennsieve/collections-service/internal/test/dbmigratetest"
	"github.com/stretchr/testify/require"
	"testing"
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
}

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
