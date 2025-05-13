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

func (c PostgresDBConfig) LoadWithEnvSettings(environmentSettings PostgresDBEnvironmentSettings) (PostgresDBConfig, error) {
	if len(c.Host) == 0 {
		host, err := environmentSettings.Host.Get()
		if err != nil {
			return PostgresDBConfig{}, err
		}
		c.Host = host
	}
	if c.Port == 0 {
		port, err := environmentSettings.Port.GetInt()
		if err != nil {
			return PostgresDBConfig{}, err
		}
		c.Port = port
	}
	if len(c.User) == 0 {
		user, err := environmentSettings.User.Get()
		if err != nil {
			return PostgresDBConfig{}, err
		}
		c.User = user
	}
	if c.Password == nil {
		c.Password = environmentSettings.Password.GetNillable()
	}
	if len(c.CollectionsDatabase) == 0 {
		databaseName, err := environmentSettings.CollectionsDatabase.Get()
		if err != nil {
			return PostgresDBConfig{}, err
		}
		c.CollectionsDatabase = databaseName
	}
	return c, nil
}

func (c PostgresDBConfig) Load() (PostgresDBConfig, error) {
	return c.LoadWithEnvSettings(DeployedPostgresDBEnvironmentSettings)
}
