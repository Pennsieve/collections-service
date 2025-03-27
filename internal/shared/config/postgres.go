package config

type PostgresDBConfig struct {
	Host                string
	Port                int
	User                string
	Password            *string
	CollectionsDatabase string
}

func LoadPostgresDBConfig() PostgresDBConfig {
	return NewPostgresDBConfigBuilder().Build()
}

type PostgresDBConfigBuilder struct {
	c *PostgresDBConfig
}

func NewPostgresDBConfigBuilder() *PostgresDBConfigBuilder {
	return &PostgresDBConfigBuilder{c: &PostgresDBConfig{}}
}

func (b *PostgresDBConfigBuilder) WithPostgresUser(postgresUser string) *PostgresDBConfigBuilder {
	b.c.User = postgresUser
	return b
}

func (b *PostgresDBConfigBuilder) WithPostgresPassword(postgresPassword string) *PostgresDBConfigBuilder {
	b.c.Password = &postgresPassword
	return b
}

func (b *PostgresDBConfigBuilder) Build() PostgresDBConfig {
	if len(b.c.Host) == 0 {
		b.c.Host = GetEnvOrDefault("POSTGRES_HOST", "localhost")
	}
	if b.c.Port == 0 {
		b.c.Port = Atoi(GetEnvOrDefault("POSTGRES_PORT", "5432"))
	}
	if len(b.c.User) == 0 {
		b.c.User = getEnv("POSTGRES_USER")
	}
	if b.c.Password == nil {
		b.c.Password = getEnvOrNil("POSTGRES_PASSWORD")
	}
	if len(b.c.CollectionsDatabase) == 0 {
		b.c.CollectionsDatabase = GetEnvOrDefault("POSTGRES_COLLECTIONS_DATABASE", "postgres")
	}
	return *b.c
}
