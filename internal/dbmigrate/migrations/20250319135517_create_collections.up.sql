CREATE TABLE IF NOT EXISTS collections
(
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(255)        NOT NULL,
    description VARCHAR(255),
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    node_id     VARCHAR(255) UNIQUE NOT NULL
);


CREATE TRIGGER collections_update_updated_at
    BEFORE UPDATE
    ON collections
    FOR EACH ROW
EXECUTE PROCEDURE update_updated_at_column();
