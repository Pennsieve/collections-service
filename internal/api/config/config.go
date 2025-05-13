package config

import "fmt"

const MaxBannersPerCollection = 4

type Config struct {
	PostgresDB      PostgresDBConfig
	PennsieveConfig PennsieveConfig
}

func LoadConfig() (Config, error) {
	postgresConfig, err := LoadPostgresDBConfig()
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
