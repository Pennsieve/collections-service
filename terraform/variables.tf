variable "aws_account" {}

variable "aws_region" {}

variable "environment_name" {}

variable "service_name" {}

variable "vpc_name" {}

variable "image_tag" {}

variable "lambda_bucket" {
  default = "pennsieve-cc-lambda-functions-use1"
}

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
}
