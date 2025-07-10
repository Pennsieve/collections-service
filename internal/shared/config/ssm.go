package config

import (
	"context"
	"fmt"
)

type SSMLookupFunc func(ctx context.Context, key string) (string, error)

type SSMSetting struct {
	Environment *string
	Service     string
	Name        string
	Value       *string
}

func NewSSMSetting(service, name string) *SSMSetting {
	return &SSMSetting{
		Service: service,
		Name:    name,
	}
}

func (s *SSMSetting) Load(ctx context.Context, lookup SSMLookupFunc) (string, error) {
	if s.Value == nil {
		if s.Environment == nil {
			return "", fmt.Errorf("environment not set for SSM setting")
		}
		key := fmt.Sprintf("%s/%s/%s", *s.Environment, s.Service, s.Name)
		value, err := lookup(ctx, key)
		if err != nil {
			return "", fmt.Errorf("error getting %s value from SSM: %w", key, err)
		}
		s.Value = &value
	}
	return *s.Value, nil
}

func (s *SSMSetting) String() string {
	if s == nil {
		return "<nil>"
	}
	env := "<environment>"
	if s.Environment != nil {
		env = *s.Environment
	}
	return fmt.Sprintf("%s/%s/%s", env, s.Service, s.Name)
}

func (s *SSMSetting) WithEnvironment(env string) *SSMSetting {
	s.Environment = &env
	return s
}

// WithValue sets the value of this SSMSetting. Probably just for testing
func (s *SSMSetting) WithValue(value string) *SSMSetting {
	s.Value = &value
	return s
}
