package apitest

import (
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/config"
	"github.com/pennsieve/collections-service/internal/api/service/jwtdiscover"
	sharedconfig "github.com/pennsieve/collections-service/internal/shared/config"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
	"strconv"
	"strings"
)

const CollectionNamespaceID = int64(-20)

type ConfigBuilder struct {
	c *config.Config
}

func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{c: &config.Config{
		Environment: "test",
	}}
}

func (b *ConfigBuilder) WithPostgresDBConfig(postgresDBConfig sharedconfig.PostgresDBConfig) *ConfigBuilder {
	b.c.PostgresDB = postgresDBConfig
	return b
}

func (b *ConfigBuilder) WithPennsieveConfig(pennsieveConfig config.PennsieveConfig) *ConfigBuilder {
	b.c.PennsieveConfig = pennsieveConfig
	return b
}

func (b *ConfigBuilder) WithEnvironment(env string) *ConfigBuilder {
	b.c.Environment = env
	return b
}

func (b *ConfigBuilder) Build() config.Config {
	return *b.c
}

func PennsieveConfig(discoverServiceURL string) config.PennsieveConfig {
	return config.NewPennsieveConfig(
		config.WithDiscoverServiceURL(discoverServiceURL),
		config.WithDOIPrefix(PennsieveDOIPrefix),
		config.WithJWTSecretKey(uuid.NewString()),
		config.WithCollectionNamespaceID(CollectionNamespaceID),
	)
}

func PennsieveConfigWithFakeURL() config.PennsieveConfig {
	return config.NewPennsieveConfig(
		config.WithDiscoverServiceURL("http://example.com/discover"),
		config.WithDOIPrefix(PennsieveDOIPrefix),
		config.WithJWTSecretKey(uuid.NewString()),
		config.WithCollectionNamespaceID(CollectionNamespaceID),
	)
}

func ExpectedOrgServiceRole(collectionNamespaceID int64) jwtdiscover.ServiceRole {
	return jwtdiscover.ServiceRole{
		Type:   jwtdiscover.OrganizationServiceRoleType,
		Id:     strconv.FormatInt(collectionNamespaceID, 10),
		NodeId: "",
		Role:   strings.ToLower(role.Owner.String()),
	}
}
