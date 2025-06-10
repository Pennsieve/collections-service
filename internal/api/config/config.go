package config

import (
	"fmt"
	sharedconfig "github.com/pennsieve/collections-service/internal/shared/config"
)

const MaxBannersPerCollection = 4
const EnvironmentKey = "ENV"

type Config struct {
	Environment     string
	PostgresDB      sharedconfig.PostgresDBConfig
	PennsieveConfig PennsieveConfig
}

func LoadConfig() (Config, error) {
	environment, err := sharedconfig.NewEnvironmentSetting(EnvironmentKey).Get()
	if err != nil {
		return Config{}, err
	}
	postgresConfig, err := sharedconfig.NewPostgresDBConfig().Load()
	if err != nil {
		return Config{}, fmt.Errorf("error loading PostgresDB config: %w", err)
	}
	pennsieveConfig, err := NewPennsieveConfig().Load(environment)
	if err != nil {
		return Config{}, fmt.Errorf("error loading Pennsieve config: %w", err)
	}
	return Config{
		Environment:     environment,
		PostgresDB:      postgresConfig,
		PennsieveConfig: pennsieveConfig,
	}, nil
}
