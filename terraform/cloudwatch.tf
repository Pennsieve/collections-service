// Create log group for collections-service API Lambda.
resource "aws_cloudwatch_log_group" "collections_service_api_lambda_log_group" {
  name              = "/aws/lambda/${aws_lambda_function.collections_service_api_lambda.function_name}"
  retention_in_days = 30
  tags              = local.common_tags
}

// Collections SERVICE API GATEWAY
resource "aws_cloudwatch_log_group" "collections_service_gateway_log_group" {
  name = "${var.environment_name}/${var.service_name}/collections-api-gateway"

  retention_in_days = 30
}
