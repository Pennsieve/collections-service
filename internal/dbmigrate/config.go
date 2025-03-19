package dbmigrate

import (
	"fmt"
	"github.com/pennsieve/collections-service/internal/shared/config"
	"strconv"
)

const VerboseLoggingKey = "VERBOSE_LOGGING"

type Config struct {
	PostgresDB     config.PostgresDBConfig
	VerboseLogging bool
}

func LoadConfig() (Config, error) {
	verboseStr := config.GetEnvOrDefault(VerboseLoggingKey, "false")
	isVerbose, err := strconv.ParseBool(verboseStr)
	if err != nil {
		return Config{}, fmt.Errorf("error converting %q value %s to bool: %w",
			VerboseLoggingKey, verboseStr, err)
	}
	return Config{
		PostgresDB:     config.LoadPostgresDBConfig(),
		VerboseLogging: isVerbose,
	}, nil
}
