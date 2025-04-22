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
		logger := logging.Default.With(slog.String("routeKey", routeKey),
			slog.String("requestId", request.RequestContext.RequestID))
		container.SetLogger(logger)

		logger.Debug("configuration",
			slog.Group("postgres",
				slog.String("user", config.PostgresDB.User),
				slog.String("collectionsDatabase", config.PostgresDB.CollectionsDatabase),
			),
			slog.Group("pennsieve",
				slog.String("doiPrefix", config.PennsieveConfig.DOIPrefix),
				slog.String("discoverURL", config.PennsieveConfig.DiscoverServiceURL),
			),
		)

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
		case routes.CreateCollectionRouteKey:
			return routes.Handle(ctx, routes.NewCreateCollectionRouteHandler(), routeParams)
		case routes.GetCollectionsRouteKey:
			return routes.Handle(ctx, routes.NewGetCollectionsRouteHandler(), routeParams)
		case routes.GetCollectionRouteKey:
			return routes.Handle(ctx, routes.NewGetCollectionRouteHandler(), routeParams)
		case routes.DeleteCollectionRouteKey:
			return routes.Handle(ctx, routes.NewDeleteCollectionRouteHandler(), routeParams)
		default:
			routeNotFound := apierrors.NewError(fmt.Sprintf("route [%s] not found", routeKey), nil, http.StatusNotFound)
			routeNotFound.LogError(logger)
			return routes.APIErrorGatewayResponse(routeNotFound), nil
		}
	}
}
