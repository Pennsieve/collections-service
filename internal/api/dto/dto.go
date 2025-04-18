package dto

import (
	"encoding/json"
	"fmt"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"log/slog"
)

type DTO interface {
	Marshal(logger *slog.Logger) (string, error)
}

func defaultMarshalImpl(dto any, logger *slog.Logger) (string, error) {
	body, marshalErr := json.Marshal(dto)
	if marshalErr != nil {
		responseErr := apierrors.NewInternalServerError(fmt.Sprintf("error marshalling response body to %T", dto), marshalErr)
		responseErr.LogError(logger)
		return "", responseErr
	}
	return string(body), nil
}

type NoContent struct{}

func (d NoContent) Marshal(_ *slog.Logger) (string, error) {
	return "", nil
}
