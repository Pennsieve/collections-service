BEGIN;
ALTER TABLE dois
    ADD COLUMN IF NOT EXISTS datasource VARCHAR(255) DEFAULT 'External';

UPDATE dois
SET datasource = 'Pennsieve'
WHERE doi LIKE '10.26275/%'
   OR doi LIKE '10.21397/%';

ALTER TABLE dois
    ALTER COLUMN datasource DROP DEFAULT,
    ALTER COLUMN datasource SET NOT NULL;
COMMIT;