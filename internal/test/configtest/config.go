package configtest

import (
	"github.com/pennsieve/collections-service/internal/shared/config"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/stretchr/testify/require"
)

// PostgresDBConfig returns a config.PostgresDBConfig suitable for use against
// the pennseivedb instance started by Docker Compose for testing. This config
// will work in both the CI case where tests are run in Docker and the local
// case where go test is run directly.
func PostgresDBConfig(t require.TestingT, options ...config.PostgresDBOption) config.PostgresDBConfig {
	test.Helper(t)
	postgresConfig, err := config.NewPostgresDBConfig(options...).LoadWithEnvSettings(TestPostgresDBEnvironmentSettings)
	require.NoError(t, err)
	return postgresConfig
}

var TestPostgresDBEnvironmentSettings = config.PostgresDBEnvironmentSettings{
	Host:                config.NewEnvironmentSettingWithDefault(config.PostgresHostKey, "localhost"),
	Port:                config.NewEnvironmentSettingWithDefault(config.PostgresPortKey, config.DefaultPostgresPort),
	User:                config.NewEnvironmentSettingWithDefault(config.PostgresUserKey, "postgres"),
	Password:            config.NewEnvironmentSettingWithDefault(config.PostgresPasswordKey, "password"),
	CollectionsDatabase: config.NewEnvironmentSettingWithDefault(config.PostgresCollectionsDatabaseKey, "postgres"),
}
