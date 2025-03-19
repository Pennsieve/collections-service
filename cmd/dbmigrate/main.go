package main

import (
	"context"
	"fmt"
	awsCfg "github.com/aws/aws-sdk-go-v2/config"
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
	logger.
		With(slog.Bool("verboseLogging", migrateConfig.VerboseLogging),
			slog.Group("postgres",
				slog.String("host", migrateConfig.PostgresDB.Host),
				slog.Int("port", migrateConfig.PostgresDB.Port),
				slog.String("username", migrateConfig.PostgresDB.User),
				slog.Bool("useRDSProxy", migrateConfig.PostgresDB.Password == nil),
				slog.String("database", migrateConfig.PostgresDB.CollectionsDatabase),
			)).
		Info("collections DB schema migration started")
	m, err := newCollectionsManager(ctx, migrateConfig)
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

func newCollectionsManager(ctx context.Context, migrateConfig dbmigrate.Config) (*dbmigrate.CollectionsMigrator, error) {
	if migrateConfig.PostgresDB.Password == nil {
		awsConfig, err := awsCfg.LoadDefaultConfig(ctx)
		if err != nil {
			return nil, fmt.Errorf("error loading AWS config: %w", err)
		}
		return dbmigrate.NewRDSProxyCollectionsMigrator(ctx, migrateConfig, awsConfig)
	}
	return dbmigrate.NewLocalCollectionsMigrator(ctx, migrateConfig)
}
