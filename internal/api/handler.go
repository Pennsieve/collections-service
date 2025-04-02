package api

import (
	"context"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/pennsieve/collections-service/internal/shared/config"
	"github.com/pennsieve/collections-service/internal/shared/container"
	"github.com/pennsieve/pennsieve-go-core/pkg/authorizer"
)

type LambdaHandler func(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error)

func Handler() LambdaHandler {
	// initializes the dependency container once per Lambda invocation
	depContainer, err := container.NewContainer()
	if err != nil {
		log.Fatalf("Failed to initialize dependency container: %v", err)
	}

	return CollectionsServiceAPIHandler(depContainer, depContainer.Config)
}

func CollectionsServiceAPIHandler(
	container container.DependencyContainer,
	config config.Config,
) LambdaHandler {
	return func(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		claims := authorizer.ParseClaims(request.RequestContext.Authorizer.Lambda)

		if claims == nil || claims.UserClaim == nil {
			return events.APIGatewayV2HTTPResponse{
				StatusCode: http.StatusUnauthorized,
				Body:       "Unauthorized",
			}, nil
		}

		switch request.RouteKey {
		default:
			return events.APIGatewayV2HTTPResponse{
				StatusCode: http.StatusNotFound,
				Body:       "Not found",
			}, nil
		}
	}
}
