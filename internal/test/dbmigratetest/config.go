package dbmigratetest

import (
	"github.com/pennsieve/collections-service/internal/dbmigrate"
	"github.com/pennsieve/collections-service/internal/test/configtest"
)

func Config() dbmigrate.Config {
	return dbmigrate.Config{
		PostgresDB:     configtest.PostgresDBConfig(),
		VerboseLogging: true,
	}
}
