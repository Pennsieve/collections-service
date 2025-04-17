package configtest

import (
	"github.com/pennsieve/collections-service/internal/shared/config"
)

// PostgresDBConfig returns a config.PostgresDBConfig suitable for use against
// the pennseivedb instance started for testing. It is preferred in tests over
// calling config.LoadPostgresDBConfig() because that method
// will not create the correct configs if the tests are running locally instead
// of in the Docker test container.
func PostgresDBConfig(options ...PostgresOption) config.PostgresDBConfig {
	builder := config.NewPostgresDBConfigBuilder().
		WithPostgresUser("postgres").
		WithPostgresPassword("password")
	for _, option := range options {
		builder = option(builder)
	}
	return builder.Build()
}

type PostgresOption func(builder *config.PostgresDBConfigBuilder) *config.PostgresDBConfigBuilder

func WithPort(port int) PostgresOption {
	return func(builder *config.PostgresDBConfigBuilder) *config.PostgresDBConfigBuilder {
		return builder.WithPort(port)
	}
}

func WithHost(host string) PostgresOption {
	return func(builder *config.PostgresDBConfigBuilder) *config.PostgresDBConfigBuilder {
		return builder.WithHost(host)
	}
}
