package dbmigratetest

import (
	collectionsconfig "github.com/pennsieve/collections-service/internal/dbmigrate"
	sharedconfig "github.com/pennsieve/collections-service/internal/shared/config"
	"github.com/pennsieve/collections-service/internal/test/configtest"
	"github.com/pennsieve/dbmigrate-go/pkg/shared/config"
)

// Config returns a [config.Config] suitable for use against
// the pennseivedb instance started for testing. It is preferred in tests over
// calling [config.LoadConfig] because that method
// will not create the correct configs if the tests are running locally instead
// of in the Docker test container.
func Config(pgOptions ...configtest.PostgresOption) config.Config {
	return config.Config{
		PostgresDB:     DBMigratePostgresDBConfig(pgOptions...),
		VerboseLogging: true,
	}
}

func DBMigratePostgresDBConfig(pgOptions ...configtest.PostgresOption) config.PostgresDBConfig {
	defaults := collectionsconfig.ConfigDefaults()
	localconfig := configtest.PostgresDBConfig(pgOptions...)
	return config.PostgresDBConfig{
		Host:     localconfig.Host,
		Port:     localconfig.Port,
		User:     localconfig.User,
		Password: localconfig.Password,
		Database: localconfig.CollectionsDatabase,
		Schema:   defaults["POSTGRES_SCHEMA"],
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
