package container

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/pennsieve/collections-service/internal/shared/clients/postgres"
	"github.com/pennsieve/collections-service/internal/shared/config"
)

type DependencyContainer interface {
	PostgresDB() postgres.DB
}

type Container struct {
	AwsConfig  aws.Config
	Config     config.Config
	postgresdb *postgres.RDSProxy
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
