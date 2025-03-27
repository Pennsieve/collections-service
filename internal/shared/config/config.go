package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	PostgresDB      PostgresDBConfig
	PennsieveConfig PennsieveConfig
}

func LoadConfig() Config {
	return Config{
		PostgresDB:      LoadPostgresDBConfig(),
		PennsieveConfig: LoadPennsieveConfig(),
	}
}

func getEnv(key string) string {
	value, exists := os.LookupEnv(key)

	if !exists {
		log.Fatalf("Failed to load '%s' from environment", key)
	}

	return value
}

func GetEnvOrDefault(key string, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	} else {
		return defaultValue
	}
}

func getEnvOrNil(key string) *string {
	if value, exists := os.LookupEnv(key); exists {
		return &value
	} else {
		return nil
	}
}

func Atoi(value string) int {
	i, err := strconv.Atoi(value)

	if err != nil {
		log.Fatalf("Failed to convert '%s' integer", value)
	}

	return i
}
