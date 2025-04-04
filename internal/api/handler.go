package api

import (
	"context"
	"fmt"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/config"
	"github.com/pennsieve/collections-service/internal/api/container"
	"github.com/pennsieve/collections-service/internal/api/routes"
	"github.com/pennsieve/collections-service/internal/shared/logging"
	"log"
	"log/slog"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
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
		logger := logging.Default.With(slog.String("routeKey", request.RouteKey))

		claims := authorizer.ParseClaims(request.RequestContext.Authorizer.Lambda)

		if claims == nil || claims.UserClaim == nil {
			err := apierrors.NewUnauthorizedError("no user claim in request")
			err.LogError(logger)
			return routes.ErrorGatewayResponse(err), nil
		}

		routeParams := routes.Params{
			Request:   request,
			Container: container,
			Config:    config,
			Claims:    claims,
			Logger:    logger,
		}
		switch request.RouteKey {
		case "POST /collections":
			routeHandler := routes.NewCreateCollectionRouteHandler()
			return routes.Handle(ctx, routeParams, routeHandler)
		default:
			routeNotFound := apierrors.NewError(fmt.Sprintf("route [%s] not found", request.RouteKey), nil, http.StatusNotFound)
			routeNotFound.LogError(logger)
			return routes.ErrorGatewayResponse(routeNotFound), nil
		}
	}
}
