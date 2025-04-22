ALTER TABLE dois
    ADD CONSTRAINT check_non_empty_doi CHECK (TRIM(doi) <> '');
