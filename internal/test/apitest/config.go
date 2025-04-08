package apitest

import (
	"github.com/pennsieve/collections-service/internal/api/config"
	sharedconfig "github.com/pennsieve/collections-service/internal/shared/config"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/configtest"
)

// Config returns a config.Config suitable for use against
// the pennseivedb instance started for testing. It is preferred in tests over
// calling config.LoadConfig() because that method
// will not create the correct configs if the tests are running locally instead
// of in the Docker test container.
func Config() config.Config {
	return config.Config{
		PostgresDB: configtest.PostgresDBConfig(),
	}
}

func PennsieveConfig(discoverServiceHost string) config.PennsieveConfig {
	return config.NewPennsieveConfigBuilder().
		WithDiscoverServiceHost(discoverServiceHost).
		WithDOIPrefix(test.PennsieveDOIPrefix).
		Build()
}

func PennsieveConfigWithFakeHost() config.PennsieveConfig {
	return config.NewPennsieveConfigBuilder().
		WithDiscoverServiceHost("http://example.com/discover").
		WithDOIPrefix(test.PennsieveDOIPrefix).
		Build()
}

type ConfigBuilder struct {
	c *config.Config
}

func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{c: &config.Config{}}
}

func (b *ConfigBuilder) WithPostgresDBConfig(postgresDBConfig sharedconfig.PostgresDBConfig) *ConfigBuilder {
	b.c.PostgresDB = postgresDBConfig
	return b
}

func (b *ConfigBuilder) WithDockerPostgresDBConfig() *ConfigBuilder {
	return b.WithPostgresDBConfig(configtest.PostgresDBConfig())
}

func (b *ConfigBuilder) WithPennsieveConfig(pennsieveConfig config.PennsieveConfig) *ConfigBuilder {
	b.c.PennsieveConfig = pennsieveConfig
	return b
}

func (b *ConfigBuilder) Build() config.Config {
	return *b.c
}
