CREATE TABLE IF NOT EXISTS collection_user
(
    collection_id  integer             not null
        references collections
            on delete cascade,
    user_id        integer             not null
        references pennsieve.users
            on delete cascade,
    permission_bit integer   default 0 not null,
    created_at     timestamp default CURRENT_TIMESTAMP,
    updated_at     timestamp default CURRENT_TIMESTAMP,
    role           varchar(50),
    primary key (collection_id, user_id)
);

create index collection_user_dataset_id_idx
    on collection_user (collection_id);

create index collection_user_user_id_idx
    on collection_user (user_id);

CREATE TRIGGER collection_user_update_updated_at
    BEFORE UPDATE
    ON collection_user
    FOR EACH ROW
EXECUTE PROCEDURE update_updated_at_column();
