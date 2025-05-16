CREATE INDEX collection_user_user_perm_coll_idx
    ON collections.collection_user (user_id, permission_bit, collection_id);

CREATE INDEX IF NOT EXISTS dois_collection_id_id_idx
    ON collections.dois (collection_id, id);
