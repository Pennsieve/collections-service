package apitest

import (
	"context"
	"github.com/pennsieve/collections-service/internal/api/config"
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/collections-service/internal/api/store/collections"
	"github.com/pennsieve/collections-service/internal/api/store/manifests"
	"github.com/pennsieve/collections-service/internal/api/store/users"
	"github.com/pennsieve/collections-service/internal/shared/clients/postgres"
	"github.com/pennsieve/collections-service/internal/shared/logging"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/stretchr/testify/require"
	"log/slog"
)

type TestContainer struct {
	TestPostgresDB       postgres.DB
	TestDiscover         service.Discover
	TestInternalDiscover service.InternalDiscover
	TestCollectionsStore collections.Store
	TestUsersStore       users.Store
	TestManifestStore    manifests.Store
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

func (c *TestContainer) CollectionsStore() collections.Store {
	if c.TestCollectionsStore == nil {
		panic("no collections.Store set for this TestContainer")
	}
	return c.TestCollectionsStore
}

func (c *TestContainer) InternalDiscover(_ context.Context) (service.InternalDiscover, error) {
	if c.TestInternalDiscover == nil {
		panic("no service.InternalDiscover set for this TestContainer")
	}
	return c.TestInternalDiscover, nil
}

func (c *TestContainer) UsersStore() users.Store {
	if c.TestUsersStore == nil {
		panic("no users.Store set for this TestContainer")
	}
	return c.TestUsersStore
}

func (c *TestContainer) ManifestStore() manifests.Store {
	if c.TestManifestStore == nil {
		panic("no manifests.Store set for this TestContainer")
	}
	return c.TestManifestStore
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

func (c *TestContainer) AddLoggingContext(args ...any) {
	c.logger = c.Logger().With(args...)
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

func (c *TestContainer) WithCollectionsStore(collectionsStore collections.Store) *TestContainer {
	c.TestCollectionsStore = collectionsStore
	return c
}

func (c *TestContainer) WithContainerStoreFromPostgresDB(collectionsDBName string) *TestContainer {
	if c.TestPostgresDB == nil {
		panic("cannot create ContainerStore from nil PostgresDB; call WithPostgresDB first")
	}
	c.TestCollectionsStore = collections.NewPostgresStore(c.TestPostgresDB, collectionsDBName, c.Logger())
	return c
}

func (c *TestContainer) WithInternalDiscover(internalDiscover service.InternalDiscover) *TestContainer {
	c.TestInternalDiscover = internalDiscover
	return c
}

func (c *TestContainer) WithHTTPTestInternalDiscover(pennsieveConfig config.PennsieveConfig) *TestContainer {
	c.TestInternalDiscover = service.NewHTTPInternalDiscover(
		pennsieveConfig.DiscoverServiceURL,
		*pennsieveConfig.JWTSecretKey.Value,
		pennsieveConfig.CollectionNamespaceID,
		c.Logger())
	return c
}

func (c *TestContainer) WithUsersStore(usersStore users.Store) *TestContainer {
	c.TestUsersStore = usersStore
	return c
}

func (c *TestContainer) WithUsersStoreFromPostgresDB(collectionsDBName string) *TestContainer {
	if c.TestPostgresDB == nil {
		panic("cannot create users.Store from nil PostgresDB; call WithPostgresDB first")
	}
	c.TestUsersStore = users.NewPostgresStore(c.TestPostgresDB, collectionsDBName, c.Logger())
	return c
}

func (c *TestContainer) WithManifestStore(manifestStore manifests.Store) *TestContainer {
	c.TestManifestStore = manifestStore
	return c
}

func (c *TestContainer) WithMinIOManifestStore(ctx context.Context, t require.TestingT, publishBucket string) *TestContainer {
	s3Client := test.DefaultMinIOS3Client(ctx, t)
	c.TestManifestStore = manifests.NewS3Store(s3Client, publishBucket, c.Logger())
	return c
}
