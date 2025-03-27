package config

type PennsieveConfig struct {
	DiscoverServiceHost string
	DOIPrefix           string
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

func (b *PennsieveConfigBuilder) WithDiscoverServiceHost(host string) *PennsieveConfigBuilder {
	b.c.DiscoverServiceHost = host
	return b
}

func (b *PennsieveConfigBuilder) WithDOIPrefix(doiPrefix string) *PennsieveConfigBuilder {
	b.c.DOIPrefix = doiPrefix
	return b
}

func (b *PennsieveConfigBuilder) Build() PennsieveConfig {
	if len(b.c.DiscoverServiceHost) == 0 {
		b.c.DiscoverServiceHost = getEnv("DISCOVER_SERVICE_HOST")
	}
	if len(b.c.DOIPrefix) == 0 {
		b.c.DOIPrefix = getEnv("PENNSIEVE_DOI_PREFIX")
	}
	return *b.c
}
