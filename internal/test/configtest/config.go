package configtest

import "github.com/pennsieve/collections-service/internal/shared/config"

func PostgresDBConfig() config.PostgresDBConfig {
	return config.NewPostgresDBConfigBuilder().
		WithPostgresUser("postgres").
		WithPostgresPassword("password").
		Build()
}
