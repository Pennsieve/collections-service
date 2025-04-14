package store

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/pennsieve/collections-service/internal/api/config"
	"github.com/pennsieve/collections-service/internal/shared/clients/postgres"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"log/slog"
	"strings"
)

const MaxDOIsPerCollection = config.MaxBannersPerCollection

type CollectionsStore interface {
	CreateCollection(ctx context.Context, userID int64, nodeID, name, description string, dois []string) (CreateCollectionResponse, error)
	GetCollections(ctx context.Context, userID int64, limit int, offset int) (GetCollectionsResponse, error)
	GetCollection(ctx context.Context, userID int64, nodeID string) (*GetCollectionResponse, error)
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

func (s *PostgresCollectionsStore) CreateCollection(ctx context.Context, userID int64, nodeID, name, description string, dois []string) (CreateCollectionResponse, error) {
	conn, err := s.db.Connect(ctx, s.databaseName)
	if err != nil {
		return CreateCollectionResponse{}, fmt.Errorf("CreateCollection error connecting to database %s: %w", s.databaseName, err)
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

func (s *PostgresCollectionsStore) GetCollections(ctx context.Context, userID int64, limit int, offset int) (GetCollectionsResponse, error) {
	if limit < 0 {
		return GetCollectionsResponse{}, fmt.Errorf("limit cannot be negative: %d", limit)
	}
	if offset < 0 {
		return GetCollectionsResponse{}, fmt.Errorf("offset cannot be negative: %d", offset)

	}
	getCollectionsArgs := pgx.NamedArgs{
		"user_id": userID,
		"limit":   limit,
		"offset":  offset,
	}
	// using ORDER BY c.id asc as a proxy for getting in order of creation, oldest first
	getCollectionsSQL := `SELECT c.*, u.role, count(*) OVER () AS total_count
			FROM collections.collections c
         			JOIN collections.collection_user u ON c.id = u.collection_id
			WHERE u.user_id = @user_id
  				and u.permission_bit > 0
			ORDER BY c.id asc
			LIMIT @limit OFFSET @offset`

	type CollectionUserJoin struct {
		Collection
		Role       PgxRole `db:"role"`
		TotalCount int     `db:"total_count"`
	}

	conn, err := s.db.Connect(ctx, s.databaseName)
	if err != nil {
		return GetCollectionsResponse{}, fmt.Errorf("GetCollections error connecting to database %s: %w", s.databaseName, err)
	}
	defer s.closeConn(ctx, conn)

	// any error here will be returned from pgx.CollectRows which also closes collectionUserJoinRows for us
	collectionUserJoinRows, _ := conn.Query(ctx, getCollectionsSQL, getCollectionsArgs)

	response := GetCollectionsResponse{Limit: limit, Offset: offset}

	var collectionIDs []int64

	collections, err := pgx.CollectRows(collectionUserJoinRows, func(row pgx.CollectableRow) (CollectionResponse, error) {
		join, err := pgx.RowToStructByName[CollectionUserJoin](row)
		if err != nil {
			return CollectionResponse{}, err
		}
		//redundant
		response.TotalCount = join.TotalCount

		collectionIDs = append(collectionIDs, join.ID)

		return CollectionResponse{
			NodeID:      join.NodeID,
			Name:        join.Name,
			Description: join.Description,
			UserRole:    join.Role.AsRole().String(),
		}, nil

	})
	if err != nil {
		return GetCollectionsResponse{}, fmt.Errorf("GetCollections: error querying for collections: %w", err)
	}

	if len(collections) == 0 {
		var totalCount int
		if err := conn.QueryRow(ctx, `SELECT count(*)
                                FROM collections.collections c
         			            	JOIN collections.collection_user u ON c.id = u.collection_id
			                    WHERE u.user_id = @user_id
  				                	and u.permission_bit > 0`, getCollectionsArgs).Scan(&totalCount); err != nil {
			return GetCollectionsResponse{}, fmt.Errorf("GetCollections: error counting total collections: %w", err)
		}
		response.TotalCount = totalCount
		return response, nil
	}

	nodeIDToCollection := make(map[string]*CollectionResponse, len(collections))
	for i := range collections {
		collection := &collections[i]
		nodeIDToCollection[collection.NodeID] = collection
	}
	getDOIsArgs := pgx.NamedArgs{"limit": MaxDOIsPerCollection, "collection_ids": collectionIDs}

	getDOIsSQL := `SELECT c.node_id, d.doi, d.total_count
				   FROM collections.collections c
				   JOIN LATERAL (
	               	SELECT doi, count(*) OVER () AS total_count
			        FROM collections.dois
				    WHERE collection_id = c.id
	                ORDER BY id asc
				    LIMIT @limit
	               ) d ON true
	               WHERE c.id = ANY(@collection_ids)
	               ORDER BY c.id asc`

	// if there is an error, it will be returned by pgx.ForEachRow which will also close doiRows
	doiRows, _ := conn.Query(ctx, getDOIsSQL, getDOIsArgs)

	var nodeID, doi string
	var totalCount int
	if _, err = pgx.ForEachRow(doiRows, []any{&nodeID, &doi, &totalCount}, func() error {
		collection := nodeIDToCollection[nodeID]
		collection.BannerDOIs = append(collection.BannerDOIs, doi)
		collection.Size = totalCount
		return nil
	}); err != nil {
		return GetCollectionsResponse{}, fmt.Errorf("GetCollections: error querying for DOIs: %w", err)
	}

	response.Collections = collections
	return response, nil
}

// GetCollection returns nil and no error if no collection with the given node id exists for the given user id.
// Otherwise, returns a non-nil response if a collection is found or nil and an error if an error occurs.
func (s *PostgresCollectionsStore) GetCollection(ctx context.Context, userID int64, nodeID string) (*GetCollectionResponse, error) {
	args := pgx.NamedArgs{"user_id": userID, "node_id": nodeID}
	sql := `SELECT c.name, c.description, u.role, d.doi
			FROM collections.collections c
         		JOIN collections.collection_user u ON c.id = u.collection_id
         		LEFT JOIN collections.dois d ON c.id = d.collection_id
			WHERE u.user_id = @user_id
  			  AND c.node_id = @node_id
			ORDER BY d.id asc`

	conn, err := s.db.Connect(ctx, s.databaseName)
	if err != nil {
		return nil, fmt.Errorf("GetCollection error connecting to database %s: %w", s.databaseName, err)
	}
	defer s.closeConn(ctx, conn)

	rows, _ := conn.Query(ctx, sql, args)

	var response *GetCollectionResponse
	var name, description, role string
	var doiOpt *string
	_, err = pgx.ForEachRow(rows, []any{&name, &description, &role, &doiOpt}, func() error {
		if response == nil {
			response = &GetCollectionResponse{
				CollectionResponse: CollectionResponse{
					NodeID:      nodeID,
					Name:        name,
					Description: description,
					UserRole:    role,
				},
			}
		}
		if doiOpt != nil {
			response.DOIs = append(response.DOIs, *doiOpt)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("GetCollection error querying for collection %s: %w", nodeID, err)
	}
	if response != nil {
		response.Size = len(response.DOIs)
		for i := 0; i < min(len(response.DOIs), MaxDOIsPerCollection); i++ {
			response.BannerDOIs = append(response.BannerDOIs, response.DOIs[i])
		}
	}
	return response, nil
}

func (s *PostgresCollectionsStore) closeConn(ctx context.Context, conn *pgx.Conn) {
	if err := conn.Close(ctx); err != nil {
		s.logger.Warn("error closing DB connection", slog.Any("error", err))
	}
}
