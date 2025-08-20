package routes

import (
	"context"
	"errors"
	"fmt"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/publishing"
	"github.com/pennsieve/collections-service/internal/api/store/collections"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
	"log/slog"
	"net/http"
)

var UnpublishCollectionRouteKey = fmt.Sprintf("POST /{%s}/unpublish", NodeIDPathParamKey)

func UnpublishCollection(ctx context.Context, params Params) (dto.UnpublishCollectionResponse, error) {
	// Get all the inputs items
	nodeID := params.Request.PathParameters[NodeIDPathParamKey]
	if len(nodeID) == 0 {
		return dto.UnpublishCollectionResponse{}, apierrors.NewBadRequestError(fmt.Sprintf(`missing %q path parameter`, NodeIDPathParamKey))
	}

	userClaim := params.Claims.UserClaim
	params.Container.AddLoggingContext(
		slog.String(NodeIDPathParamKey, nodeID),
		slog.String("userNodeId", userClaim.NodeId))

	// Lookup info for the unpublish request to Discover
	collection, err := params.Container.CollectionsStore().GetCollection(ctx, userClaim.Id, nodeID)
	if err != nil {
		if errors.Is(err, collections.ErrCollectionNotFound) {
			return dto.UnpublishCollectionResponse{}, apierrors.NewCollectionNotFoundError(nodeID)
		}
		return dto.UnpublishCollectionResponse{}, apierrors.NewInternalServerError(
			"error querying store for collection to unpublish",
			err)
	}

	// Check permissions
	minRequiredRole := role.Owner
	if !collection.UserRole.Implies(minRequiredRole) {
		return dto.UnpublishCollectionResponse{}, apierrors.NewForbiddenError(
			fmt.Sprintf("collection %s not unpublished; requires user role: %s",
				nodeID,
				minRequiredRole),
		)
	}

	if err := validatePublishStatusForUnpublish(collection.Publication); err != nil {
		return dto.UnpublishCollectionResponse{}, err
	}

	return dto.UnpublishCollectionResponse{}, nil

}

func NewUnpublishCollectionRouteHandler() Handler[dto.UnpublishCollectionResponse] {
	return Handler[dto.UnpublishCollectionResponse]{
		HandleFunc:        UnpublishCollection,
		SuccessStatusCode: http.StatusOK,
		Headers:           DefaultResponseHeaders(),
	}
}

func validatePublishStatusForUnpublish(publication *collections.Publication) error {
	if publication == nil {
		return apierrors.NewConflictError("error unpublishing: collection has not been published")
	}
	if publication.Status == publishing.InProgressStatus {
		return apierrors.NewConflictError(fmt.Sprintf("error unpublishing: another publication process is already in progress: %s", publication.Type))
	}
	if publication.Type == publishing.RemovalType {
		return apierrors.NewConflictError("error unpublishing: collection already unpublished")
	}
	return nil
}
