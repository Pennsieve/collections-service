package config

import (
	"log"
	"os"
	"strconv"
)

const CollectionsSchemaName = "collections"

type Config struct {
	PostgresDB PostgresDBConfig
}

type PostgresDBConfig struct {
	Host                string
	Port                int
	User                string
	Password            *string
	CollectionsDatabase string
}

func LoadPostgresDBConfig() PostgresDBConfig {
	return PostgresDBConfig{
		Host:                GetEnvOrDefault("POSTGRES_HOST", "localhost"),
		Port:                Atoi(GetEnvOrDefault("POSTGRES_PORT", "5432")),
		User:                getEnv("POSTGRES_USER"),
		Password:            getEnvOrNil("POSTGRES_PASSWORD"),
		CollectionsDatabase: GetEnvOrDefault("POSTGRES_COLLECTIONS_DATABASE", "collections_postgres"),
	}
}

func LoadConfig() Config {
	return Config{
		PostgresDB: LoadPostgresDBConfig(),
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
