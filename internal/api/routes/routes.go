package routes

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/config"
	"github.com/pennsieve/collections-service/internal/api/container"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/validate"
	"github.com/pennsieve/collections-service/internal/shared/util"
	"github.com/pennsieve/pennsieve-go-core/pkg/authorizer"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/user"
	"log/slog"
	"strconv"
)

// Func is the function type to which all route-handling functions should conform.
// In addition, the error should always be an instance of *apierrors.Error.
// We do not have this in the return type below because of https://go.dev/doc/faq#nil_error
// The one problem I've seen is with testify's assert.NoError() function which fails to
// identify nil *apierrors.Error as a non-error.
type Func[T dto.DTO] func(ctx context.Context, params Params) (T, error)

type Params struct {
	Request   events.APIGatewayV2HTTPRequest
	Container container.DependencyContainer
	Config    config.Config
	Claims    *authorizer.Claims
}

type Handler[T dto.DTO] struct {
	HandleFunc        Func[T]
	SuccessStatusCode int
	Headers           map[string]string
}

func Handle[T dto.DTO](ctx context.Context, handler Handler[T], params Params) (events.APIGatewayV2HTTPResponse, error) {
	response, err := handler.HandleFunc(ctx, params)
	if err != nil {
		return handleError(err, params.Container.Logger())
	}
	body, err := response.Marshal()
	if err != nil {
		return handleError(err, params.Container.Logger())
	}
	return events.APIGatewayV2HTTPResponse{
		StatusCode: handler.SuccessStatusCode,
		Headers:    handler.Headers,
		Body:       body,
	}, nil
}

// DefaultResponseHeaders is a function instead of variable so that callers can
// modify the returned map without changing a package-wide variable.
func DefaultResponseHeaders() map[string]string {
	return map[string]string{"content-type": util.ApplicationJSON}
}

func DefaultErrorResponseHeaders() map[string]string {
	return map[string]string{"content-type": util.ApplicationJSON}
}

func APIErrorGatewayResponse(err *apierrors.Error) events.APIGatewayV2HTTPResponse {
	return events.APIGatewayV2HTTPResponse{
		StatusCode: err.StatusCode,
		Headers:    DefaultErrorResponseHeaders(),
		Body:       fmt.Sprintf(`{"message": %q, "errorId": %q}`, err.UserMessage, err.ID),
	}
}

func handleError(err error, logger *slog.Logger) (events.APIGatewayV2HTTPResponse, error) {
	var apiError *apierrors.Error
	if !errors.As(err, &apiError) {
		apiError = apierrors.NewInternalServerError("server error", err)
	}
	apiError.LogError(logger)

	return APIErrorGatewayResponse(apiError), nil

}

func GetIntQueryParam(queryParams map[string]string, key string, requiredMin int, defaultValue int) (int, error) {
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

func GetBoolQueryParam(queryParams map[string]string, key string, defaultValue bool) (bool, error) {
	if strVal, present := queryParams[key]; present {
		value, err := strconv.ParseBool(strVal)
		if err != nil {
			return false, apierrors.NewBadRequestErrorWithCause(fmt.Sprintf("value of [%s] must be a bool", key), err)

		}
		return value, nil
	}
	return defaultValue, nil
}

// GetUserID gets the user's id from userClaim and safely converts it from int64 to int32, returning an Internal Service apierrors.Error if this
// is not possible. The Postgres type of user id is integer, not bigint, so really only needs to be int32.
func GetUserID(userClaim *user.Claim) (int32, error) {
	userID, err := util.SafeInt64To32(userClaim.Id)
	if err != nil {
		return 0, apierrors.NewInternalServerError("error converting user id", err)
	}
	return userID, nil
}
