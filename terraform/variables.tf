variable "aws_account" {}

variable "aws_region" {}

variable "environment_name" {}

variable "service_name" {}

variable "vpc_name" {}

variable "image_tag" {}

variable "lambda_bucket" {
  default = "pennsieve-cc-lambda-functions-use1"
}

variable "api_domain_name" {}

variable "dbmigrate_service_name" {
  default = "collections-service-dbmigrate"
}

variable "dbmigrate_postgres_user" {}

variable "api_postgres_user" {}

variable "pennsieve_postgres_database" {
  default = "pennsieve_postgres"
}

locals {
  common_tags = {
    aws_account      = var.aws_account
    aws_region       = data.aws_region.current_region.name
    environment_name = var.environment_name
  }
  rds_db_connect_arn        = "${replace(replace(data.terraform_remote_state.pennsieve_postgres.outputs.rds_proxy_endpoint_arn, ":rds:", ":rds-db:"), ":db-proxy:", ":dbuser:")}/${var.api_postgres_user}"
  discover_service_host     = data.terraform_remote_state.discover_service.outputs.internal_fqdn
  pennsieve_doi_prefix      = var.environment_name == "prod" ? "10.26275" : "10.21397"
  collections_id_space_id   = data.terraform_remote_state.discover_service.outputs.doi_collections_id_space_id
  log_level                 = var.environment_name == "prod" ? "INFO" : "DEBUG"
  cors_allowed_origins      = var.environment_name == "prod" ? ["https://discover.pennsieve.io", "https://app.pennsieve.io"] : ["http://localhost:3000", "https://discover.pennsieve.net", "https://app.pennsieve.net"]
}
