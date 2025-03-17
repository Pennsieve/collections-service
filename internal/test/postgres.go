package test

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type PostgresDB struct {
	host     string
	port     int
	user     string
	password string
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
