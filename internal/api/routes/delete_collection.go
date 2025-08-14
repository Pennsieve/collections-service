package routes

import (
	"context"
	"errors"
	"fmt"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/store/collections"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
	"log/slog"
	"net/http"
)

var DeleteCollectionRouteKey = fmt.Sprintf("DELETE /{%s}", NodeIDPathParamKey)

func DeleteCollection(ctx context.Context, params Params) (dto.NoContent, error) {
	nodeID := params.Request.PathParameters[NodeIDPathParamKey]
	if len(nodeID) == 0 {
		return dto.NoContent{}, apierrors.NewBadRequestError(fmt.Sprintf(`missing %q path parameter`, NodeIDPathParamKey))
	}
	userClaim := params.Claims.UserClaim
	params.Container.AddLoggingContext(
		slog.String(NodeIDPathParamKey, nodeID),
		slog.String("userNodeId", userClaim.NodeId),
		slog.Int64("userId", userClaim.Id))

	userID, err := GetUserID(userClaim)
	if err != nil {
		return dto.NoContent{}, err
	}

	storeResp, err := params.Container.CollectionsStore().GetCollection(ctx, userID, nodeID)
	if err != nil {
		if errors.Is(err, collections.ErrCollectionNotFound) {
			return dto.NoContent{}, apierrors.NewCollectionNotFoundError(nodeID)
		}
		return dto.NoContent{}, apierrors.NewInternalServerError(
			"error querying store for collection to delete",
			err)
	}
	if !storeResp.UserRole.Implies(role.Manager) {
		return dto.NoContent{}, apierrors.NewForbiddenError(
			fmt.Sprintf("collection %s not deleted; requires user role: %s",
				nodeID,
				role.Manager),
		)
	}
	if err := params.Container.CollectionsStore().DeleteCollection(ctx, storeResp.ID); err != nil {
		return dto.NoContent{}, apierrors.NewInternalServerError("error deleting collection", err)
	}
	return dto.NoContent{}, nil
}

func NewDeleteCollectionRouteHandler() Handler[dto.NoContent] {
	return Handler[dto.NoContent]{
		HandleFunc:        DeleteCollection,
		SuccessStatusCode: http.StatusNoContent,
	}
}
