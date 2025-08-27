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
	// Adding the cause to the user message, since it can be useful for the user to figure out how to fix the request
	return NewBadRequestErrorWithCause(fmt.Sprintf("error unmarshalling request body to %T: %v", bodyType, cause), cause)
}

func NewBadRequestError(userMessage string) *Error {
	return NewBadRequestErrorWithCause(userMessage, nil)
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

func NewForbiddenError(userMessage string) *Error {
	return NewError(userMessage, nil, http.StatusForbidden)
}

func NewCollectionNotFoundError(missingID string) *Error {
	return NewError(fmt.Sprintf("collection %s not found", missingID), nil, http.StatusNotFound)
}

func NewConflictError(userMessage string) *Error {
	return NewConflictErrorWithCause(userMessage, nil)
}

func NewConflictErrorWithCause(userMessage string, cause error) *Error {
	return NewError(userMessage, cause, http.StatusConflict)
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
	logger.Error("returning API error to caller",
		slog.Group("error",
			slog.String("id", e.ID),
			slog.String("userMessage", e.UserMessage),
			slog.Any("cause", cause),
		),
	)
}
