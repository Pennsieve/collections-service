# collections-service

A service to manage Dataset collections

## Deployments

The collections-service will be deployed into non-prod when a PR is merged into the main branch. Any new
database migrations will be run against the non-prod database at that time as well.

To deploy a new version to prod, run the
[service-deploy/pennsieve-prod/.../collections-service-release](https://jenkins.pennsieve.cc/job/service-deploy/job/pennsieve-prod/job/us-east-1/job/prod-vpc-use1/job/prod/job/collections-service-release/)
job with the desired image tag. This will take care of running any required database migrations against the prod
database as well as deploying a new Lambda for the service.

## Migrations

The collections-service uses its own schema called `collections` in the `pennsieve_postgres` database.

SQL migration files for the `collections` schema live in the `internal/dbmigrate/migrations` directory.

Jenkins runs `cmd/dbmigrate` against Postgres which uses [golang-migrate](https://github.com/golang-migrate/migrate) to
manage and track migrations.

In non-prod the migrations will be run when a PR is merged into `main`. In prod, it will be run as part of the
`collections-service-release` job under `service-deploy/pennsieve-prod`.

Use the `generate-migration-files.sh` script to create empty migration files in the appropriate place. It creates both
`{version}_{migration name}.up.sql` and `{version}_{migration name}.down.sql` files, the first for the migration
and the second to reverse the migration. This is what golang-migrate prefers/requires.
