# Postgres connections variables used by dbmigrate via cloudwrap
resource "aws_ssm_parameter" "postgres_host" {
  name  = "/${var.environment_name}/${var.service_name}/postgres-host"
  type  = "String"
  value = data.terraform_remote_state.pennsieve_postgres.outputs.rds_proxy_endpoint
}

resource "aws_ssm_parameter" "postgres_user" {
  name  = "/${var.environment_name}/${var.service_name}/postgres-user"
  type  = "String"
  value = local.rds_proxy_user
}

resource "aws_ssm_parameter" "postgres_collections_database" {
  name  = "/${var.environment_name}/${var.service_name}/postgres-collections-database"
  type  = "String"
  value = local.pennsieve_postgres_database
}