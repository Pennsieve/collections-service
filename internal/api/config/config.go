package config

import (
	sharedconfig "github.com/pennsieve/collections-service/internal/shared/config"
)

type Config struct {
	PostgresDB      sharedconfig.PostgresDBConfig
	PennsieveConfig PennsieveConfig
}

func LoadConfig() Config {
	return Config{
		PostgresDB:      sharedconfig.LoadPostgresDBConfig(),
		PennsieveConfig: LoadPennsieveConfig(),
	}
}
