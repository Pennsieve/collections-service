package routes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/publishing"
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/collections-service/internal/api/store/collections"
	"github.com/pennsieve/collections-service/internal/api/store/manifests"
	"github.com/pennsieve/collections-service/internal/api/store/users"
	"github.com/pennsieve/collections-service/internal/api/validate"
	"github.com/pennsieve/collections-service/internal/shared/util"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"strings"
)

var PublishCollectionRouteKey = fmt.Sprintf("POST /{%s}/publish", NodeIDPathParamKey)

func PublishCollection(ctx context.Context, params Params) (dto.PublishCollectionResponse, error) {
	// Get all the inputs items
	nodeID := params.Request.PathParameters[NodeIDPathParamKey]
	if len(nodeID) == 0 {
		return dto.PublishCollectionResponse{}, apierrors.NewBadRequestError(fmt.Sprintf(`missing %q path parameter`, NodeIDPathParamKey))
	}

	userClaim := params.Claims.UserClaim
	params.Container.AddLoggingContext(
		slog.String(NodeIDPathParamKey, nodeID),
		slog.String("userNodeId", userClaim.NodeId))

	requestBody := params.Request.Body
	if len(requestBody) == 0 {
		return dto.PublishCollectionResponse{}, apierrors.NewBadRequestError("missing request body")
	}
	if logger := params.Container.Logger(); logger.Enabled(ctx, slog.LevelDebug) {
		logger.Debug("publish collection request body", slog.String("body", requestBody))
	}

	var publishRequest dto.PublishCollectionRequest
	decoder := json.NewDecoder(strings.NewReader(requestBody))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&publishRequest); err != nil {
		return dto.PublishCollectionResponse{}, apierrors.NewRequestUnmarshallError(publishRequest, err)
	}

	// Validate the request body
	if err := validatePublishRequest(&publishRequest); err != nil {
		return dto.PublishCollectionResponse{}, err
	}

	// Lookup info for the initial publish request to Discover
	collection, err := params.Container.CollectionsStore().GetCollection(ctx, userClaim.Id, nodeID)
	if err != nil {
		if errors.Is(err, collections.ErrCollectionNotFound) {
			return dto.PublishCollectionResponse{}, apierrors.NewCollectionNotFoundError(nodeID)
		}
		return dto.PublishCollectionResponse{}, apierrors.NewInternalServerError(
			"error querying store for collection to publish",
			err)
	}

	// Check permissions
	minRequiredRole := role.Owner
	if !collection.UserRole.Implies(minRequiredRole) {
		return dto.PublishCollectionResponse{}, apierrors.NewForbiddenError(
			fmt.Sprintf("collection %s not published; requires user role: %s",
				nodeID,
				minRequiredRole),
		)
	}

	// Make sure there is no in-progress publish for this collection
	if err := params.Container.CollectionsStore().StartPublish(ctx, collection.ID, userClaim.Id, publishing.PublicationType); err != nil {
		if errors.Is(err, collections.ErrPublishInProgress) {
			// deliberately leave publish status alone, i.e., no cleanup
			return dto.PublishCollectionResponse{}, apierrors.NewConflictError(err.Error())
		}
		return dto.PublishCollectionResponse{},
			cleanupOnError(ctx,
				apierrors.NewInternalServerError("error registering start of publish", err),
				cleanupStatusIfExists(params.Container.CollectionsStore(), collection.ID),
			)
	}

	if len(collection.Description) == 0 {
		return dto.PublishCollectionResponse{}, cleanupOnError(
			ctx,
			apierrors.NewBadRequestError("published description cannot be empty"),
			cleanupStatus(params.Container.CollectionsStore(), collection.ID),
		)
	}

	pennsieveDOIs, _ := GroupByDatasource(collection.DOIs)

	banners := make([]string, 0)
	if len(pennsieveDOIs) > 0 {
		discoverDOIRes, err := params.Container.Discover().GetDatasetsByDOI(ctx, pennsieveDOIs)
		if err != nil {
			return dto.PublishCollectionResponse{},
				cleanupOnError(ctx,
					apierrors.NewInternalServerError("error getting DOI info from Discover", err),
					cleanupStatus(params.Container.CollectionsStore(), collection.ID),
				)
		}
		if len(discoverDOIRes.Unpublished) > 0 {
			return dto.PublishCollectionResponse{},
				cleanupOnError(ctx,
					apierrors.NewBadRequestError(fmt.Sprintf("collection contains unpublished DOIs: %s", strings.Join(slices.Collect(maps.Keys(discoverDOIRes.Unpublished)), ", "))),
					cleanupStatus(params.Container.CollectionsStore(), collection.ID),
				)
		}

		banners = collectBanners(pennsieveDOIs, discoverDOIRes.Published)
	}

	userResp, err := params.Container.UsersStore().GetUser(ctx, userClaim.Id)
	if err != nil {
		return dto.PublishCollectionResponse{},
			cleanupOnError(ctx,
				apierrors.NewInternalServerError("error getting user information", err),
				cleanupStatus(params.Container.CollectionsStore(), collection.ID),
			)
	}

	discoverPubReq := service.PublishDOICollectionRequest{
		Name:             collection.Name,
		Description:      collection.Description,
		Banners:          banners,
		DOIs:             collection.DOIs.Strings(),
		License:          publishRequest.License,
		Tags:             publishRequest.Tags,
		OwnerID:          userClaim.Id,
		OwnerNodeID:      userClaim.NodeId,
		OwnerFirstName:   util.SafeDeref(userResp.FirstName),
		OwnerLastName:    util.SafeDeref(userResp.LastName),
		OwnerORCID:       util.SafeDeref(userResp.ORCID),
		CollectionNodeID: collection.NodeID,
	}

	// Initiate publish to Discover
	internalDiscover, err := params.Container.InternalDiscover(ctx)
	if err != nil {
		return dto.PublishCollectionResponse{},
			cleanupOnError(ctx,
				apierrors.NewInternalServerError("error getting internal Discover dependency", err),
				cleanupStatus(params.Container.CollectionsStore(), collection.ID),
			)
	}
	discoverPubResp, err := internalDiscover.PublishCollection(ctx, collection.ID, collection.UserRole, discoverPubReq)
	if err != nil {
		return dto.PublishCollectionResponse{},
			cleanupOnError(ctx,
				apierrors.NewInternalServerError("error publishing to Discover", err),
				cleanupStatus(params.Container.CollectionsStore(), collection.ID),
			)
	}
	params.Container.Logger().Info("publish started on Discover",
		slog.Int64("publishedDatasetId", discoverPubResp.PublishedDatasetID),
		slog.Int64("publishedVersion", discoverPubResp.PublishedVersion),
		slog.String("status", discoverPubResp.Status),
		slog.String("ownerFirstName", discoverPubReq.OwnerFirstName),
		slog.String("ownerLastName", discoverPubReq.OwnerLastName),
		slog.String("ownerOrcid", discoverPubReq.OwnerORCID),
	)

	// Create manifest and copy to S3
	manifest, err := publishing.NewManifestBuilder().
		WithID(discoverPubResp.PublicID).
		WithPennsieveDatasetID(discoverPubResp.PublishedDatasetID).
		WithVersion(discoverPubResp.PublishedVersion).
		WithName(collection.Name).
		WithDescription(collection.Description).
		WithCreator(creator(userResp)).
		WithLicense(publishRequest.License).
		WithKeywords(publishRequest.Tags).
		WithReferences(pennsieveDOIs).
		Build()
	if err != nil {
		return dto.PublishCollectionResponse{},
			cleanupOnError(ctx,
				apierrors.NewInternalServerError("error creating manifest", err),
				cleanupStatus(params.Container.CollectionsStore(), collection.ID),
				finalizeDiscoverFailure(internalDiscover, discoverPubResp.PublishedDatasetID, discoverPubResp.PublishedVersion, collection),
			)
	}

	manifestKey := manifest.S3Key()
	saveManifestResp, err := params.Container.ManifestStore().SaveManifest(ctx, manifestKey, manifest)
	if err != nil {
		return dto.PublishCollectionResponse{},
			// assuming if this failed then there is nothing to clean up in S3
			cleanupOnError(ctx,
				apierrors.NewInternalServerError("error publishing manifest", err),
				cleanupStatus(params.Container.CollectionsStore(), collection.ID),
				finalizeDiscoverFailure(internalDiscover, discoverPubResp.PublishedDatasetID, discoverPubResp.PublishedVersion, collection),
			)
	}
	manifestS3VersionID := saveManifestResp.S3VersionID

	params.Container.Logger().Info("wrote manifest to S3",
		slog.String("key", manifestKey),
		slog.String("s3VersionId", manifestS3VersionID))

	discoverFinalizeReq := service.FinalizeDOICollectionPublishRequest{
		PublishedDatasetID: discoverPubResp.PublishedDatasetID,
		PublishedVersion:   discoverPubResp.PublishedVersion,
		PublishSuccess:     true,
		FileCount:          len(manifest.Files),
		TotalSize:          manifest.TotalSize(),
		ManifestKey:        manifestKey,
		ManifestVersionID:  manifestS3VersionID,
	}
	discoverFinalizeResp, err := internalDiscover.FinalizeCollectionPublish(ctx, collection.ID, collection.NodeID, collection.UserRole, discoverFinalizeReq)
	if err != nil {
		return dto.PublishCollectionResponse{},
			cleanupOnError(ctx,
				apierrors.NewInternalServerError("error finalizing publish with Discover", err),
				cleanupStatus(params.Container.CollectionsStore(), collection.ID),
				cleanupManifest(params.Container.ManifestStore(), manifestKey, manifestS3VersionID),
				finalizeDiscoverFailure(internalDiscover, discoverPubResp.PublishedDatasetID, discoverPubResp.PublishedVersion, collection),
			)
	}
	// Mark publish as finished
	// TODO: Make the final status a function of the status returned by Discover finalize instead of just assuming Completed
	if err := params.Container.CollectionsStore().FinishPublish(ctx, collection.ID, publishing.CompletedStatus, true); err != nil {
		return dto.PublishCollectionResponse{},
			cleanupOnError(ctx,
				apierrors.NewInternalServerError("error marking publish as complete", err),
				cleanupStatus(params.Container.CollectionsStore(), collection.ID),
				cleanupManifest(params.Container.ManifestStore(), manifestKey, manifestS3VersionID),
				finalizeDiscoverFailure(internalDiscover, discoverPubResp.PublishedDatasetID, discoverPubResp.PublishedVersion, collection),
			)
	}

	publishResponse := dto.PublishCollectionResponse{
		PublishedDatasetID: discoverPubResp.PublishedDatasetID,
		PublishedVersion:   discoverPubResp.PublishedVersion,
		Status:             discoverFinalizeResp.Status,
	}
	return publishResponse, nil
}

func NewPublishCollectionRouteHandler() Handler[dto.PublishCollectionResponse] {
	return Handler[dto.PublishCollectionResponse]{
		HandleFunc:        PublishCollection,
		SuccessStatusCode: http.StatusOK,
		Headers:           DefaultResponseHeaders(),
	}
}

func validatePublishRequest(publishRequest *dto.PublishCollectionRequest) error {
	trimmedLic := strings.TrimSpace(publishRequest.License)
	publishRequest.License = trimmedLic
	if err := validate.License(trimmedLic); err != nil {
		return err
	}
	if err := validate.Tags(publishRequest.Tags); err != nil {
		return err
	}
	return nil
}

func creator(user users.GetUserResponse) publishing.PublishedContributor {
	return publishing.PublishedContributor{
		FirstName:     util.SafeDeref(user.FirstName),
		LastName:      util.SafeDeref(user.LastName),
		Orcid:         util.SafeDeref(user.ORCID),
		MiddleInitial: util.SafeDeref(user.MiddleInitial),
		Degree:        util.SafeDeref(user.Degree),
	}
}

type cleanupFunc func(ctx context.Context) error

// cleanupStatus sets publishing status to failed. If a status does not already exist
// the error will be added to the cleanup errors.
func cleanupStatus(collectionsStore collections.Store, collectionID int64) cleanupFunc {
	return func(ctx context.Context) error {
		return collectionsStore.FinishPublish(ctx, collectionID, publishing.FailedStatus, true)
	}
}

// cleanupStatusIfExists sets publishing status to failed if a status exists, otherwise does nothing
func cleanupStatusIfExists(collectionsStore collections.Store, collectionID int64) cleanupFunc {
	return func(ctx context.Context) error {
		return collectionsStore.FinishPublish(ctx, collectionID, publishing.FailedStatus, false)
	}
}

func cleanupManifest(manifestStore manifests.Store, key string, s3VersionID string) cleanupFunc {
	return func(ctx context.Context) error {
		return manifestStore.DeleteManifestVersion(ctx, key, s3VersionID)
	}
}

func finalizeDiscoverFailure(discover service.InternalDiscover, publishedDatasetID, publishedVersion int64, collection collections.GetCollectionResponse) cleanupFunc {
	return func(ctx context.Context) error {
		request := service.FinalizeDOICollectionPublishRequest{
			PublishedDatasetID: publishedDatasetID,
			PublishedVersion:   publishedVersion,
			PublishSuccess:     false,
		}
		_, err := discover.FinalizeCollectionPublish(ctx, collection.ID, collection.NodeID, collection.UserRole, request)
		return err
	}
}

func cleanupOnError(ctx context.Context, originalErr *apierrors.Error, cleanups ...cleanupFunc) error {
	var cleanupErrs []string
	for _, cleanup := range cleanups {
		if cleanupErr := cleanup(ctx); cleanupErr != nil {
			cleanupErrs = append(cleanupErrs,
				fmt.Sprintf("in addition an error occured when running cleanup function: %s",
					cleanupErr))
		}
	}
	if len(cleanupErrs) == 0 {
		return originalErr
	}
	joined := strings.Join(cleanupErrs, "; ")
	var cause error
	if origCause := originalErr.Cause; origCause == nil {
		cause = fmt.Errorf("cleanup errors: %s", joined)
	} else {
		cause = fmt.Errorf("%w; %s", originalErr, joined)
	}
	return apierrors.NewError(originalErr.UserMessage, cause, originalErr.StatusCode)
}
