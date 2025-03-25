###################### COLLECTIONS SERVICE API LAMBDA #####################

resource "aws_lambda_function" "collections_service_api_lambda" {
  description   = "Lambda function for handling dataset collections management API requests"
  function_name = "${var.environment_name}-${var.service_name}-api-lambda-${data.terraform_remote_state.region.outputs.aws_region_shortname}"
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  architectures = ["arm64"]
  role          = aws_iam_role.collections_service_api_lambda_role.arn
  timeout       = 900
  memory_size   = 128
  s3_bucket     = var.lambda_bucket
  s3_key        = "${var.service_name}/${var.service_name}-api-${var.image_tag}.zip"

  vpc_config {
    subnet_ids = tolist(data.terraform_remote_state.vpc.outputs.private_subnet_ids)
    security_group_ids = [data.terraform_remote_state.platform_infrastructure.outputs.upload_v2_security_group_id]
  }

  environment {
    variables = {
      ENV    = var.environment_name
      REGION = var.aws_region

      POSTGRES_HOST                 = data.terraform_remote_state.pennsieve_postgres.outputs.rds_proxy_endpoint,
      POSTGRES_USER                 = "${var.environment_name}_rds_proxy_user",
      POSTGRES_COLLECTIONS_DATABASE = "pennsieve_postgres",
    }
  }
}
