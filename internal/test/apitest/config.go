package apitest

import (
	"github.com/pennsieve/collections-service/internal/api/config"
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

func PennsieveConfig(discoverServiceHost string) config.PennsieveConfig {
	return config.NewPennsieveConfigBuilder().
		WithDiscoverServiceHost(discoverServiceHost).
		WithDOIPrefix(test.DOIPrefix).
		Build()
}
