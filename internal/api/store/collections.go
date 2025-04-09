package store

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/pennsieve/collections-service/internal/shared/clients/postgres"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
	"log/slog"
	"strings"
)

type CollectionsStore interface {
	CreateCollection(ctx context.Context, userID int64, nodeID, name, description string, dois []string) (CreateCollectionResponse, error)
	GetCollections(ctx context.Context, userID int64, limit int, offset int) (GetCollectionsResponse, error)
}

type PostgresCollectionsStore struct {
	db           postgres.DB
	databaseName string
	logger       *slog.Logger
}

func NewPostgresCollectionsStore(db postgres.DB, collectionsDatabaseName string, logger *slog.Logger) *PostgresCollectionsStore {
	return &PostgresCollectionsStore{
		db:           db,
		databaseName: collectionsDatabaseName,
		logger:       logger.With(slog.String("type", "PostgresCollectionsStore")),
	}
}

type CreateCollectionResponse struct {
	ID          int64
	CreatorRole role.Role
}

func (s *PostgresCollectionsStore) CreateCollection(ctx context.Context, userID int64, nodeID, name, description string, dois []string) (CreateCollectionResponse, error) {
	conn, err := s.db.Connect(ctx, s.databaseName)
	if err != nil {
		return CreateCollectionResponse{}, fmt.Errorf("error connecting to database %s: %w", s.databaseName, err)
	}
	defer s.closeConn(ctx, conn)
	creatorPermission := pgdb.Owner

	insertCollectionArgs := pgx.NamedArgs{
		"name":           name,
		"description":    description,
		"node_id":        nodeID,
		"user_id":        userID,
		"permission_bit": creatorPermission,
		"role":           PgxRole(creatorPermission.ToRole()),
	}
	insertCollectionSQLFormat := `WITH new_collection AS (
      INSERT INTO collections.collections (name, description, node_id) 
                                VALUES (@name, @description, @node_id) RETURNING id
    ) %s
	INSERT INTO collections.collection_user (collection_id, user_id, permission_bit, role)
	SELECT id, @user_id, @permission_bit, @role
	FROM new_collection
	RETURNING (select id from new_collection);`

	var insertDOISQL string
	if len(dois) > 0 {
		var values []string
		for i, doi := range dois {
			key := fmt.Sprintf("doi_%d", i)
			values = append(values, fmt.Sprintf("(@%s)", key))
			insertCollectionArgs[key] = doi
		}
		insertDOISQLFormat := `, t AS (
                          INSERT INTO collections.dois (collection_id, doi)
                          SELECT new_collection.id, doi
                          FROM new_collection, (VALUES %s) AS new_dois(doi)
                       )`
		insertDOISQL = fmt.Sprintf(insertDOISQLFormat, strings.Join(values, ", "))
	}
	insertCollectionSQL := fmt.Sprintf(insertCollectionSQLFormat, insertDOISQL)
	var collectionID int64
	if err := conn.QueryRow(ctx, insertCollectionSQL, insertCollectionArgs).Scan(&collectionID); err != nil {
		return CreateCollectionResponse{}, fmt.Errorf("error inserting new collection %s: %w", name, err)
	}
	s.logger.Debug("inserted new collection",
		slog.Int64("id", collectionID),
		slog.String("name", name))
	return CreateCollectionResponse{
		ID:          collectionID,
		CreatorRole: creatorPermission.ToRole(),
	}, nil
}

type GetCollectionsResponse struct {
	TotalCount int64
}

func (s *PostgresCollectionsStore) GetCollections(ctx context.Context, userID int64, limit int, offset int) (GetCollectionsResponse, error) {
	return GetCollectionsResponse{}, nil
}

func (s *PostgresCollectionsStore) closeConn(ctx context.Context, conn *pgx.Conn) {
	if err := conn.Close(ctx); err != nil {
		s.logger.Warn("error closing DB connection", slog.Any("error", err))
	}
}
