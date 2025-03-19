# collections-service

A service to manage Dataset collections

## Migrations

SQL migration files live in the `internal/dbmigrate/migrations` directory.

Jenkins runs `cmd/dbmigrate` against Postgres which uses [golang-migrate](https://github.com/golang-migrate/migrate).

In non-prod the migrations will be run when a PR is merged into `main`. In prod, it will be run as part of the deploy
job.

Use the `generate-migration-files.sh` script to create empty migration files in the appropriate place. It creates both
`{version}_{migration name}.up.sql` and `{version}_{migration name}.down.sql` files, the first for the migration
and the second to reverse the migration. This is what golang-migrate prefers/requires.
