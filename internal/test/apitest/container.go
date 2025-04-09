package apitest

import (
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/collections-service/internal/api/store"
	"github.com/pennsieve/collections-service/internal/shared/clients/postgres"
	"github.com/pennsieve/collections-service/internal/shared/logging"
	"log/slog"
)

type TestContainer struct {
	TestPostgresDB       postgres.DB
	TestDiscover         service.Discover
	TestCollectionsStore store.CollectionsStore
	logger               *slog.Logger
}

func (c *TestContainer) PostgresDB() postgres.DB {
	if c.TestPostgresDB == nil {
		panic("no postgres.DB set for this TestContainer")
	}
	return c.TestPostgresDB
}

func (c *TestContainer) Discover() service.Discover {
	if c.TestDiscover == nil {
		panic("no service.Discover set for this TestContainer")
	}
	return c.TestDiscover
}

func (c *TestContainer) CollectionsStore() store.CollectionsStore {
	if c.TestCollectionsStore == nil {
		panic("no store.CollectionsStore set for this TestContainer")
	}
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

func NewTestContainer() *TestContainer {
	return &TestContainer{}
}

func (c *TestContainer) WithPostgresDB(db postgres.DB) *TestContainer {
	c.TestPostgresDB = db
	return c
}

func (c *TestContainer) WithDiscover(discover service.Discover) *TestContainer {
	c.TestDiscover = discover
	return c
}

func (c *TestContainer) WithHTTPTestDiscover(mockServerURL string) *TestContainer {
	c.TestDiscover = service.NewHTTPDiscover(mockServerURL, c.Logger())
	return c
}

func (c *TestContainer) WithCollectionsStore(collectionsStore store.CollectionsStore) *TestContainer {
	c.TestCollectionsStore = collectionsStore
	return c
}

func (c *TestContainer) WithContainerStoreFromPostgresDB(collectionsDBName string) *TestContainer {
	if c.TestPostgresDB == nil {
		panic("cannot create ContainerStore from nil PostgresDB; call WithPostgresDB first")
	}
	c.TestCollectionsStore = store.NewPostgresCollectionsStore(c.TestPostgresDB, collectionsDBName, c.Logger())
	return c
}
