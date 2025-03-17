package postgres

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	"github.com/jackc/pgx/v5"
)

type PostgresDB interface {
	Connect(ctx context.Context, databaseName string) (*pgx.Conn, error)
}

type RDSProxy struct {
	config aws.Config
	host   string
	port   int
	user   string
}

func NewRDSProxy(config aws.Config, host string, port int, user string) *RDSProxy {
	return &RDSProxy{
		config,
		host,
		port,
		user,
	}
}

func (db *RDSProxy) Connect(ctx context.Context, databaseName string) (*pgx.Conn, error) {
	authenticationToken, err := auth.BuildAuthToken(
		ctx,
		fmt.Sprintf("%s:%d", db.host, db.port),
		db.config.Region,
		db.user,
		db.config.Credentials,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create authentication token: %w", err)
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s",
		db.host, db.port, db.user, authenticationToken, databaseName,
	)

	return pgx.Connect(ctx, dsn)
}
