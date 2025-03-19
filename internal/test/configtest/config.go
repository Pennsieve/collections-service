package configtest

import "github.com/pennsieve/collections-service/internal/shared/config"

// PostgresDBConfig returns a config.PostgresDBConfig suitable for use against
// the pennseivedb instance started for testing. It is preferred in tests over
// calling config.LoadPostgresDBConfig() because that method
// will not create the correct configs if the tests are running locally instead
// of in the Docker test container.
func PostgresDBConfig() config.PostgresDBConfig {
	return config.NewPostgresDBConfigBuilder().
		WithPostgresUser("postgres").
		WithPostgresPassword("password").
		Build()
}
