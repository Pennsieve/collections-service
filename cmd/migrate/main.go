package main

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/pennsieve/collections-service/internal/migrate"
	"log/slog"
	"os"
)

var logger = slog.Default()

func main() {
	logger.Info("collections DB schema migration started")
	ctx := context.Background()
	migrateConfig, err := migrate.LoadConfig()
	if err != nil {
		logger.Error("error loading config", slog.Any("error", err))
		os.Exit(1)
	}
	m, err := migrate.NewRDSProxyCollectionsMigrator(ctx, migrateConfig, aws.Config{})
	if err != nil {
		logger.Error("error creating CollectionsMigrator", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		sourceErr, dbErr := m.Close()
		if sourceErr != nil {
			logger.Warn("error with source when closing migrator", slog.Any("error", sourceErr))
		}
		if dbErr != nil {
			logger.Warn("error with database when closing migrator", slog.Any("error", dbErr))
		}
	}()

	if err := m.Up(); err != nil {
		logger.Error("error running up migrations", slog.Any("error", err))
		os.Exit(1)
	}

	logger.Info("collections DB schema migration complete")
}
