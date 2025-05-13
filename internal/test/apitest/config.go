package apitest

import (
	"github.com/pennsieve/collections-service/internal/api/config"
	config2 "github.com/pennsieve/collections-service/internal/shared/config"
)

type ConfigBuilder struct {
	c *config.Config
}

func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{c: &config.Config{}}
}

func (b *ConfigBuilder) WithPostgresDBConfig(postgresDBConfig config2.PostgresDBConfig) *ConfigBuilder {
	b.c.PostgresDB = postgresDBConfig
	return b
}

func (b *ConfigBuilder) WithPennsieveConfig(pennsieveConfig config.PennsieveConfig) *ConfigBuilder {
	b.c.PennsieveConfig = pennsieveConfig
	return b
}

func (b *ConfigBuilder) Build() config.Config {
	return *b.c
}

func PennsieveConfig(discoverServiceURL string) config.PennsieveConfig {
	return config.NewPennsieveConfig(config.WithDiscoverServiceURL(discoverServiceURL),
		config.WithDOIPrefix(PennsieveDOIPrefix))
}

func PennsieveConfigWithFakeURL() config.PennsieveConfig {
	return config.NewPennsieveConfig(
		config.WithDiscoverServiceURL("http://example.com/discover"),
		config.WithDOIPrefix(PennsieveDOIPrefix))
}
