package routes

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/shared/container"
	"github.com/pennsieve/pennsieve-go-core/pkg/authorizer"
	"log/slog"
)

var DefaultHeaders = map[string]string{"content-type": "application/json"}

type Route[T any] interface {
	Handle(ctx context.Context, request events.APIGatewayV2HTTPRequest, container container.DependencyContainer, claims *authorizer.Claims) (T, *apierrors.Error)
	GetLogger() *slog.Logger
	SuccessfulStatusCode() int
	Headers() map[string]string
}

func Handle[T any](ctx context.Context, request events.APIGatewayV2HTTPRequest, container container.DependencyContainer, claims *authorizer.Claims, route Route[T]) (events.APIGatewayV2HTTPResponse, error) {
	response, err := route.Handle(ctx, request, container, claims)
	if err != nil {
		err.LogError(route.GetLogger())
		return err.GatewayResponse(), nil
	}
	body, marshalErr := json.Marshal(response)
	if marshalErr != nil {
		err = apierrors.NewInternalServerError(marshalErr)
		err.LogError(route.GetLogger())
		return err.GatewayResponse(), nil
	}
	return events.APIGatewayV2HTTPResponse{
		StatusCode: route.SuccessfulStatusCode(),
		Headers:    route.Headers(),
		Body:       string(body),
	}, nil
}
