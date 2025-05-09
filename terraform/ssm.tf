# Postgres connections variables used by dbmigrate via cloudwrap
resource "aws_ssm_parameter" "postgres_host" {
  name  = "/${var.environment_name}/${var.dbmigrate_service_name}/postgres-host"
  type  = "String"
  value = data.terraform_remote_state.pennsieve_postgres.outputs.master_fqdn
}

resource "aws_ssm_parameter" "postgres_user" {
  name  = "/${var.environment_name}/${var.dbmigrate_service_name}/postgres-user"
  type  = "String"
  value = var.dbmigrate_postgres_user
}

resource "aws_ssm_parameter" "postgres_password" {
  name  = "/${var.environment_name}/${var.dbmigrate_service_name}/postgres-password"
  type  = "SecureString"
  value = "dummy"

  lifecycle {
    ignore_changes = [value]
  }
}

resource "aws_ssm_parameter" "postgres_database" {
  name  = "/${var.environment_name}/${var.dbmigrate_service_name}/postgres-database"
  type  = "String"
  value = var.pennsieve_postgres_database
}
