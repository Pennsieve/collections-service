package apitest

import (
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/collections-service/internal/api/store"
	"github.com/pennsieve/collections-service/internal/shared/clients/postgres"
	"github.com/pennsieve/collections-service/internal/shared/logging"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/stretchr/testify/require"
	"log/slog"
)

type TestContainer struct {
	t                    require.TestingT
	TestPostgresDB       postgres.DB
	TestDiscover         service.Discover
	TestCollectionsStore store.CollectionsStore
	logger               *slog.Logger
}

func (c *TestContainer) PostgresDB() postgres.DB {
	require.NotNil(c.t, c.TestPostgresDB, "no postgres.DB set for this TestContainer")
	return c.TestPostgresDB
}

func (c *TestContainer) Discover() service.Discover {
	require.NotNil(c.t, c.TestDiscover, "no service.Discover set for this TestContainer")
	return c.TestDiscover
}

func (c *TestContainer) CollectionsStore() store.CollectionsStore {
	require.NotNil(c.t, c.TestCollectionsStore, "no store.CollectionsStore set for this TestContainer")
	return c.TestCollectionsStore
}

func (c *TestContainer) Logger() *slog.Logger {
	if c.logger == nil {
		c.logger = logging.Default
	}
	return c.logger
}

func (c *TestContainer) SetLogger(logger *slog.Logger) {
	c.logger = logger
}

func NewTestContainer(t require.TestingT) *TestContainer {
	test.Helper(t)
	return &TestContainer{t: t}
}

func (c *TestContainer) WithPostgresDB(db postgres.DB) *TestContainer {
	c.TestPostgresDB = db
	return c
}

func (c *TestContainer) WithDiscover(discover service.Discover) *TestContainer {
	c.TestDiscover = discover
	return c
}

func (c *TestContainer) WithCollectionsStore(collectionsStore store.CollectionsStore) *TestContainer {
	c.TestCollectionsStore = collectionsStore
	return c
}
