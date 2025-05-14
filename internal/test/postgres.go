package test

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/pennsieve/collections-service/internal/shared/config"
	"github.com/stretchr/testify/require"
)

type PostgresDB struct {
	host     string
	port     int
	user     string
	password string
}

func NewPostgresDBFromConfig(t require.TestingT, pgConfig config.PostgresDBConfig) *PostgresDB {
	Helper(t)
	require.NotNil(t, pgConfig.Password)
	return NewPostgresDB(pgConfig.Host, pgConfig.Port, pgConfig.User, *pgConfig.Password)
}

func NewPostgresDB(host string, port int, user string, password string) *PostgresDB {
	return &PostgresDB{
		host,
		port,
		user,
		password,
	}
}

func (db *PostgresDB) Connect(ctx context.Context, databaseName string) (*pgx.Conn, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s",
		db.host, db.port, db.user, db.password, databaseName,
	)

	return pgx.Connect(ctx, dsn)
}

func CloseConnection(ctx context.Context, t require.TestingT, conn *pgx.Conn) {
	Helper(t)
	require.NoError(t, conn.Close(ctx))
}

// PostgresDBConfig returns a [config.PostgresDBConfig] suitable for use against
// the pennseivedb instance started by Docker Compose for testing. This config
// will work in both the CI case where tests are run in Docker and the local
// case where go test is run directly.
func PostgresDBConfig(t require.TestingT, options ...config.PostgresDBOption) config.PostgresDBConfig {
	Helper(t)
	postgresConfig, err := config.NewPostgresDBConfig(options...).LoadWithEnvSettings(PostgresDBEnvironmentSettings)
	require.NoError(t, err)
	return postgresConfig
}

// PostgresDBEnvironmentSettings are the env settings useful for tests. They are defined
// so that when [config.PostgresDBConfig.LoadWithEnvSettings] is called in tests, the returned [config.PostgresDBConfig]
// will work whether the test is running in Docker (where the env vars are set) or running locally (where the env vars are not set)
var PostgresDBEnvironmentSettings = config.PostgresDBEnvironmentSettings{
	Host:                config.NewEnvironmentSettingWithDefault(config.PostgresHostKey, "localhost"),
	Port:                config.NewEnvironmentSettingWithDefault(config.PostgresPortKey, config.DefaultPostgresPort),
	User:                config.NewEnvironmentSettingWithDefault(config.PostgresUserKey, "postgres"),
	Password:            config.NewEnvironmentSettingWithDefault(config.PostgresPasswordKey, "password"),
	CollectionsDatabase: config.NewEnvironmentSettingWithDefault(config.PostgresCollectionsDatabaseKey, "postgres"),
}
