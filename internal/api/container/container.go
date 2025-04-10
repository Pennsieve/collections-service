package container

import (
	"context"
	"github.com/pennsieve/collections-service/internal/api/config"
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/collections-service/internal/api/store"
	"github.com/pennsieve/collections-service/internal/shared/logging"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/pennsieve/collections-service/internal/shared/clients/postgres"
)

type DependencyContainer interface {
	PostgresDB() postgres.DB
	Discover() service.Discover
	CollectionsStore() store.CollectionsStore
	Logger() *slog.Logger
	SetLogger(logger *slog.Logger)
}

type Container struct {
	AwsConfig        aws.Config
	Config           config.Config
	postgresdb       *postgres.RDSProxy
	discover         *service.HTTPDiscover
	collectionsStore *store.PostgresCollectionsStore
	logger           *slog.Logger
}

func NewContainer() (*Container, error) {
	containerConfig := config.LoadConfig()

	awsCfg, err := awsConfig.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, err
	}

	return NewContainerFromConfig(containerConfig, awsCfg), nil
}

func NewContainerFromConfig(config config.Config, awsConfig aws.Config) *Container {
	return &Container{
		Config:    config,
		AwsConfig: awsConfig,
	}
}

func (c *Container) SetLogger(logger *slog.Logger) {
	c.logger = logger
}

func (c *Container) Logger() *slog.Logger {
	if c.logger == nil {
		c.logger = logging.Default.With(slog.String("warning", "should set logger with context"))
	}
	return c.logger
}

func (c *Container) PostgresDB() postgres.DB {
	if c.postgresdb == nil {
		pgCfg := c.Config.PostgresDB
		c.postgresdb = postgres.NewRDSProxy(
			c.AwsConfig,
			pgCfg.Host,
			pgCfg.Port,
			pgCfg.User,
		)
	}

	return c.postgresdb
}

func (c *Container) Discover() service.Discover {
	if c.discover == nil {
		c.discover = service.NewHTTPDiscover(c.Config.PennsieveConfig.DiscoverServiceURL, c.Logger())
	}
	return c.discover
}

func (c *Container) CollectionsStore() store.CollectionsStore {
	if c.collectionsStore == nil {
		c.collectionsStore = store.NewPostgresCollectionsStore(c.PostgresDB(),
			c.Config.PostgresDB.CollectionsDatabase,
			c.Logger())
	}
	return c.collectionsStore
}
