package api

import (
	"context"
	"fmt"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/routes"
	"github.com/pennsieve/collections-service/internal/shared/logging"
	"log"
	"log/slog"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
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

	return CollectionsServiceAPIHandler(depContainer)
}

func CollectionsServiceAPIHandler(
	container container.DependencyContainer,
) LambdaHandler {
	return func(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		claims := authorizer.ParseClaims(request.RequestContext.Authorizer.Lambda)

		if claims == nil || claims.UserClaim == nil {
			return events.APIGatewayV2HTTPResponse{
				StatusCode: http.StatusUnauthorized,
				Body:       "Unauthorized",
			}, nil
		}

		logger := logging.Default.With(slog.String("routeKey", request.RouteKey))

		switch request.RouteKey {
		case "POST /collections":
			route := routes.CreateCollectionRoute{Logger: logger}
			return routes.Handle(ctx, request, container, claims, route)
		default:
			routeNotFound := apierrors.NewError(fmt.Sprintf("route [%s] not found", request.RouteKey), nil, http.StatusNotFound)
			routeNotFound.LogError(logger)
			return routeNotFound.GatewayResponse(), nil
		}
	}
}
