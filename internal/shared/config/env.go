package config

import (
	"fmt"
	"os"
	"strconv"
)

// EnvironmentSetting represents a setting that comes from an environment variable (Key)
// with an optional default value *Default
type EnvironmentSetting struct {
	Key     string
	Default *string
}

// NewEnvironmentSetting returns an EnvironmentSetting for env var 'key' with no default value.
func NewEnvironmentSetting(key string) EnvironmentSetting {
	return EnvironmentSetting{Key: key}
}

// NewEnvironmentSettingWithDefault returns an EnvironmentSetting for env var 'key' with a default value of 'defaultValue'.
func NewEnvironmentSettingWithDefault(key string, defaultValue string) EnvironmentSetting {
	return EnvironmentSetting{
		Key:     key,
		Default: &defaultValue,
	}
}

// Get returns the value of the environment variable Key if it is set.
// If Key is not set, Get will return *Default if Default != nil.
// Otherwise, returns an error.
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

// GetNillable returns the value of the environment variable Key if it is set.
// If Key is not set, Get will return *Default.
func (e EnvironmentSetting) GetNillable() *string {
	value, exists := os.LookupEnv(e.Key)
	if !exists {
		return e.Default
	}
	return &value
}

// GetInt returns a value in the same way as Get, but converted from string to int.
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

// GetInt64 returns a value in the same way as Get, but converted from string to int64.
func (e EnvironmentSetting) GetInt64() (int64, error) {
	valueStr, err := e.Get()
	if err != nil {
		return 0, err
	}
	value, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("error converting '%s' value '%s' to int64: %w", e.Key, valueStr, err)
	}
	return value, nil
}

// GetInt32 returns a value in the same way as Get, but converted from string to int64.
func (e EnvironmentSetting) GetInt32() (int32, error) {
	valueStr, err := e.Get()
	if err != nil {
		return 0, err
	}
	value, err := strconv.ParseInt(valueStr, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("error converting '%s' value '%s' to int32: %w", e.Key, valueStr, err)
	}
	return int32(value), nil
}
