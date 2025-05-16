CREATE INDEX collection_user_user_collection_idx
    ON collection_user (user_id, collection_id);

CREATE INDEX IF NOT EXISTS dois_collection_id_id_idx
    ON collections.dois (collection_id, id);
