package config

import (
	"fmt"
	sharedconfig "github.com/pennsieve/collections-service/internal/shared/config"
	"strings"
)

type PennsieveConfig struct {
	DiscoverServiceURL string
	DOIPrefix          string
	JWTSecretKey       *sharedconfig.SSMSetting
	CollectionsIDSpace CollectionsPublishingIDSpace
	PublishBucket      string
}

type CollectionsPublishingIDSpace struct {
	ID   int64
	Name string
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

func WithCollectionsIDSpace(id int64, name string) PennsieveOption {
	return func(pennsieveConfig *PennsieveConfig) {
		pennsieveConfig.CollectionsIDSpace = CollectionsPublishingIDSpace{
			ID:   id,
			Name: name,
		}
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
	if c.CollectionsIDSpace.ID == 0 {
		idSpaceID, err := settings.CollectionsIDSpaceID.GetInt64()
		if err != nil {
			return PennsieveConfig{}, err
		}
		idSpaceName, err := settings.CollectionsIDSpaceName.Get()
		if err != nil {
			return PennsieveConfig{}, err
		}
		c.CollectionsIDSpace = CollectionsPublishingIDSpace{
			ID:   idSpaceID,
			Name: idSpaceName,
		}
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
