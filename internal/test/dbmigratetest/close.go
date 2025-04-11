package dbmigratetest

import (
	"github.com/pennsieve/collections-service/internal/dbmigrate"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/stretchr/testify/require"
)

func Close(t require.TestingT, migrator *dbmigrate.CollectionsMigrator) {
	test.Helper(t)
	srcErr, dbErr := migrator.Close()
	require.NoError(t, srcErr)
	require.NoError(t, dbErr)
}
