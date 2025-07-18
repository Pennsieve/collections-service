package config

import (
	"fmt"
	sharedconfig "github.com/pennsieve/collections-service/internal/shared/config"
	"strings"
)

type PennsieveConfig struct {
	DiscoverServiceURL    string
	DOIPrefix             string
	JWTSecretKey          *sharedconfig.SSMSetting
	CollectionNamespaceID int64
	PublishBucket         string
}

func NewPennsieveConfig(options ...PennsieveOption) PennsieveConfig {
	pennsieveConfig := PennsieveConfig{}
	for _, option := range options {
		option(&pennsieveConfig)
	}
	return pennsieveConfig
}

type PennsieveOption func(pennsieveConfig *PennsieveConfig)

func WithDiscoverServiceURL(url string) PennsieveOption {
	return func(pennsieveConfig *PennsieveConfig) {
		pennsieveConfig.DiscoverServiceURL = url
	}
}

func WithDOIPrefix(doiPrefix string) PennsieveOption {
	return func(pennsieveConfig *PennsieveConfig) {
		pennsieveConfig.DOIPrefix = doiPrefix
	}
}

func WithJWTSecretKey(jwtSecretKey string) PennsieveOption {
	return func(pennsieveConfig *PennsieveConfig) {
		pennsieveConfig.JWTSecretKey = NewJWTSecretKeySetting().WithValue(jwtSecretKey)
	}
}

func WithCollectionNamespaceID(namespaceID int64) PennsieveOption {
	return func(pennsieveConfig *PennsieveConfig) {
		pennsieveConfig.CollectionNamespaceID = namespaceID
	}
}

func WithPublishBucket(publishBucket string) PennsieveOption {
	return func(pennsieveConfig *PennsieveConfig) {
		pennsieveConfig.PublishBucket = publishBucket
	}
}

// LoadWithSettings returns a copy of this PennsieveConfig where any missing fields are populated by the
// given PennsieveSettings.
func (c PennsieveConfig) LoadWithSettings(environmentName string, settings PennsieveSettings) (PennsieveConfig, error) {
	if len(c.DiscoverServiceURL) == 0 {
		url, err := settings.DiscoverServiceHost.Get()
		if err != nil {
			return PennsieveConfig{}, err
		}
		if !strings.HasPrefix(url, "http") {
			url = fmt.Sprintf("https://%s", url)
		}
		c.DiscoverServiceURL = url
	}
	if len(c.DOIPrefix) == 0 {
		prefix, err := settings.DOIPrefix.Get()
		if err != nil {
			return PennsieveConfig{}, err
		}
		c.DOIPrefix = prefix
	}
	if c.CollectionNamespaceID == 0 {
		namespaceID, err := settings.CollectionNamespaceID.GetInt64()
		if err != nil {
			return PennsieveConfig{}, err
		}
		c.CollectionNamespaceID = namespaceID
	}

	if len(c.PublishBucket) == 0 {
		publishBucket, err := settings.PublishBucket.Get()
		if err != nil {
			return PennsieveConfig{}, err
		}
		c.PublishBucket = publishBucket
	}

	if c.JWTSecretKey == nil {
		c.JWTSecretKey = settings.JWTSecretKey.WithEnvironment(environmentName)
	}

	return c, nil
}

// Load returns a copy of this PennsieveConfig where any missing fields are populated by the
// given DeployedPennsieveSettings.
func (c PennsieveConfig) Load(environmentName string) (PennsieveConfig, error) {
	return c.LoadWithSettings(environmentName, DeployedPennsieveSettings)
}
