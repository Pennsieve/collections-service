.PHONY: help clean test test-ci package publish docker-clean vet tidy docker-image-clean clean-ci

LAMBDA_BUCKET ?= "pennsieve-cc-lambda-functions-use1"
WORKING_DIR   ?= "$(shell pwd)"
SERVICE_NAME  ?= "collections-service"
API_PACKAGE_NAME  ?= "${SERVICE_NAME}-api-${IMAGE_TAG}.zip"
DBMIGRATE_IMAGE_NAME ?= "${SERVICE_NAME}-dbmigrate:${IMAGE_TAG}"

.DEFAULT: help

help:
	@echo "Make Help for $(SERVICE_NAME)"
	@echo ""
	@echo "make test			- run tests"
	@echo "make package			- build and zip services"
	@echo "make publish			- package and publish services to S3"
	@echo "make clean           - delete bin directory and shutdown any Docker services"

local-services:
	docker compose -f docker-compose.test.yml down --remove-orphans
	docker compose -f docker-compose.test.yml -f docker-compose.local.override.yml up -d pennsievedb

test: local-services
	go test -v ./...

test-ci:
	docker compose -f docker-compose.test.yml down --remove-orphans
	docker compose -f docker-compose.test.yml up --build --abort-on-container-exit --exit-code-from test

package:
	@echo "***************************"
	@echo "*   Building API lambda   *"
	@echo "***************************"
	@echo ""
		env GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o $(WORKING_DIR)/bin/api/bootstrap $(WORKING_DIR)/cmd/api; \
		cd $(WORKING_DIR)/bin/api/; \
		zip -r $(WORKING_DIR)/bin/api/$(API_PACKAGE_NAME) .
	@echo "************************************************"
	@echo "*   Building Collections dbmigrate container   *"
	@echo "************************************************"
	@echo ""
	docker buildx build --platform linux/amd64 -t $(DBMIGRATE_IMAGE_NAME) -f Dockerfile.cloudwrap-dbmigrate .

publish:
	@echo "*****************************"
	@echo "*   Publishing API lambda   *"
	@echo "*****************************"
	@echo ""
	aws s3 cp $(WORKING_DIR)/bin/api/$(API_PACKAGE_NAME) s3://$(LAMBDA_BUCKET)/$(SERVICE_NAME)/
	@echo "**************************************************"
	@echo "*   Publishing Collections dbmigrate container   *"
	@echo "**************************************************"
	@echo ""
	docker push $(DBMIGRATE_IMAGE_NAME)

# Spin down active docker containers.
docker-clean:
	docker compose -f docker-compose.test.yml down

docker-image-clean:
	docker rmi -f $(DBMIGRATE_IMAGE_NAME)

clean: docker-clean
		rm -rf $(WORKING_DIR)/bin

clean-ci: clean docker-image-clean

vet:
	go vet ./...

tidy:
	go mod tidy

