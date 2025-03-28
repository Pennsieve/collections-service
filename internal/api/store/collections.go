package store

import (
	"context"
	"fmt"
	"github.com/pennsieve/collections-service/internal/shared/clients/postgres"
)

type RDSCollectionsStore struct {
	db           postgres.DB
	databaseName string
}

func NewRDSCollectionsStore(db postgres.DB, collectionsDatabaseName string) *RDSCollectionsStore {
	return &RDSCollectionsStore{
		db:           db,
		databaseName: collectionsDatabaseName,
	}
}

func (s *RDSCollectionsStore) CreateCollection(ctx context.Context, nodeID, name, description string, dois []string) error {
	conn, err := s.db.Connect(ctx, s.databaseName)
	if err != nil {
		return fmt.Errorf("error connecting to database %s: %w", s.databaseName, err)
	}
	defer conn.Close(ctx)
	//WITH new_collection AS (
	//    INSERT INTO collections.collections (name, description, node_id) VALUES ('tttestt', 'this is a test', '12345-abcdef')
	//        RETURNING id),
	//     t
	//         AS (INSERT
	//         INTO collections.dois (collection_id, doi)
	//             SELECT new_collection.id, doi
	//             FROM new_collection,
	//                  (VALUES ('doi-1'), ('doi-2')) as new_dois(doi))
	//INSERT
	//INTO collections.collection_user (collection_id, user_id, permission_bit, role)
	//SELECT id, 1, 32, 'owner'
	//FROM new_collection
	//RETURNING (select id from new_collection);

	_, err = conn.Exec(ctx, "")

	if err != nil {
		return fmt.Errorf("error inserting new collection %s: %w", name, err)
	}

	return nil
}
