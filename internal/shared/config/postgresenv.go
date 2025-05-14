package config

const PostgresHostKey = "POSTGRES_HOST"
const PostgresPortKey = "POSTGRES_PORT"
const PostgresUserKey = "POSTGRES_USER"
const PostgresPasswordKey = "POSTGRES_PASSWORD"
const PostgresCollectionsDatabaseKey = "POSTGRES_COLLECTIONS_DATABASE"

type PostgresDBEnvironmentSettings struct {
	Host                EnvironmentSetting
	Port                EnvironmentSetting
	User                EnvironmentSetting
	Password            EnvironmentSetting
	CollectionsDatabase EnvironmentSetting
}

var DefaultPostgresPort = "5432"

// DeployedPostgresDBEnvironmentSettings are the settings used for actual deployments (as opposed to tests).
// The only default value the Postgres port 5432.
var DeployedPostgresDBEnvironmentSettings = PostgresDBEnvironmentSettings{
	Host:                NewEnvironmentSetting(PostgresHostKey),
	Port:                NewEnvironmentSettingWithDefault(PostgresPortKey, DefaultPostgresPort),
	User:                NewEnvironmentSetting(PostgresUserKey),
	Password:            NewEnvironmentSetting(PostgresPasswordKey),
	CollectionsDatabase: NewEnvironmentSetting(PostgresCollectionsDatabaseKey),
}
