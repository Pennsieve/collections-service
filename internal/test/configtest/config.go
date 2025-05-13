package configtest

import (
	"github.com/pennsieve/collections-service/internal/api/config"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/stretchr/testify/require"
)

// PostgresDBConfig returns a config.PostgresDBConfig suitable for use against
// the pennseivedb instance started for testing. It is preferred in tests over
// calling config.LoadPostgresDBConfig() because that method
// will not create the correct configs if the tests are running locally instead
// of in the Docker test container.
func PostgresDBConfig(t require.TestingT, options ...config.PostgresDBOption) config.PostgresDBConfig {
	test.Helper(t)
	opts := []config.PostgresDBOption{
		config.WithPostgresUser("postgres"),
		config.WithPostgresPassword("password"),
		config.WithCollectionsDatabase("postgres"),
	}
	opts = append(opts, options...)
	postgresConfig, err := config.LoadPostgresDBConfig(opts...)
	require.NoError(t, err)
	return postgresConfig
}
