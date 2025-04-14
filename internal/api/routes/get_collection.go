package routes

import (
	"context"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"log/slog"
	"net/http"
)

const GetCollectionRouteKey = "GET /{nodeId}"

func GetCollection(ctx context.Context, params Params) (dto.GetCollectionResponse, error) {
	nodeID := params.Request.PathParameters["nodeId"]
	if len(nodeID) == 0 {
		return dto.GetCollectionResponse{}, apierrors.NewBadRequestError(`missing "nodeId" path parameter`)
	}
	userClaim := params.Claims.UserClaim
	params.Container.AddLoggingContext(
		slog.String("nodeId", nodeID),
		slog.String("userNodeId", userClaim.NodeId))

	storeResp, err := params.Container.CollectionsStore().GetCollection(ctx, userClaim.Id, nodeID)
	if err != nil {
		return dto.GetCollectionResponse{}, apierrors.NewInternalServerError(
			"error querying store for collection",
			err)
	}
	if storeResp == nil {
		return dto.GetCollectionResponse{}, apierrors.NewCollectionNotFoundError(nodeID)
	}

	response := dto.GetCollectionResponse{
		CollectionResponse: dto.CollectionResponse{
			NodeID:      nodeID,
			Name:        storeResp.Name,
			Description: storeResp.Description,
			Banners:     nil,
			Size:        storeResp.Size,
			UserRole:    storeResp.UserRole,
		},
		Contributors: nil,
		Datasets:     nil,
	}

	pennsieveDOIs, _ := CategorizeDOIs(params.Config.PennsieveConfig.DOIPrefix, storeResp.DOIs)
	if len(pennsieveDOIs) > 0 {
		discoverResp, err := params.Container.Discover().GetDatasetsByDOI(pennsieveDOIs)
		if err != nil {
			return dto.GetCollectionResponse{}, apierrors.NewInternalServerError(
				"error querying Discover for datasets in collection",
				err)
		}
		for _, published := range discoverResp.Published {
			datasetDTO, err := dto.NewPennsieveDataset(published)
			if err != nil {
				return dto.GetCollectionResponse{}, apierrors.NewInternalServerError(
					"error marshalling Discover PublicDataset",
					err)
			}
			response.Datasets = append(response.Datasets, datasetDTO)
		}
	}
	return response, nil
}

func NewGetCollectionRouteHandler() Handler[dto.GetCollectionResponse] {
	return Handler[dto.GetCollectionResponse]{
		HandleFunc:        GetCollection,
		SuccessStatusCode: http.StatusOK,
		Headers:           DefaultResponseHeaders(),
	}
}
