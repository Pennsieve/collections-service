package config

import (
	"fmt"
	"os"
	"strconv"
)

func GetEnv(key string) (string, error) {
	value, exists := os.LookupEnv(key)

	if !exists {
		return "", fmt.Errorf("failed to load '%s' from environment", key)
	}

	return value, nil
}

func GetEnvOrDefault(key string, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	} else {
		return defaultValue
	}
}

func GetIntEnvOrDefault(key string, defaultValue string) (int, error) {
	valueStr := GetEnvOrDefault(key, defaultValue)
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, fmt.Errorf("failed to convert '%s' value '%s' to int: %w", key, valueStr, err)
	}
	return value, nil
}

func GetEnvOrNil(key string) *string {
	if value, exists := os.LookupEnv(key); exists {
		return &value
	} else {
		return nil
	}
}
