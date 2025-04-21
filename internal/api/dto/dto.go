package dto

import (
	"encoding/json"
	"fmt"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
)

type DTO interface {
	Marshal() (string, error)
}

func defaultMarshalImpl(dto any) (string, error) {
	body, err := json.Marshal(dto)
	if err != nil {
		return "", apierrors.NewInternalServerError(fmt.Sprintf("error marshalling response body to %T", dto), err)
	}
	return string(body), nil
}

type NoContent struct{}

func (d NoContent) Marshal() (string, error) {
	return "", nil
}
