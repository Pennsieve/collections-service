variable "aws_account" {}

variable "aws_region" {}

variable "environment_name" {}

variable "service_name" {}

variable "vpc_name" {}

variable "image_tag" {}

variable "lambda_bucket" {
  default = "pennsieve-cc-lambda-functions-use1"
}

locals {
  common_tags = {
    aws_account      = var.aws_account
    aws_region       = data.aws_region.current_region.name
    environment_name = var.environment_name
  }
  pennsieve_postgres_database = "pennsieve_postgres"
  rds_proxy_user              = "${var.environment_name}_rds_proxy_user"
  rds_db_connect_arn          = "${replace(replace(data.terraform_remote_state.pennsieve_postgres.outputs.rds_proxy_endpoint_arn, ":rds:", ":rds-db:"), ":db-proxy:", ":dbuser:")}/${local.rds_proxy_user}"
  discover_service_host       = data.terraform_remote_state.discover_service.outputs.internal_fqdn
  pennsieve_doi_prefix        = var.environment_name == "prod" ? "10.26275" : "10.21397"
}
