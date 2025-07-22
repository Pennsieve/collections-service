package config

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestSSMSetting_Key tests that key is correctly constructed. Initially we were missing the leading '/' in the key
func TestSSMSetting_Key(t *testing.T) {

	env, service, name := "test-env", "test-service", "test-param-name"

	setting := NewSSMSetting(service, name).WithEnvironment(env)

	expectedKey := fmt.Sprintf("/%s/%s/%s", env, service, name)
	expectedValue := uuid.NewString()
	actualValue, err := setting.Load(context.Background(), func(ctx context.Context, key string) (string, error) {
		assert.Equal(t, expectedKey, key)
		return expectedValue, nil
	})

	require.NoError(t, err)
	assert.Equal(t, expectedValue, actualValue)
}
