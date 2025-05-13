package test

import (
	"context"
	"fmt"
	"github.com/pennsieve/collections-service/internal/api/config"
	"github.com/stretchr/testify/require"

	"github.com/jackc/pgx/v5"
)

type PostgresDB struct {
	host     string
	port     int
	user     string
	password string
}

func NewPostgresDBFromConfig(t require.TestingT, pgConfig config.PostgresDBConfig) *PostgresDB {
	Helper(t)
	require.NotNil(t, pgConfig.Password)
	return NewPostgresDB(pgConfig.Host, pgConfig.Port, pgConfig.User, *pgConfig.Password)
}

func NewPostgresDB(host string, port int, user string, password string) *PostgresDB {
	return &PostgresDB{
		host,
		port,
		user,
		password,
	}
}

func (db *PostgresDB) Connect(ctx context.Context, databaseName string) (*pgx.Conn, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s",
		db.host, db.port, db.user, db.password, databaseName,
	)

	return pgx.Connect(ctx, dsn)
}

func CloseConnection(ctx context.Context, t require.TestingT, conn *pgx.Conn) {
	Helper(t)
	require.NoError(t, conn.Close(ctx))
}
