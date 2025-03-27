package containertest

import (
	"github.com/pennsieve/collections-service/internal/shared/clients/postgres"
	"github.com/pennsieve/collections-service/internal/shared/config"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/configtest"
)

// Config returns a config.Config suitable for use against
// the pennseivedb instance started for testing. It is preferred in tests over
// calling config.LoadConfig() because that method
// will not create the correct configs if the tests are running locally instead
// of in the Docker test container.
func Config() config.Config {
	return config.Config{
		PostgresDB: configtest.PostgresDBConfig(),
	}
}

type IntegrationTestContainer struct {
	Config     config.Config
	postgresdb postgres.DB
}

func NewIntegrationTestContainer() *IntegrationTestContainer {
	containerConfig := Config()

	return &IntegrationTestContainer{
		Config: containerConfig,
	}
}

func (c *IntegrationTestContainer) PostgresDB() postgres.DB {
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
	MockPostgresDB postgres.DB
}

func NewMockTestContainer() *MockTestContainer {
	return &MockTestContainer{
		MockPostgresDB: &test.PostgresDB{}, // TODO mock
	}
}

func (c *MockTestContainer) PostgresDB() postgres.DB {
	return c.MockPostgresDB
}
