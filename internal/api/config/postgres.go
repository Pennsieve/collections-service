package config

type PostgresDBConfig struct {
	Host                string
	Port                int
	User                string
	Password            *string
	CollectionsDatabase string
}

type PostgresDBOption func(postgresDBConfig *PostgresDBConfig)

func NewPostgresDBConfig(options ...PostgresDBOption) PostgresDBConfig {
	postgresConfig := PostgresDBConfig{}
	for _, option := range options {
		option(&postgresConfig)
	}
	return postgresConfig
}

func WithPostgresUser(postgresUser string) PostgresDBOption {
	return func(postgresDBConfig *PostgresDBConfig) {
		postgresDBConfig.User = postgresUser
	}
}

func WithPostgresPassword(postgresPassword string) PostgresDBOption {
	return func(postgresDBConfig *PostgresDBConfig) {
		postgresDBConfig.Password = &postgresPassword
	}
}

func WithHost(host string) PostgresDBOption {
	return func(postgresDBConfig *PostgresDBConfig) {
		postgresDBConfig.Host = host
	}
}

func WithPort(port int) PostgresDBOption {
	return func(postgresDBConfig *PostgresDBConfig) {
		postgresDBConfig.Port = port
	}
}

func WithCollectionsDatabase(databaseName string) PostgresDBOption {
	return func(postgresDBConfig *PostgresDBConfig) {
		postgresDBConfig.CollectionsDatabase = databaseName
	}
}

func LoadPostgresDBConfig(options ...PostgresDBOption) (PostgresDBConfig, error) {
	postgresConfig := NewPostgresDBConfig(options...)
	if len(postgresConfig.Host) == 0 {
		postgresConfig.Host = GetEnvOrDefault("POSTGRES_HOST", "localhost")
	}
	if postgresConfig.Port == 0 {
		port, err := GetIntEnvOrDefault("POSTGRES_PORT", "5432")
		if err != nil {
			return PostgresDBConfig{}, err
		}
		postgresConfig.Port = port
	}
	if len(postgresConfig.User) == 0 {
		user, err := GetEnv("POSTGRES_USER")
		if err != nil {
			return PostgresDBConfig{}, err
		}
		postgresConfig.User = user
	}
	if postgresConfig.Password == nil {
		postgresConfig.Password = GetEnvOrNil("POSTGRES_PASSWORD")
	}
	if len(postgresConfig.CollectionsDatabase) == 0 {
		databaseName, err := GetEnv("POSTGRES_COLLECTIONS_DATABASE")
		if err != nil {
			return PostgresDBConfig{}, err
		}
		postgresConfig.CollectionsDatabase = databaseName
	}
	return postgresConfig, nil
}
