package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"net"
	"net/url"
)

type CollectionMigrator struct {
	wrapped *migrate.Migrate
}

func NewRDSProxyCollectionsMigrator(ctx context.Context, migrateConfig Config, awsConfig aws.Config) (*CollectionMigrator, error) {
	authenticationToken, err := auth.BuildAuthToken(
		ctx,
		fmt.Sprintf("%s:%d", migrateConfig.PostgresDB.Host, migrateConfig.PostgresDB.Port),
		awsConfig.Region,
		migrateConfig.PostgresDB.User,
		awsConfig.Credentials,
	)
	if err != nil {
		return nil, fmt.Errorf("error building auth token for CollectionsMigrator: %w", err)
	}
	return newCollectionsMigrator(
		migrateConfig.PostgresDB.User,
		authenticationToken,
		migrateConfig.PostgresDB.Host,
		migrateConfig.PostgresDB.Port,
		migrateConfig.PostgresDB.CollectionsDatabase,
		migrateConfig.VerboseLogging)
}

func NewLocalCollectionsMigrator(migrateConfig Config) (*CollectionMigrator, error) {
	if migrateConfig.PostgresDB.Password == nil {
		return nil, fmt.Errorf("password cannot be nil for local CollectionsMigrator")
	}
	return newCollectionsMigrator(
		migrateConfig.PostgresDB.User,
		*migrateConfig.PostgresDB.Password,
		migrateConfig.PostgresDB.Host,
		migrateConfig.PostgresDB.Port,
		migrateConfig.PostgresDB.CollectionsDatabase,
		migrateConfig.VerboseLogging)

}

func (m *CollectionMigrator) Up() error {
	return m.wrapped.Up()
}

func (m *CollectionMigrator) Close() (source error, database error) {
	return m.wrapped.Close()
}

func newCollectionsMigrator(username, password, host string,
	port int,
	databaseName string,
	verboseLogging bool) (*CollectionMigrator, error) {
	db, err := sql.Open("pgx5",
		datasourceName(username,
			password,
			host,
			port,
			databaseName),
	)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}
	driver, err := pgx.WithInstance(db, &pgx.Config{SchemaName: "collections"})
	if err != nil {
		if closeErr := db.Close(); closeErr != nil {
			return nil, fmt.Errorf(
				"error creating migration driver: %w; in addition an error occured when closing DB connection: %v",
				err, closeErr)
		}
		return nil, fmt.Errorf("error creating migration driver: %w", err)
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file:///migrations",
		databaseName, driver)
	if err != nil {
		if closeErr := driver.Close(); closeErr != nil {
			return nil, fmt.Errorf(
				"error creating Migrate instance: %w; in addition an error occured when closing the migrate driver: %v",
				err,
				closeErr)
		}
		return nil, fmt.Errorf("error creating Migrate instance: %w", err)
	}
	m.Log = NewLogger(verboseLogging)
	return &CollectionMigrator{wrapped: m}, nil
}

func datasourceName(username, password, host string, port int, databaseName string) string {
	datasource := url.URL{
		Scheme: "pgx5",
		User:   url.UserPassword(username, password),
		Host:   net.JoinHostPort(host, fmt.Sprintf("%d", port)),
		Path:   databaseName,
	}
	return datasource.String()
}
