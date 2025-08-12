package routes

import (
	"context"
	"errors"
	"fmt"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/collections-service/internal/api/store/collections"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
	"log/slog"
	"net/http"
)

const NodeIDPathParamKey = "nodeId"
const IncludePublishedDatasetQueryParamKey = "includePublishedDataset"

var GetCollectionRouteKey = fmt.Sprintf("GET /{%s}", NodeIDPathParamKey)

func GetCollection(ctx context.Context, params Params) (dto.GetCollectionResponse, error) {
	nodeID := params.Request.PathParameters[NodeIDPathParamKey]
	if len(nodeID) == 0 {
		return dto.GetCollectionResponse{}, apierrors.NewBadRequestError(fmt.Sprintf(`missing %q path parameter`, NodeIDPathParamKey))
	}
	includePublishedDataset, err := GetBoolQueryParam(params.Request.QueryStringParameters, IncludePublishedDatasetQueryParamKey, false)
	if err != nil {
		return dto.GetCollectionResponse{}, err
	}
	userClaim := params.Claims.UserClaim
	params.Container.AddLoggingContext(
		slog.String(NodeIDPathParamKey, nodeID),
		slog.String("userNodeId", userClaim.NodeId))

	// GetCollection only returns the collection if the given user has >= Guest permission,
	// so no further authz is required for this route.
	storeResp, err := params.Container.CollectionsStore().GetCollection(ctx, userClaim.Id, nodeID)
	if err != nil {
		if errors.Is(err, collections.ErrCollectionNotFound) {
			return dto.GetCollectionResponse{}, apierrors.NewCollectionNotFoundError(nodeID)
		}
		return dto.GetCollectionResponse{}, apierrors.NewInternalServerError(
			"error querying store for collection",
			err)
	}

	var datasetPublishStatusResp *service.DatasetPublishStatusResponse
	if includePublishedDataset && storeResp.Publication != nil {
		datasetPublishStatusResp, err = params.getDatasetPublishStatus(ctx, storeResp.ID, nodeID, storeResp.UserRole)
		if err != nil {
			return dto.GetCollectionResponse{}, err
		}
	}

	return params.StoreToDTOCollection(ctx, storeResp, datasetPublishStatusResp)
}

func NewGetCollectionRouteHandler() Handler[dto.GetCollectionResponse] {
	return Handler[dto.GetCollectionResponse]{
		HandleFunc:        GetCollection,
		SuccessStatusCode: http.StatusOK,
		Headers:           DefaultResponseHeaders(),
	}
}

func (p Params) getDatasetPublishStatus(ctx context.Context, collectionID int64, collectionNodeID string, userRole role.Role) (*service.DatasetPublishStatusResponse, error) {
	internalDiscover, err := p.Container.InternalDiscover(ctx)
	if err != nil {
		return nil,
			apierrors.NewInternalServerError("error getting Discover service", err)
	}
	datasetPublishStatus, err := internalDiscover.GetCollectionPublishStatus(ctx, collectionID, collectionNodeID, userRole)
	if err != nil {
		return nil, apierrors.NewInternalServerError("error getting publish status from Discover", err)
	}
	return &datasetPublishStatus, nil
}
