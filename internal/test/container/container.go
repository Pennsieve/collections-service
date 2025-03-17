package container

import (
	"github.com/pennsieve/collections-service/internal/shared/clients/postgres"
	"github.com/pennsieve/collections-service/internal/shared/config"
	"github.com/pennsieve/collections-service/internal/test"
)

type IntegrationTestContainer struct {
	Config     config.Config
	postgresdb postgres.PostgresDB
}

func NewIntegrationTestContainer() *IntegrationTestContainer {
	containerConfig := config.LoadConfig()

	return &IntegrationTestContainer{
		Config: containerConfig,
	}
}

func (c *IntegrationTestContainer) PostgresDB() postgres.PostgresDB {
	if c.postgresdb == nil {
		pgConfig := c.Config.PostgresDB
		c.postgresdb = test.NewPostgresDB(
			pgConfig.Host,
			pgConfig.Port,
			pgConfig.User,
			*pgConfig.Password,
		)
	}

	return c.postgresdb
}

type MockTestContainer struct {
	MockPostgresDB postgres.PostgresDB
}

func NewMockTestContainer() *MockTestContainer {
	return &MockTestContainer{
		MockPostgresDB: &test.PostgresDB{}, // TODO mock
	}
}

func (c *MockTestContainer) PostgresDB() postgres.PostgresDB {
	return c.MockPostgresDB
}
