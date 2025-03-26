CREATE TABLE IF NOT EXISTS dois
(
    id            SERIAL PRIMARY KEY,
    collection_id INTEGER      NOT NULL REFERENCES collections (id) ON DELETE CASCADE,
    doi           VARCHAR(255) NOT NULL,
    UNIQUE (collection_id, doi)
);

CREATE TRIGGER dois_update_updated_at
    BEFORE UPDATE
    ON dois
    FOR EACH ROW
EXECUTE PROCEDURE update_updated_at_column();
