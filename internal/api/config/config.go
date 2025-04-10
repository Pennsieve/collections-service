package config

import (
	sharedconfig "github.com/pennsieve/collections-service/internal/shared/config"
)

const MaxBannersPerCollection = 4

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
