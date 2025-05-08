resource "aws_apigatewayv2_api" "collections_service_api" {
  name          = "Collections Serverless API"
  protocol_type = "HTTP"
  description   = "API for the lambda-based Collections API"
  cors_configuration {
    allow_origins     = local.cors_allowed_origins
    allow_methods = ["OPTIONS", "GET", "POST", "PATCH", "DELETE"]
    allow_headers = ["*"]
    allow_credentials = true
    expose_headers = ["*"]
    max_age           = 300
  }
  body = templatefile("${path.module}/collections-service.yml", {
    authorize_lambda_invoke_uri    = data.terraform_remote_state.api_gateway.outputs.authorizer_lambda_invoke_uri
    gateway_authorizer_role        = data.terraform_remote_state.api_gateway.outputs.authorizer_invocation_role
    collections_service_lambda_arn = aws_lambda_function.collections_service_api_lambda.arn
  })
}

resource "aws_apigatewayv2_api_mapping" "collections_service_api_map" {
  api_id          = aws_apigatewayv2_api.collections_service_api.id
  domain_name     = var.api_domain_name
  stage           = aws_apigatewayv2_stage.collections_service_gateway_stage.id
  api_mapping_key = "collections"

}

resource "aws_apigatewayv2_stage" "collections_service_gateway_stage" {
  api_id = aws_apigatewayv2_api.collections_service_api.id

  name        = "$default"
  auto_deploy = true

  access_log_settings {
    destination_arn = aws_cloudwatch_log_group.collections_service_gateway_log_group.arn

    format = jsonencode({
      requestId               = "$context.requestId"
      sourceIp                = "$context.identity.sourceIp"
      requestTime             = "$context.requestTime"
      protocol                = "$context.protocol"
      httpMethod              = "$context.httpMethod"
      resourcePath            = "$context.resourcePath"
      routeKey                = "$context.routeKey"
      status                  = "$context.status"
      responseLength          = "$context.responseLength"
      integrationErrorMessage = "$context.integrationErrorMessage"
    }
    )
  }
}

resource "aws_apigatewayv2_integration" "collections_service_integration" {
  api_id             = aws_apigatewayv2_api.collections_service_api.id
  integration_type   = "AWS_PROXY"
  connection_type    = "INTERNET"
  integration_method = "POST"
  integration_uri    = aws_lambda_function.collections_service_api_lambda.invoke_arn
}

resource "aws_lambda_permission" "collections_service_lambda_permission" {
  statement_id  = "AllowExecutionFromAPIGateway"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.collections_service_api_lambda.function_name
  principal     = "apigateway.amazonaws.com"

  source_arn = "${aws_apigatewayv2_api.collections_service_api.execution_arn}/*/*"
}