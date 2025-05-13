package config

import (
	"fmt"
	config2 "github.com/pennsieve/collections-service/internal/shared/config"
)

const MaxBannersPerCollection = 4

type Config struct {
	PostgresDB      config2.PostgresDBConfig
	PennsieveConfig PennsieveConfig
}

func LoadConfig() (Config, error) {
	postgresConfig, err := config2.NewPostgresDBConfig().Load()
	if err != nil {
		return Config{}, fmt.Errorf("error loading PostgresDB config: %w", err)
	}
	pennsieveConfig, err := LoadPennsieveConfig()
	if err != nil {
		return Config{}, fmt.Errorf("error loading Pennsieve config: %w", err)
	}
	return Config{
		PostgresDB:      postgresConfig,
		PennsieveConfig: pennsieveConfig,
	}, nil
}
