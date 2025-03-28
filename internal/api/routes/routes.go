package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/shared/config"
	"github.com/pennsieve/collections-service/internal/shared/container"
	"github.com/pennsieve/pennsieve-go-core/pkg/authorizer"
	"log/slog"
)

func DefaultResponseHeaders() map[string]string {
	return map[string]string{"content-type": "application/json"}
}

type Params struct {
	Request   events.APIGatewayV2HTTPRequest
	Container container.DependencyContainer
	Config    config.Config
	Claims    *authorizer.Claims
	Logger    *slog.Logger
}

type Func[T any] func(ctx context.Context, params Params) (T, *apierrors.Error)

type Handler[T any] struct {
	Handle            Func[T]
	SuccessStatusCode int
	Headers           map[string]string
}

func Handle[T any](ctx context.Context, params Params, handler Handler[T]) (events.APIGatewayV2HTTPResponse, error) {
	response, err := handler.Handle(ctx, params)
	if err != nil {
		err.LogError(params.Logger)
		return ErrorGatewayResponse(err), nil
	}
	body, marshalErr := json.Marshal(response)
	if marshalErr != nil {
		err = apierrors.NewInternalServerError(fmt.Sprintf("error marshalling response body to %T", response), marshalErr)
		err.LogError(params.Logger)
		return ErrorGatewayResponse(err), nil
	}
	return events.APIGatewayV2HTTPResponse{
		StatusCode: handler.SuccessStatusCode,
		Headers:    handler.Headers,
		Body:       string(body),
	}, nil
}

func ErrorGatewayResponse(err *apierrors.Error) events.APIGatewayV2HTTPResponse {
	return events.APIGatewayV2HTTPResponse{
		StatusCode: err.StatusCode,
		Headers:    DefaultResponseHeaders(),
		Body:       fmt.Sprintf(`{"message": %q, "error_id": %q}`, err.UserMessage, err.ID),
	}
}
