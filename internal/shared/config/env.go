package config

import (
	"fmt"
	"os"
	"strconv"
)

type EnvironmentSetting struct {
	Key     string
	Default *string
}

func NewEnvironmentSetting(key string) EnvironmentSetting {
	return EnvironmentSetting{Key: key}
}

func NewEnvironmentSettingWithDefault(key string, defaultValue string) EnvironmentSetting {
	return EnvironmentSetting{
		Key:     key,
		Default: &defaultValue,
	}
}

func (e EnvironmentSetting) Get() (string, error) {
	value, exists := os.LookupEnv(e.Key)
	if !exists {
		if e.Default != nil {
			return *e.Default, nil
		}
		return "", fmt.Errorf("environment variable '%s' is not set and has no default", e.Key)
	}
	return value, nil
}

func (e EnvironmentSetting) GetNillable() *string {
	value, exists := os.LookupEnv(e.Key)
	if !exists {
		return e.Default
	}
	return &value
}

func (e EnvironmentSetting) GetInt() (int, error) {
	valueStr, err := e.Get()
	if err != nil {
		return 0, err
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, fmt.Errorf("error converting '%s' value '%s' to int: %w", e.Key, valueStr, err)
	}
	return value, nil
}

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
