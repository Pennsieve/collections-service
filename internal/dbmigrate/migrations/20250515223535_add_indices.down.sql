DROP INDEX IF EXISTS collection_user_user_collection_idx;

DROP INDEX IF EXISTS dois_collection_id_id_idx;

CREATE INDEX IF NOT EXISTS collection_user_user_id_idx
    on collection_user (user_id);