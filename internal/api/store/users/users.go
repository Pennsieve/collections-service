package users

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/pennsieve/collections-service/internal/shared/clients/postgres"
	"log/slog"
)

type Store interface {
	// GetUser returns the user with the given id if user exists. Otherwise, returns ErrUserNotFound
	GetUser(ctx context.Context, userID int32) (GetUserResponse, error)
}

type PostgresStore struct {
	db           postgres.DB
	databaseName string
	logger       *slog.Logger
}

func NewPostgresStore(db postgres.DB, usersDatabaseName string, logger *slog.Logger) *PostgresStore {
	return &PostgresStore{
		db:           db,
		databaseName: usersDatabaseName,
		logger:       logger.With(slog.String("type", "users.PostgresStore")),
	}
}

func (s *PostgresStore) GetUser(ctx context.Context, userID int32) (GetUserResponse, error) {
	conn, err := s.db.Connect(ctx, s.databaseName)
	if err != nil {
		return GetUserResponse{}, fmt.Errorf("CreateCollection error connecting to database %s: %w", s.databaseName, err)
	}
	defer s.closeConn(ctx, conn)

	args := pgx.NamedArgs{"id": userID}
	query := `SELECT u.first_name, u.middle_initial, u.last_name, u.degree, u.orcid_authorization->>'orcid' from pennsieve.users u where id = @id`

	var user GetUserResponse
	if err := conn.QueryRow(ctx, query, args).Scan(&user.FirstName, &user.MiddleInitial, &user.LastName, &user.Degree, &user.ORCID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return GetUserResponse{}, ErrUserNotFound
		}
		return GetUserResponse{}, fmt.Errorf("error looking up user %d: %w", userID, err)
	}

	return user, nil

}

func (s *PostgresStore) closeConn(ctx context.Context, conn *pgx.Conn) {
	if err := conn.Close(ctx); err != nil {
		s.logger.Warn("error closing users.PostgresStore DB connection", slog.Any("error", err))
	}
}
