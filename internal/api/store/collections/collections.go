package collections

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/pennsieve/collections-service/internal/api/config"
	"github.com/pennsieve/collections-service/internal/api/datasource"
	"github.com/pennsieve/collections-service/internal/api/publishing"
	"github.com/pennsieve/collections-service/internal/shared/clients/postgres"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"log/slog"
	"strings"
	"time"
)

const MaxBannerDOIsPerCollection = config.MaxBannersPerCollection

type Store interface {
	CreateCollection(ctx context.Context, userID int64, nodeID, name, description string, dois []DOI) (CreateCollectionResponse, error)
	// GetCollections returns a paginated list of collection summaries that the given user has at least guest permission on.
	GetCollections(ctx context.Context, userID int64, limit int, offset int) (GetCollectionsResponse, error)
	// GetCollection returns a the given collection if it exists and if the given user has at least guest permission on it.
	GetCollection(ctx context.Context, userID int64, nodeID string) (GetCollectionResponse, error)
	DeleteCollection(ctx context.Context, collectionID int64) error
	UpdateCollection(ctx context.Context, userID, collectionID int64, update UpdateCollectionRequest) (GetCollectionResponse, error)
}

type PostgresStore struct {
	db           postgres.DB
	databaseName string
	logger       *slog.Logger
}

func NewPostgresStore(db postgres.DB, collectionsDatabaseName string, logger *slog.Logger) *PostgresStore {
	return &PostgresStore{
		db:           db,
		databaseName: collectionsDatabaseName,
		logger:       logger.With(slog.String("type", "collections.PostgresStore")),
	}
}

func (s *PostgresStore) CreateCollection(ctx context.Context, userID int64, nodeID, name, description string, dois []DOI) (CreateCollectionResponse, error) {
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
			doiKey := fmt.Sprintf("doi_%d", i)
			datasourceKey := fmt.Sprintf("datasource_%d", i)
			values = append(values, fmt.Sprintf("(@%s, @%s)", doiKey, datasourceKey))
			insertCollectionArgs[doiKey] = doi.Value
			insertCollectionArgs[datasourceKey] = doi.Datasource
		}
		insertDOISQLFormat := `, t AS (
                          INSERT INTO collections.dois (collection_id, doi, datasource)
                          SELECT new_collection.id, doi, datasource
                          FROM new_collection, (VALUES %s) AS new_dois(doi, datasource)
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

func (s *PostgresStore) GetCollections(ctx context.Context, userID int64, limit int, offset int) (GetCollectionsResponse, error) {
	if limit < 0 {
		return GetCollectionsResponse{}, fmt.Errorf("limit cannot be negative: %d", limit)
	}
	if offset < 0 {
		return GetCollectionsResponse{}, fmt.Errorf("offset cannot be negative: %d", offset)

	}
	getCollectionsArgs := pgx.NamedArgs{
		"user_id":  userID,
		"limit":    limit,
		"offset":   offset,
		"min_perm": pgdb.Guest,
	}
	// using ORDER BY c.id asc as a proxy for getting in order of creation, oldest first
	getCollectionsSQL := `SELECT c.id, c.name, c.description, c.node_id, u.role, count(*) OVER () AS total_count
			FROM collections.collections c
         			JOIN collections.collection_user u ON c.id = u.collection_id
			WHERE u.user_id = @user_id AND u.permission_bit >= @min_perm
			ORDER BY c.id asc
			LIMIT @limit OFFSET @offset`

	conn, err := s.db.Connect(ctx, s.databaseName)
	if err != nil {
		return GetCollectionsResponse{}, fmt.Errorf("GetCollections error connecting to database %s: %w", s.databaseName, err)
	}
	defer s.closeConn(ctx, conn)

	// any error here will be returned from pgx.CollectRows which also closes collectionUserJoinRows for us
	collectionUserJoinRows, _ := conn.Query(ctx, getCollectionsSQL, getCollectionsArgs)

	response := GetCollectionsResponse{Limit: limit, Offset: offset}

	// limit may be zero
	collectionIDs := make([]int64, 0, limit+1)
	collections, err := pgx.CollectRows(collectionUserJoinRows, func(row pgx.CollectableRow) (CollectionSummary, error) {
		var id int64
		var name, description, nodeID string
		var role PgxRole
		var totalCount int
		err := row.Scan(&id, &name, &description, &nodeID, &role, &totalCount)
		if err != nil {
			return CollectionSummary{}, err
		}
		//redundant after the first
		response.TotalCount = totalCount

		collectionIDs = append(collectionIDs, id)

		return CollectionSummary{
			CollectionBase: CollectionBase{
				ID:          id,
				NodeID:      nodeID,
				Name:        name,
				Description: description,
				UserRole:    role.AsRole(),
			}}, nil

	})
	if err != nil {
		return GetCollectionsResponse{}, fmt.Errorf("GetCollections: error querying for collections: %w", err)
	}

	// We may have gotten no collections because limit == 0 or offset it too large,
	// but we still want to return a correct total count, so recount with no limit or offset.
	if len(collections) == 0 {
		var totalCount int
		if err := conn.QueryRow(ctx, `SELECT count(*)
	                                FROM collections.collections c
	         			            	JOIN collections.collection_user u ON c.id = u.collection_id
				                    WHERE u.user_id = @user_id AND u.permission_bit >= @min_perm`, getCollectionsArgs).Scan(&totalCount); err != nil {
			return GetCollectionsResponse{}, fmt.Errorf("GetCollections: error counting total collections: %w", err)
		}
		response.TotalCount = totalCount
		return response, nil
	}

	nodeIDToCollection := make(map[string]*CollectionSummary, len(collections))
	for i := range collections {
		collection := &collections[i]
		nodeIDToCollection[collection.NodeID] = collection
	}
	getDOIsArgs := pgx.NamedArgs{"limit": MaxBannerDOIsPerCollection, "collection_ids": collectionIDs}

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

// getCollectionByIDColumn returns the error ErrCollectionNotFound if no collection with the given idValue exists for the given user id.
// idColumn should be either "id" or "node_id"
func getCollectionByIDColumn(ctx context.Context, conn *pgx.Conn, userID int64, idColumn string, idValue any) (GetCollectionResponse, error) {
	args := pgx.NamedArgs{"user_id": userID, idColumn: idValue, "min_perm": pgdb.Guest}

	idCondition := fmt.Sprintf("c.%s = @%s", idColumn, idColumn)

	sql := fmt.Sprintf(`SELECT c.id, c.node_id, c.name, c.description, u.role, d.doi, d.datasource
			FROM collections.collections c
         		JOIN collections.collection_user u ON c.id = u.collection_id
         		LEFT JOIN collections.dois d ON c.id = d.collection_id
			WHERE u.user_id = @user_id AND u.permission_bit >= @min_perm
  			  AND %s
			ORDER BY d.id asc`, idCondition)

	rows, _ := conn.Query(ctx, sql, args)

	var response *GetCollectionResponse
	var id int64
	var nodeID string
	var name, description string
	var pgxRole PgxRole
	var doiOpt *string
	var datasourceOpt *datasource.DOIDatasource
	_, err := pgx.ForEachRow(rows, []any{&id, &nodeID, &name, &description, &pgxRole, &doiOpt, &datasourceOpt}, func() error {
		if response == nil {
			response = &GetCollectionResponse{
				CollectionBase: CollectionBase{
					ID:          id,
					NodeID:      nodeID,
					Name:        name,
					Description: description,
					UserRole:    pgxRole.AsRole(),
				},
			}
		}
		if doiOpt != nil {
			response.DOIs = append(response.DOIs, DOI{
				Value:      *doiOpt,
				Datasource: *datasourceOpt,
			})
		}
		return nil
	})
	if err != nil {
		return GetCollectionResponse{}, err
	}
	if response == nil {
		return GetCollectionResponse{}, ErrCollectionNotFound
	}

	response.Size = len(response.DOIs)
	return *response, nil
}

func getCollectionByNodeID(ctx context.Context, conn *pgx.Conn, userID int64, nodeID string) (GetCollectionResponse, error) {
	return getCollectionByIDColumn(ctx, conn, userID, "node_id", nodeID)
}

func getCollectionByID(ctx context.Context, conn *pgx.Conn, userID int64, collectionID int64) (GetCollectionResponse, error) {
	return getCollectionByIDColumn(ctx, conn, userID, "id", collectionID)
}

// GetCollection returns the error ErrCollectionNotFound if no collection with the given node id exists for the given user id.
func (s *PostgresStore) GetCollection(ctx context.Context, userID int64, nodeID string) (GetCollectionResponse, error) {
	conn, err := s.db.Connect(ctx, s.databaseName)
	if err != nil {
		return GetCollectionResponse{}, fmt.Errorf("GetCollection error connecting to database %s: %w", s.databaseName, err)
	}
	defer s.closeConn(ctx, conn)
	return getCollectionByNodeID(ctx, conn, userID, nodeID)
}

func (s *PostgresStore) DeleteCollection(ctx context.Context, collectionID int64) error {
	conn, err := s.db.Connect(ctx, s.databaseName)
	if err != nil {
		return fmt.Errorf("DeleteCollection error connecting to database %s: %w", s.databaseName, err)
	}
	defer s.closeConn(ctx, conn)

	commandTag, err := conn.Exec(
		ctx,
		"DELETE FROM collections.collections WHERE id = @collection_id",
		pgx.NamedArgs{"collection_id": collectionID},
	)
	if err != nil {
		return fmt.Errorf("DeleteCollection error deleting collection %d: %w", collectionID, err)
	}
	if commandTag.RowsAffected() == 0 {
		return ErrCollectionNotFound
	}
	return nil
}

func (s *PostgresStore) UpdateCollection(ctx context.Context, userID, collectionID int64, update UpdateCollectionRequest) (GetCollectionResponse, error) {

	// Create SQL for name and description update if necessary
	var collectionUpdateSQL string
	collectionUpdateArgs := pgx.NamedArgs{}
	if update.Name != nil || update.Description != nil {
		var sets []string
		if update.Name != nil {
			sets = append(sets, "name = @name")
			collectionUpdateArgs["name"] = *update.Name
		}
		if update.Description != nil {
			sets = append(sets, "description = @description")
			collectionUpdateArgs["description"] = *update.Description
		}
		collectionUpdateArgs["collection_id"] = collectionID
		collectionUpdateSQL = fmt.Sprintf(`UPDATE collections.collections
                               SET %s
                               WHERE id = @collection_id`,
			strings.Join(sets, ","))
	}

	// Create SQL for DOI deletes if necessary
	var doiDeleteSQL string
	doiDeleteArgs := pgx.NamedArgs{}
	if len(update.DOIs.Remove) > 0 {
		var wheres []string
		for i, doi := range update.DOIs.Remove {
			doiVar := fmt.Sprintf("doi_%d", i)
			wheres = append(wheres, fmt.Sprintf("(collection_id = @collection_id AND doi = @%s)", doiVar))
			doiDeleteArgs[doiVar] = doi
		}
		doiDeleteArgs["collection_id"] = collectionID
		doiDeleteSQL = fmt.Sprintf(`DELETE FROM collections.dois WHERE %s`, strings.Join(wheres, " OR "))
	}

	// Create SQL for DOI adds if necessary
	var doiAddSQL string
	doiAddArgs := pgx.NamedArgs{}
	if len(update.DOIs.Add) > 0 {
		var values []string
		for i, doi := range update.DOIs.Add {
			doiVar := fmt.Sprintf("doi_%d", i)
			datasourceVar := fmt.Sprintf("datasource_%d", i)
			values = append(values, fmt.Sprintf("(@collection_id, @%s, @%s)", doiVar, datasourceVar))
			doiAddArgs[doiVar] = doi.Value
			doiAddArgs[datasourceVar] = doi.Datasource
		}
		doiAddArgs["collection_id"] = collectionID
		doiAddSQL = fmt.Sprintf(`INSERT INTO collections.dois (collection_id, doi, datasource) VALUES %s ON CONFLICT (collection_id, doi) DO NOTHING`, strings.Join(values, ", "))
	}

	conn, err := s.db.Connect(ctx, s.databaseName)
	if err != nil {
		return GetCollectionResponse{}, fmt.Errorf("UpdateCollection error connecting to database %s: %w", s.databaseName, err)
	}
	defer s.closeConn(ctx, conn)

	// Run any updates in a transaction
	if err := pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
		if len(collectionUpdateSQL) > 0 {
			commandTag, err := tx.Exec(ctx, collectionUpdateSQL, collectionUpdateArgs)
			if err != nil {
				return fmt.Errorf("error updating collection %d name/description: %w", collectionID, err)
			}
			if commandTag.RowsAffected() == 0 {
				return ErrCollectionNotFound
			}
		}
		// We can't really detect CollectionNotFound with the DOI queries, but we will catch it below when looking up
		// the updated collection to return. Plus the caller should have already tried to look up the collection for
		// authz purposes.
		if len(doiDeleteSQL) > 0 {
			if _, err := tx.Exec(ctx, doiDeleteSQL, doiDeleteArgs); err != nil {
				return fmt.Errorf("error deleting collection %d DOIs: %w", collectionID, err)
			}
		}

		if len(doiAddSQL) > 0 {
			if _, err := tx.Exec(ctx, doiAddSQL, doiAddArgs); err != nil {
				return fmt.Errorf("error adding collection %d DOIs: %w", collectionID, err)
			}
		}
		return nil
	}); err != nil {
		return GetCollectionResponse{}, fmt.Errorf("UpdateCollection error updating collection %d: %w", collectionID, err)
	}

	updatedCollection, err := getCollectionByID(ctx, conn, userID, collectionID)
	if err != nil {
		return GetCollectionResponse{}, fmt.Errorf("UpdateCollection error getting updated collection %d: %w", collectionID, err)
	}
	return updatedCollection, nil
}

func (s *PostgresStore) StartPublish(ctx context.Context, collectionID int64, userID int64) error {
	conn, err := s.db.Connect(ctx, s.databaseName)
	if err != nil {
		return fmt.Errorf("StartPublish error connecting to database %s: %w", s.databaseName, err)
	}
	defer s.closeConn(ctx, conn)

	query := `INSERT INTO collections.publish_status (collection_id, status, type, user_id, started_at)
              VALUES (@collection_id, @status, @type, @user_id, @started_at)
              ON CONFLICT (collection_id) DO UPDATE
                SET status = EXCLUDED.status,
                    type = EXCLUDED.type,
                    user_id = EXCLUDED.user_id,
                    started_at = EXCLUDED.started_at
                WHERE collections.publish_status.status != @in_progress`

	args := pgx.NamedArgs{
		"collection_id": collectionID,
		"status":        publishing.InProgressStatus,
		"type":          publishing.PublicationType,
		"user_id":       userID,
		"started_at":    time.Now().UTC(),
		"in_progress":   publishing.InProgressStatus,
	}

	tag, err := conn.Exec(ctx, query, args)
	if err != nil {
		return fmt.Errorf("error starting publish of collection %d for user %d: %w",
			collectionID,
			userID,
			err)
	}
	if tag.RowsAffected() == int64(0) {
		return ErrPublishInProgress
	}

	return nil

}

func (s *PostgresStore) closeConn(ctx context.Context, conn *pgx.Conn) {
	if err := conn.Close(ctx); err != nil {
		s.logger.Warn("error closing collections.PostgresStore DB connection", slog.Any("error", err))
	}
}
