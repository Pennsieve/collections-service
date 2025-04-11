package routes

import (
	"context"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"log/slog"
	"net/http"
)

func GetCollection(ctx context.Context, params Params) (dto.GetCollectionResponse, error) {
	nodeID := params.Request.PathParameters["nodeId"]
	if len(nodeID) == 0 {
		return dto.GetCollectionResponse{}, apierrors.NewBadRequestError(`missing "nodeId" path parameter`)
	}
	params.Container.AddLoggingContext(slog.String("nodeId", nodeID))

	response := dto.GetCollectionResponse{}
	return response, nil
}

func NewGetCollectionRouteHandler() Handler[dto.GetCollectionResponse] {
	return Handler[dto.GetCollectionResponse]{
		HandleFunc:        GetCollection,
		SuccessStatusCode: http.StatusOK,
		Headers:           DefaultResponseHeaders(),
	}
}
