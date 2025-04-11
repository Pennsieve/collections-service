package config

import (
	"fmt"
	sharedconfig "github.com/pennsieve/collections-service/internal/shared/config"
	"strings"
)

type PennsieveConfig struct {
	DiscoverServiceURL string
	DOIPrefix          string
}

func LoadPennsieveConfig() PennsieveConfig {
	return NewPennsieveConfigBuilder().Build()
}

type PennsieveConfigBuilder struct {
	c *PennsieveConfig
}

func NewPennsieveConfigBuilder() *PennsieveConfigBuilder {
	return &PennsieveConfigBuilder{c: &PennsieveConfig{}}
}

func (b *PennsieveConfigBuilder) WithDiscoverServiceURL(url string) *PennsieveConfigBuilder {
	b.c.DiscoverServiceURL = url
	return b
}

func (b *PennsieveConfigBuilder) WithDOIPrefix(doiPrefix string) *PennsieveConfigBuilder {
	b.c.DOIPrefix = doiPrefix
	return b
}

func (b *PennsieveConfigBuilder) Build() PennsieveConfig {
	if len(b.c.DiscoverServiceURL) == 0 {
		url := sharedconfig.GetEnv("DISCOVER_SERVICE_HOST")
		if !strings.HasPrefix(url, "http") {
			url = fmt.Sprintf("https://%s", url)
		}
		b.c.DiscoverServiceURL = url
	}
	if len(b.c.DOIPrefix) == 0 {
		b.c.DOIPrefix = sharedconfig.GetEnv("PENNSIEVE_DOI_PREFIX")
	}
	return *b.c
}
