package config

import (
	"fmt"
	sharedconfig "github.com/pennsieve/collections-service/internal/shared/config"
)

const MaxBannersPerCollection = 4

type Config struct {
	PostgresDB      sharedconfig.PostgresDBConfig
	PennsieveConfig PennsieveConfig
}

func LoadConfig() (Config, error) {
	postgresConfig, err := sharedconfig.NewPostgresDBConfig().Load()
	if err != nil {
		return Config{}, fmt.Errorf("error loading PostgresDB config: %w", err)
	}
	pennsieveConfig, err := NewPennsieveConfig().Load()
	if err != nil {
		return Config{}, fmt.Errorf("error loading Pennsieve config: %w", err)
	}
	return Config{
		PostgresDB:      postgresConfig,
		PennsieveConfig: pennsieveConfig,
	}, nil
}
