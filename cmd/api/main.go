package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/pennsieve/collections-service/internal/api"
)

func main() {
	lambda.Start(api.Handler())
}
