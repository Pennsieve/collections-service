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
		routeKey := request.RouteKey
		logger := logging.Default.With(slog.String("routeKey", routeKey))
		container.SetLogger(logger)

		claims := authorizer.ParseClaims(request.RequestContext.Authorizer.Lambda)

		if claims == nil || claims.UserClaim == nil {
			err := apierrors.NewUnauthorizedError("no user claim in request")
			err.LogError(logger)
			return routes.APIErrorGatewayResponse(err), nil
		}

		routeParams := routes.Params{
			Request:   request,
			Container: container,
			Config:    config,
			Claims:    claims,
		}
		switch routeKey {
		case "POST /collections":
			return routes.Handle(ctx, routeParams, routes.NewCreateCollectionRouteHandler())
		case "GET /collections":
			return routes.Handle(ctx, routeParams, routes.NewGetCollectionsRouteHandler())
		default:
			routeNotFound := apierrors.NewError(fmt.Sprintf("route [%s] not found", routeKey), nil, http.StatusNotFound)
			routeNotFound.LogError(logger)
			return routes.APIErrorGatewayResponse(routeNotFound), nil
		}
	}
}
