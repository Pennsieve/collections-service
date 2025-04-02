package main

import (
	"context"
	"github.com/pennsieve/collections-service/internal/dbmigrate"
	"log/slog"
	"os"
)

var logger = slog.Default()

func main() {
	ctx := context.Background()
	migrateConfig, err := dbmigrate.LoadConfig()
	if err != nil {
		logger.Error("error loading config", slog.Any("error", err))
		os.Exit(1)
	}
	if migrateConfig.PostgresDB.Password == nil {
		logger.Error("password must be specified; cannot currently use RDS proxy for migrates since no Postgres role with the appropriate grants has credentials in the proxy")
		os.Exit(1)
	}
	logger.
		With(slog.Bool("verboseLogging", migrateConfig.VerboseLogging),
			slog.Group("postgres",
				slog.String("host", migrateConfig.PostgresDB.Host),
				slog.Int("port", migrateConfig.PostgresDB.Port),
				slog.String("username", migrateConfig.PostgresDB.User),
				slog.String("database", migrateConfig.PostgresDB.CollectionsDatabase),
			)).
		Info("collections DB schema migration started")
	m, err := dbmigrate.NewLocalCollectionsMigrator(ctx, migrateConfig)
	if err != nil {
		logger.Error("error creating CollectionsMigrator", slog.Any("error", err))
		os.Exit(1)
	}
	defer m.CloseAndLogError()

	if err := m.Up(); err != nil {
		logger.Error("error running 'up' migrations", slog.Any("error", err))
		os.Exit(1)
	}

	logger.Info("collections DB schema migration complete")
}
