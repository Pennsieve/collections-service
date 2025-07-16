package config

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
)

func TestPennsieveConfig_Load(t *testing.T) {
	expectedDiscoverHost := uuid.NewString()
	expectedPennsieveDOIPrefix := uuid.NewString()
	expectedCollectionNamespaceID := int64(-20)
	expectedPublishBucket := uuid.NewString()

	t.Setenv(DiscoverServiceHostKey, expectedDiscoverHost)
	t.Setenv(PennsieveDOIPrefixKey, expectedPennsieveDOIPrefix)
	t.Setenv(CollectionNamespaceIDKey, strconv.FormatInt(expectedCollectionNamespaceID, 10))
	t.Setenv(PublishBucketKey, expectedPublishBucket)

	expectedEnvironment := uuid.NewString()
	config, err := NewPennsieveConfig().Load(expectedEnvironment)
	require.NoError(t, err)

	assert.Equal(t, fmt.Sprintf("https://%s", expectedDiscoverHost), config.DiscoverServiceURL)
	assert.Equal(t, expectedPennsieveDOIPrefix, config.DOIPrefix)
	assert.Equal(t, expectedCollectionNamespaceID, config.CollectionNamespaceID)
	assert.Equal(t, expectedPublishBucket, config.PublishBucket)

	assert.NotNil(t, config.JWTSecretKey)

	if assert.NotNil(t, config.JWTSecretKey.Environment) {
		assert.Equal(t, expectedEnvironment, *config.JWTSecretKey.Environment)
	}
	assert.Equal(t, ServiceName, config.JWTSecretKey.Service)
	assert.Equal(t, JWTSecretKeySSMName, config.JWTSecretKey.Name)
}
