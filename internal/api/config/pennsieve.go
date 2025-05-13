package config

import (
	"fmt"
	config2 "github.com/pennsieve/collections-service/internal/shared/config"
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

func LoadPennsieveConfig(options ...PennsieveOption) (PennsieveConfig, error) {
	pennsieveConfig := NewPennsieveConfig(options...)
	if len(pennsieveConfig.DiscoverServiceURL) == 0 {
		url, err := config2.GetEnv("DISCOVER_SERVICE_HOST")
		if err != nil {
			return PennsieveConfig{}, err
		}
		if !strings.HasPrefix(url, "http") {
			url = fmt.Sprintf("https://%s", url)
		}
		pennsieveConfig.DiscoverServiceURL = url
	}
	if len(pennsieveConfig.DOIPrefix) == 0 {
		prefix, err := config2.GetEnv("PENNSIEVE_DOI_PREFIX")
		if err != nil {
			return PennsieveConfig{}, err
		}
		pennsieveConfig.DOIPrefix = prefix
	}
	return pennsieveConfig, nil
}
