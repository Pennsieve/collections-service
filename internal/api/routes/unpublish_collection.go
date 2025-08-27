package routes

import (
	"context"
	"errors"
	"fmt"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/publishing"
	"github.com/pennsieve/collections-service/internal/api/service"
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

	// Validate the status
	if err := validatePublishStatusForUnpublish(collection.Publication); err != nil {
		return dto.UnpublishCollectionResponse{}, err
	}

	// Set unpublish in progress
	if err := params.Container.CollectionsStore().StartPublish(ctx, collection.ID, userClaim.Id, publishing.RemovalType); err != nil {
		if errors.Is(err, collections.ErrPublishInProgress) {
			// deliberately leave publish status alone, i.e., no cleanup
			return dto.UnpublishCollectionResponse{}, apierrors.NewConflictError(err.Error())
		}

		return dto.UnpublishCollectionResponse{}, cleanupOnError(ctx,
			params.Container.Logger(),
			apierrors.NewInternalServerError("error registering start of unpublish", err),
			cleanupStatusIfExists(params.Container.CollectionsStore(), collection.ID))

	}

	// unpublish in Discover. Discover removes objects from S3, so we don't need to do it here.
	internalDiscover, err := params.Container.InternalDiscover(ctx)
	if err != nil {
		return dto.UnpublishCollectionResponse{},
			cleanupOnError(ctx,
				params.Container.Logger(),
				apierrors.NewInternalServerError("error getting internal Discover dependency", err),
				cleanupStatus(params.Container.CollectionsStore(), collection.ID),
			)
	}
	discoverUnpubResp, err := internalDiscover.UnpublishCollection(ctx, collection.ID, collection.NodeID, collection.UserRole)
	if err != nil {
		var apiError *apierrors.Error
		var neverPublishedError service.CollectionNeverPublishedError
		if errors.As(err, &neverPublishedError) {
			apiError = apierrors.NewConflictError("Discover reports collection not published")
		} else {
			apiError = apierrors.NewInternalServerError("error unpublishing with Discover", err)
		}
		return dto.UnpublishCollectionResponse{},
			cleanupOnError(ctx,
				params.Container.Logger(),
				apiError,
				cleanupStatus(params.Container.CollectionsStore(), collection.ID))
	}
	params.Container.Logger().Info("unpublished on Discover",
		slog.Any("publishedDatasetId", discoverUnpubResp.PublishedDatasetID),
		slog.Any("publishedVersion", discoverUnpubResp.PublishedVersionCount),
		slog.Any("status", discoverUnpubResp.Status),
		slog.Any("lastPublishedDate", discoverUnpubResp.LastPublishedDate),
		slog.String("name", discoverUnpubResp.Name),
		slog.Any("sourceOrganizationId", discoverUnpubResp.SourceOrganizationID),
		slog.Any("sourceDatasetId", discoverUnpubResp.SourceDatasetID),
	)

	collectionsServiceStatus := discoverUnpubResp.Status.ToPublishingStatus()

	// Mark unpublish as finished
	if err := params.Container.CollectionsStore().FinishPublish(ctx, collection.ID, collectionsServiceStatus, true); err != nil {
		return dto.UnpublishCollectionResponse{},
			cleanupOnError(ctx, params.Container.Logger(),
				apierrors.NewInternalServerError("error marking unpublish as complete", err),
				cleanupStatus(params.Container.CollectionsStore(), collection.ID),
			)
	}

	return dto.UnpublishCollectionResponse{
		PublishedDatasetID: discoverUnpubResp.PublishedDatasetID,
		PublishedVersion:   discoverUnpubResp.PublishedVersionCount,
		Status:             discoverUnpubResp.Status,
	}, nil

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
	if publication.Type == publishing.RemovalType && publication.Status == publishing.CompletedStatus {
		return apierrors.NewConflictError("error unpublishing: collection already unpublished")
	}
	return nil
}
