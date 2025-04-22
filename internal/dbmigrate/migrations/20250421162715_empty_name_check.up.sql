ALTER TABLE collections
    ADD CONSTRAINT check_non_empty_name CHECK (TRIM(name) <> '');