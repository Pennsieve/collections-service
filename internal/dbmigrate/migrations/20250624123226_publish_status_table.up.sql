CREATE TABLE publish_status
(
    collection_id INTEGER PRIMARY KEY,
    status        VARCHAR(50) NOT NULL,
    type          VARCHAR(50) NOT NULL,
    started_at    TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    finished_at   TIMESTAMP,
    user_id       INTEGER,
    FOREIGN KEY (collection_id) REFERENCES collections (id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES pennsieve.users (id) ON DELETE SET NULL
);