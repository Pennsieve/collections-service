####################### COLLECTIONS SERVICE API LAMBDA POLICY #######################

resource "aws_iam_role" "collections_service_api_lambda_role" {
  name = "${var.environment_name}-${var.service_name}-api-lambda-role-${data.terraform_remote_state.region.outputs.aws_region_shortname}"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_iam_role_policy_attachment" "collections_service_api_lambda_iam_policy_attachment" {
  role       = aws_iam_role.collections_service_api_lambda_role.name
  policy_arn = aws_iam_policy.collections_service_api_lambda_iam_policy.arn
}

resource "aws_iam_policy" "collections_service_api_lambda_iam_policy" {
  name   = "${var.environment_name}-${var.service_name}-api-lambda-iam-policy-${data.terraform_remote_state.region.outputs.aws_region_shortname}"
  path   = "/"
  policy = data.aws_iam_policy_document.collections_service_api_iam_policy_document.json
}

data "aws_iam_policy_document" "collections_service_api_iam_policy_document" {

  statement {
    sid    = "CollectionsServiceAPILambdaLogsPermissions"
    effect = "Allow"
    actions = [
      "logs:CreateLogGroup",
      "logs:CreateLogStream",
      "logs:PutDestination",
      "logs:PutLogEvents",
      "logs:DescribeLogStreams"
    ]
    resources = ["*"]
  }

  statement {
    sid    = "CollectionsServiceAPILambdaEC2Permissions"
    effect = "Allow"
    actions = [
      "ec2:CreateNetworkInterface",
      "ec2:DescribeNetworkInterfaces",
      "ec2:DeleteNetworkInterface",
      "ec2:AssignPrivateIpAddresses",
      "ec2:UnassignPrivateIpAddresses"
    ]
    resources = ["*"]
  }

  statement {
    sid    = "CollectionsServiceAPIRDSPermissions"
    effect = "Allow"

    actions = [
      "rds-db:connect"
    ]

    resources = [local.rds_db_connect_arn]
  }

  statement {
    sid    = "CollectionsServiceAPISecretsManagerPermissions"
    effect = "Allow"

    actions = [
      "kms:Decrypt",
      "secretsmanager:GetSecretValue",
    ]

    resources = [
      data.aws_kms_key.ssm_kms_key.arn,
    ]
  }

  statement {
    sid    = "CollectionsServiceAPISSMPermissions"
    effect = "Allow"

    actions = [
      "ssm:GetParameter",
      "ssm:GetParameters",
      "ssm:GetParametersByPath",
    ]

    resources = [
      "arn:aws:ssm:${data.aws_region.current_region.name}:${data.aws_caller_identity.current.account_id}:parameter/${var.environment_name}/${var.service_name}/*"
    ]
  }

  statement {
    sid    = "S3BucketAccess"
    effect = "Allow"
    actions = [
      "s3:Get*",
      "s3:Put*",
      "s3:List*"
    ]

    resources = [
      data.terraform_remote_state.platform_infrastructure.outputs.discover_publish50_bucket_arn,
      "${data.terraform_remote_state.platform_infrastructure.outputs.discover_publish50_bucket_arn}/*",
    ]
  }
}
