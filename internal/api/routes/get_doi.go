package routes

import (
	"context"
	"errors"
	"fmt"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/collections-service/internal/api/store/collections"
	"log/slog"
	"net/http"
)

var GetDOIRouteKey = fmt.Sprintf("GET /{%s}/doi", NodeIDPathParamKey)

func GetDOI(ctx context.Context, params Params) (dto.GetLatestDOIResponse, error) {
	nodeID := params.Request.PathParameters[NodeIDPathParamKey]
	if len(nodeID) == 0 {
		return dto.GetLatestDOIResponse{}, apierrors.NewBadRequestError(fmt.Sprintf(`missing %q path parameter`, NodeIDPathParamKey))
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
			return dto.GetLatestDOIResponse{}, apierrors.NewCollectionNotFoundError(nodeID)
		}
		return dto.GetLatestDOIResponse{}, apierrors.NewInternalServerError(
			"error querying store for collection",
			err)
	}

	doiService, err := params.Container.DOI(ctx)
	if err != nil {
		return dto.GetLatestDOIResponse{}, apierrors.NewInternalServerError("error getting DOI service", err)
	}
	latestDOI, err := doiService.GetLatestDOI(ctx, storeResp.ID, nodeID, storeResp.UserRole)
	if err != nil {
		var notFoundErr service.LatestDOINotFoundError
		if errors.As(err, &notFoundErr) {
			return dto.GetLatestDOIResponse{}, apierrors.NewCollectionDOINotFoundError(storeResp.NodeID)
		}
		return dto.GetLatestDOIResponse{}, apierrors.NewInternalServerError("error calling DOI service", err)
	}
	return latestDOI, nil
}

func NewGetDOIRouteHandler() Handler[dto.GetLatestDOIResponse] {
	return Handler[dto.GetLatestDOIResponse]{
		HandleFunc:        GetDOI,
		SuccessStatusCode: http.StatusOK,
		Headers:           DefaultResponseHeaders(),
	}
}
