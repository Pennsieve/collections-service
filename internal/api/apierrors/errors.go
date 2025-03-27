package apierrors

import (
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/google/uuid"
	"log/slog"
	"net/http"
)

type Error struct {
	UserMessage string
	Cause       error
	StatusCode  int
	ID          string
}

func NewError(userMessage string, cause error, statusCode int) *Error {
	return &Error{
		UserMessage: userMessage,
		Cause:       cause,
		StatusCode:  statusCode,
		ID:          uuid.NewString(),
	}
}

func NewInternalServerError(cause error) *Error {
	return NewError("internal server error", cause, http.StatusInternalServerError)
}

func NewCollectionNotFoundError(missingID string) *Error {
	return NewError(fmt.Sprintf("collection %s not found", missingID), nil, http.StatusNotFound)
}

func (e *Error) Error() string {
	if e.Cause == nil {
		return e.UserMessage
	}
	return e.Cause.Error()
}

func (e *Error) LogError(logger *slog.Logger) {
	var cause string
	if e.Cause == nil {
		cause = "none"
	} else {
		cause = e.Cause.Error()
	}
	logger.Error(e.UserMessage,
		slog.Group("error",
			slog.String("id", e.ID),
			slog.Any("cause", cause),
		),
	)
}

func (e *Error) GatewayResponse() events.APIGatewayV2HTTPResponse {
	return events.APIGatewayV2HTTPResponse{
		StatusCode: e.StatusCode,
		Headers:    map[string]string{"content-type": "application/json"},
		Body:       fmt.Sprintf(`{"message": %q, "id": %q}`, e.UserMessage, e.ID),
	}
}
