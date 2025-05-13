package dbmigratetest

import (
	sharedconfig "github.com/pennsieve/collections-service/internal/api/config"
	collectionsconfig "github.com/pennsieve/collections-service/internal/dbmigrate"
	"github.com/pennsieve/collections-service/internal/test/configtest"
	"github.com/pennsieve/dbmigrate-go/pkg/config"
	"github.com/stretchr/testify/require"
)

// Config returns a [config.Config] suitable for use against
// the pennseivedb instance started for testing. It is preferred in tests over
// calling [config.LoadConfig] because that method
// will not create the correct configs if the tests are running locally instead
// of in the Docker test container.
func Config(t require.TestingT, host string, port int) config.Config {
	return config.Config{
		PostgresDB:     DBMigratePostgresDBConfig(t, host, port),
		VerboseLogging: true,
	}
}

func DBMigratePostgresDBConfig(t require.TestingT, host string, port int) config.PostgresDBConfig {
	defaults := collectionsconfig.ConfigDefaults()
	localconfig := configtest.PostgresDBConfig(t, sharedconfig.WithHost(host), sharedconfig.WithPort(port))
	return config.PostgresDBConfig{
		Host:     localconfig.Host,
		Port:     localconfig.Port,
		User:     localconfig.User,
		Password: localconfig.Password,
		Database: localconfig.CollectionsDatabase,
		Schema:   defaults[config.PostgresSchemaKey],
	}
}

func ToSharedPostgresDBConfig(config config.PostgresDBConfig) sharedconfig.PostgresDBConfig {
	return sharedconfig.PostgresDBConfig{
		Host:                config.Host,
		Port:                config.Port,
		User:                config.User,
		Password:            config.Password,
		CollectionsDatabase: config.Database,
	}
}
