package config

import (
	"fmt"
	"strings"
)

type PennsieveConfig struct {
	DiscoverServiceURL string
	DOIPrefix          string
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

// LoadWithEnvSettings returns a copy of this PennsieveConfig where any missing fields are populated by the
// given PennsieveEnvironmentSettings.
func (c PennsieveConfig) LoadWithEnvSettings(environmentSettings PennsieveEnvironmentSettings) (PennsieveConfig, error) {
	if len(c.DiscoverServiceURL) == 0 {
		url, err := environmentSettings.DiscoverServiceHost.Get()
		if err != nil {
			return PennsieveConfig{}, err
		}
		if !strings.HasPrefix(url, "http") {
			url = fmt.Sprintf("https://%s", url)
		}
		c.DiscoverServiceURL = url
	}
	if len(c.DOIPrefix) == 0 {
		prefix, err := environmentSettings.DOIPrefix.Get()
		if err != nil {
			return PennsieveConfig{}, err
		}
		c.DOIPrefix = prefix
	}
	return c, nil
}

// Load returns a copy of this PennsieveConfig where any missing fields are populated by the
// given DeployedPennsieveEnvironmentSettings.
func (c PennsieveConfig) Load() (PennsieveConfig, error) {
	return c.LoadWithEnvSettings(DeployedPennsieveEnvironmentSettings)
}
