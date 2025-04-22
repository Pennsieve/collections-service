#!/bin/bash

# Create a postgres Docker image that contains the Pennsieve API schema that includes the collections schema.
# Relies on being called from Makefile to ensure latest version of the collections-migrations image is used.
# So run 'make build-postgres' rather than this script directly.

set -eu

REGISTRY="pennsieve"
REPO="pennsievedb-collections"
ENVIRONMENT="${ENVIRONMENT:-local}"

if [[ "$ENVIRONMENT" != "local" ]]; then
    echo "Environment is $ENVIRONMENT. Getting tag from git log and pushing images"
    TAG="$(git log --name-only --oneline -2 | tail -n +2 | grep -E internal/dbmigrate/migrations | xargs basename -a | grep -E '^2.*\.sql$' | awk -F '_' '{print $1}' | sort -u | tail -n 1)"
else
    echo "Environment is local. Getting tag from local filesystem and not pushing images"
    TAG="$(find internal/dbmigrate/migrations | xargs basename -a | grep -E '^2.*\.sql$' | awk -F '_' '{print $1}' | sort -u | tail -n 1)"
fi

###################################
run_migrations() {
  echo -e "\nBuilding database container.\n"
  docker compose -f docker-compose.build-postgres.yml down -v --remove-orphans
  docker compose -f docker-compose.build-postgres.yml rm -f
  docker compose -f docker-compose.build-postgres.yml build --pull

  echo -e "\nStarting base pennsievedb seed container.\n"
  docker compose -f docker-compose.build-postgres.yml up -d base-pennsievedb && sleep 5
  container=$(docker compose -f docker-compose.build-postgres.yml ps base-pennsievedb | tail -n 1 | awk '{print $1}')

  echo -e "\nRunning migrations....\n"

  docker compose -f docker-compose.build-postgres.yml run collection-migrations

  while true; do
    HEALTH=$(docker inspect --format='{{.State.Health.Status}}' $container)
    [ "$HEALTH" != healthy ] || break
    sleep 1
  done

  echo -e "\nMigrations complete.\n"
}

create_container() {
  tag=$1
  container=$(docker compose -f docker-compose.build-postgres.yml ps base-pennsievedb | tail -n 1 | awk '{print $1}')
  container_id=$(docker inspect --format='{{.Id}}' $container)

  echo -e "\nCreating new ${REGISTRY}/${REPO}:$tag from $container_id\n"
  docker compose -f docker-compose.build-postgres.yml stop

  if [[ $tag =~ .*seed.* ]]; then
    docker commit $container_id $REGISTRY/$REPO:latest-seed
    docker tag $REGISTRY/$REPO:latest-seed $REGISTRY/$REPO:$tag

    if [[ "$ENVIRONMENT" != "local" ]]; then
      echo -e "\nPushing $REGISTRY/$REPO:latest-seed\n"
      docker push $REGISTRY/$REPO:latest-seed

      echo -e "\nPushing $REGISTRY/$REPO:$tag\n"
      docker push $REGISTRY/$REPO:$tag
    fi
  else
    docker commit $container_id $REGISTRY/$REPO
    docker tag $REGISTRY/$REPO:latest $REGISTRY/$REPO:$tag

    if [[ "$ENVIRONMENT" != "local" ]]; then
        echo -e "\nPushing $REGISTRY/$REPO:$tag\n"
        docker push $REGISTRY/$REPO:$tag

        echo -e "\nPushing $REGISTRY/$REPO:latest\n"
        docker push $REGISTRY/$REPO
    fi
  fi
}

###################################

echo -e "\nStarting build-postgres script...\n"

if [[ ! -z $TAG ]]; then
  #Only build seed since that is all we need for tests
  echo -e "\nCreating a new image with tag: ${TAG}-seed\n"
  run_migrations
  create_container $TAG-seed
  docker compose -f docker-compose.build-postgres.yml down -v
  docker compose -f docker-compose.build-postgres.yml rm -f
else
  echo -e "\nCould not find a valid tag. Not creating a ${REPO} image.\n"
fi

echo -e "\nThe build-postgres script is complete.\n"
