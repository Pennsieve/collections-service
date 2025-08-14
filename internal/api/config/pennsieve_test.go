package config_test

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/config"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
)

func TestPennsieveConfig_Load(t *testing.T) {
	expectedDiscoverHost := uuid.NewString()
	expectedPennsieveDOIPrefix := uuid.NewString()
	expectedCollectionsIDSpaceID := apitest.CollectionsIDSpaceID
	expectedCollectionsIDSpaceName := apitest.CollectionsIDSpaceName
	expectedPublishBucket := uuid.NewString()

	t.Setenv(config.DiscoverServiceHostKey, expectedDiscoverHost)
	t.Setenv(config.PennsieveDOIPrefixKey, expectedPennsieveDOIPrefix)
	t.Setenv(config.CollectionsIDSpaceIDKey, strconv.FormatInt(int64(expectedCollectionsIDSpaceID), 10))
	t.Setenv(config.CollectionsIDSpaceNameKey, expectedCollectionsIDSpaceName)
	t.Setenv(config.PublishBucketKey, expectedPublishBucket)

	expectedEnvironment := uuid.NewString()
	actualConfig, err := config.NewPennsieveConfig().Load(expectedEnvironment)
	require.NoError(t, err)

	assert.Equal(t, fmt.Sprintf("https://%s", expectedDiscoverHost), actualConfig.DiscoverServiceURL)
	assert.Equal(t, expectedPennsieveDOIPrefix, actualConfig.DOIPrefix)
	assert.Equal(t, expectedCollectionsIDSpaceID, actualConfig.CollectionsIDSpace.ID)
	assert.Equal(t, expectedCollectionsIDSpaceName, actualConfig.CollectionsIDSpace.Name)
	assert.Equal(t, expectedPublishBucket, actualConfig.PublishBucket)

	assert.NotNil(t, actualConfig.JWTSecretKey)

	if assert.NotNil(t, actualConfig.JWTSecretKey.Environment) {
		assert.Equal(t, expectedEnvironment, *actualConfig.JWTSecretKey.Environment)
	}
	assert.Equal(t, config.ServiceName, actualConfig.JWTSecretKey.Service)
	assert.Equal(t, config.JWTSecretKeySSMName, actualConfig.JWTSecretKey.Name)
}
