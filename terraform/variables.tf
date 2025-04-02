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
}
