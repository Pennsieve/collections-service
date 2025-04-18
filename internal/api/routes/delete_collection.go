package routes

import (
	"context"
	"fmt"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"net/http"
)

var DeleteCollectionRouteKey = fmt.Sprintf("DELETE /{%s}", NodeIDPathParamKey)

func DeleteCollection(ctx context.Context, params Params) (dto.NoContent, error) {
	return dto.NoContent{}, nil
}

func NewDeleteCollectionRouteHandler() Handler[dto.NoContent] {
	return Handler[dto.NoContent]{
		HandleFunc:        DeleteCollection,
		SuccessStatusCode: http.StatusNoContent,
	}
}
