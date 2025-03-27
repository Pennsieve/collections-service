package routes

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/shared/container"
	"github.com/pennsieve/pennsieve-go-core/pkg/authorizer"
	"log/slog"
	"net/http"
)

func CreateCollection(ctx context.Context, request events.APIGatewayV2HTTPRequest, container container.DependencyContainer, claims *authorizer.Claims) (dto.CollectionResponse, *apierrors.Error) {
	return dto.CollectionResponse{}, apierrors.NewError("not yet implemented", nil, http.StatusInternalServerError)
}

type CreateCollectionRoute struct {
	Logger *slog.Logger
}

func (c CreateCollectionRoute) Handle(ctx context.Context, request events.APIGatewayV2HTTPRequest, container container.DependencyContainer, claims *authorizer.Claims) (dto.CollectionResponse, *apierrors.Error) {
	return CreateCollection(ctx, request, container, claims)
}

func (c CreateCollectionRoute) GetLogger() *slog.Logger {
	return c.Logger
}

func (c CreateCollectionRoute) SuccessfulStatusCode() int {
	return http.StatusCreated
}

func (c CreateCollectionRoute) Headers() map[string]string {
	return DefaultHeaders
}
