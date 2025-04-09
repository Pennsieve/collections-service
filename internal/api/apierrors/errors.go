package apierrors

import (
	"fmt"
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

func NewInternalServerError(userMessage string, cause error) *Error {
	if len(userMessage) == 0 {
		userMessage = "internal server error"
	}
	return NewError(userMessage, cause, http.StatusInternalServerError)
}

func NewRequestUnmarshallError(bodyType any, cause error) *Error {
	return NewBadRequestErrorWithCause(fmt.Sprintf("error unmarshalling request body to %T", bodyType), cause)
}

func NewBadRequestError(userMessage string) *Error {
	return NewError(userMessage, nil, http.StatusBadRequest)
}

func NewBadRequestErrorWithCause(userMessage string, cause error) *Error {
	return NewError(userMessage, cause, http.StatusBadRequest)
}

func NewUnauthorizedError(userMessage string) *Error {
	if len(userMessage) == 0 {
		userMessage = "unauthorized"
	}
	return NewError(userMessage, nil, http.StatusUnauthorized)
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
