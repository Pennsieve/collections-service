package container

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pennsieve/collections-service/internal/api/config"
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/collections-service/internal/api/store/collections"
	"github.com/pennsieve/collections-service/internal/api/store/manifests"
	"github.com/pennsieve/collections-service/internal/api/store/users"
	"github.com/pennsieve/collections-service/internal/shared/clients/ssm"
	"github.com/pennsieve/collections-service/internal/shared/logging"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	awsSSM "github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/pennsieve/collections-service/internal/shared/clients/postgres"
)

type DependencyContainer interface {
	PostgresDB() postgres.DB

	Discover() service.Discover
	// InternalDiscover returns a Discover service for calling
	// the internal endpoints. Since these require authz, the setup
	// is a little different and requires calling SSM. So it is separated
	// out from Discover so that we only do this setup if the internal
	// endpoints will be used.
	InternalDiscover(ctx context.Context) (service.InternalDiscover, error)
	DOI(ctx context.Context) (service.DOI, error)

	CollectionsStore() collections.Store
	UsersStore() users.Store
	ManifestStore() manifests.Store

	Logger() *slog.Logger
	SetLogger(logger *slog.Logger)
	AddLoggingContext(args ...any)
}

type Container struct {
	AwsConfig        aws.Config
	Config           config.Config
	postgresdb       *postgres.RDSProxy
	discover         *service.HTTPDiscover
	internalDiscover *service.HTTPInternalDiscover
	doi              *service.HTTPDOI
	collectionsStore *collections.PostgresStore
	usersStore       *users.PostgresStore
	manifestStore    *manifests.S3Store
	parameterStore   *ssm.AWSParameterStore
	logger           *slog.Logger
}

func NewContainer() (*Container, error) {
	containerConfig, err := config.LoadConfig()
	if err != nil {
		return nil, err
	}

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

func (c *Container) AddLoggingContext(args ...any) {
	c.logger = c.Logger().With(args...)
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

func (c *Container) CollectionsStore() collections.Store {
	if c.collectionsStore == nil {
		c.collectionsStore = collections.NewPostgresStore(c.PostgresDB(),
			c.Config.PostgresDB.CollectionsDatabase,
			c.Logger())
	}
	return c.collectionsStore
}

func (c *Container) UsersStore() users.Store {
	if c.usersStore == nil {
		c.usersStore = users.NewPostgresStore(c.PostgresDB(), c.Config.PostgresDB.CollectionsDatabase, c.Logger())
	}
	return c.usersStore
}

func (c *Container) ManifestStore() manifests.Store {
	if c.manifestStore == nil {
		s3Client := s3.NewFromConfig(c.AwsConfig)
		c.manifestStore = manifests.NewS3Store(s3Client, c.Config.PennsieveConfig.PublishBucket, c.Logger())
	}
	return c.manifestStore
}

// ParameterStore is not part of the interface, since right now it is only used internally by Config.
func (c *Container) ParameterStore() ssm.ParameterStore {
	if c.parameterStore == nil {
		c.parameterStore = ssm.NewAWSParameterStore(awsSSM.NewFromConfig(c.AwsConfig))
	}
	return c.parameterStore
}

func (c *Container) loadJWTSecretKey(ctx context.Context) (string, error) {
	jwtSecretKey, err := c.Config.PennsieveConfig.JWTSecretKey.Load(
		ctx,
		c.ParameterStore().GetParameter)
	if err != nil {
		return "", fmt.Errorf("error creating internal service; cannot get JWT secret Key from SSM: %w", err)
	}
	return jwtSecretKey, nil
}

func (c *Container) InternalDiscover(ctx context.Context) (service.InternalDiscover, error) {
	if c.internalDiscover == nil {
		jwtSecretKey, err := c.loadJWTSecretKey(ctx)
		if err != nil {
			return nil, fmt.Errorf("error creating internal discover: %w", err)
		}
		c.internalDiscover = service.NewHTTPInternalDiscover(
			c.Config.PennsieveConfig.DiscoverServiceURL,
			jwtSecretKey,
			c.Config.PennsieveConfig.CollectionsIDSpace.ID,
			c.Logger())
	}
	return c.internalDiscover, nil
}

func (c *Container) DOI(ctx context.Context) (service.DOI, error) {
	if c.doi == nil {
		jwtSecretKey, err := c.loadJWTSecretKey(ctx)
		if err != nil {
			return nil, fmt.Errorf("error creating doi service: %w", err)
		}
		c.doi = service.NewHTTPDOI(
			c.Config.PennsieveConfig.DOIServiceURL,
			jwtSecretKey,
			c.Config.PennsieveConfig.CollectionsIDSpace.ID,
			c.Logger(),
		)
	}
	return c.doi, nil
}
