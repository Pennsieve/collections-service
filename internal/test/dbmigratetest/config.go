package dbmigratetest

import (
	"github.com/pennsieve/collections-service/internal/dbmigrate"
	"github.com/pennsieve/collections-service/internal/test/configtest"
)

// Config returns a dbmigrate.Config suitable for use against
// the pennseivedb instance started for testing. It is preferred in tests over
// calling dbmigrate.LoadConfig() because that method
// will not create the correct configs if the tests are running locally instead
// of in the Docker test container.
func Config(pgOptions ...configtest.PostgresOption) dbmigrate.Config {
	return dbmigrate.Config{
		PostgresDB:     configtest.PostgresDBConfig(pgOptions...),
		VerboseLogging: true,
	}
}
