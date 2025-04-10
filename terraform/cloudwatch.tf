// Create log group for collections-service API Lambda.
resource "aws_cloudwatch_log_group" "collections_service_api_lambda_log_group" {
  name              = "/aws/lambda/${aws_lambda_function.collections_service_api_lambda.function_name}"
  retention_in_days = 30
  tags              = local.common_tags
}

// Send logs from collections-service API Lambda to Datadog
resource "aws_cloudwatch_log_subscription_filter" "collections_service_api_lambda_datadog_subscription" {
  name            = "${aws_cloudwatch_log_group.collections_service_api_lambda_log_group.name}-subscription"
  log_group_name  = aws_cloudwatch_log_group.collections_service_api_lambda_log_group.name
  filter_pattern  = ""
  destination_arn = data.terraform_remote_state.region.outputs.datadog_delivery_stream_arn
  role_arn        = data.terraform_remote_state.region.outputs.cw_logs_to_datadog_logs_firehose_role_arn
}

// Collections SERVICE API GATEWAY
resource "aws_cloudwatch_log_group" "collections_service_gateway_log_group" {
  name = "${var.environment_name}/${var.service_name}/collections-api-gateway"

  retention_in_days = 30
}
