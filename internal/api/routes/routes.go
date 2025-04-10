package routes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/config"
	"github.com/pennsieve/collections-service/internal/api/container"
	"github.com/pennsieve/collections-service/internal/api/validate"
	"github.com/pennsieve/collections-service/internal/shared/util"
	"github.com/pennsieve/pennsieve-go-core/pkg/authorizer"
	"log/slog"
	"net/http"
	"strconv"
)

// Func is the function type that all route-handling functions should conform to.
// In addition, the error should always be an instance of *apierrors.Error.
// We do not have this in the return type below because of https://go.dev/doc/faq#nil_error
// The one problem I've seen is with testify's assert.NoError() function which fails to
// identify nil *apierrors.Error as a non-error.
type Func[T any] func(ctx context.Context, params Params) (T, error)

type Params struct {
	Request   events.APIGatewayV2HTTPRequest
	Container container.DependencyContainer
	Config    config.Config
	Claims    *authorizer.Claims
}

type Handler[T any] struct {
	HandleFunc        Func[T]
	SuccessStatusCode int
	Headers           map[string]string
}

func Handle[T any](ctx context.Context, params Params, handler Handler[T]) (events.APIGatewayV2HTTPResponse, error) {
	response, err := handler.HandleFunc(ctx, params)
	if err != nil {
		var apiError *apierrors.Error
		if errors.As(err, &apiError) {
			apiError.LogError(params.Container.Logger())
			return APIErrorGatewayResponse(apiError), nil
		} else {
			params.Container.Logger().Error("handler returned a non-apierrors error; consider modifying route handler to always return an *apierrors.Error",
				slog.Any("cause", err))
			return StdErrorGatewayResponse(err), nil
		}
	}
	body, marshalErr := json.Marshal(response)
	if marshalErr != nil {
		responseErr := apierrors.NewInternalServerError(fmt.Sprintf("error marshalling response body to %T", response), marshalErr)
		responseErr.LogError(params.Container.Logger())
		return APIErrorGatewayResponse(responseErr), nil
	}
	return events.APIGatewayV2HTTPResponse{
		StatusCode: handler.SuccessStatusCode,
		Headers:    handler.Headers,
		Body:       string(body),
	}, nil
}

// DefaultResponseHeaders is a function instead of variable so that callers can
// modify the returned map without changing a package-wide variable.
func DefaultResponseHeaders() map[string]string {
	return map[string]string{"content-type": util.ApplicationJSON}
}

func APIErrorGatewayResponse(err *apierrors.Error) events.APIGatewayV2HTTPResponse {
	return events.APIGatewayV2HTTPResponse{
		StatusCode: err.StatusCode,
		Headers:    DefaultResponseHeaders(),
		Body:       fmt.Sprintf(`{"message": %q, "error_id": %q}`, err.UserMessage, err.ID),
	}
}

func StdErrorGatewayResponse(err error) events.APIGatewayV2HTTPResponse {
	return events.APIGatewayV2HTTPResponse{
		StatusCode: http.StatusInternalServerError,
		Headers:    DefaultResponseHeaders(),
		Body:       fmt.Sprintf(`{"message": %q}`, err.Error()),
	}
}

func GetIntQueryParam(queryParams map[string]string, key string, requiredMin int, defaultValue int) (int, *apierrors.Error) {
	if strVal, present := queryParams[key]; present {
		value, err := strconv.Atoi(strVal)
		if err != nil {
			return 0, apierrors.NewBadRequestErrorWithCause(fmt.Sprintf("value of [%s] must be an integer", key), err)
		}
		if err := validate.IntQueryParamValue(key, value, requiredMin); err != nil {
			return 0, err
		}
		return value, nil
	}
	return defaultValue, nil
}
