package routes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/collections-service/internal/api/store/collections"
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

	if err := validatePublishRequest(&publishRequest); err != nil {
		return dto.PublishCollectionResponse{}, err
	}

	collection, err := params.Container.CollectionsStore().GetCollection(ctx, userClaim.Id, nodeID)
	if err != nil {
		if errors.Is(err, collections.ErrCollectionNotFound) {
			return dto.PublishCollectionResponse{}, apierrors.NewCollectionNotFoundError(nodeID)
		}
		return dto.PublishCollectionResponse{}, apierrors.NewInternalServerError(
			"error querying store for collection to publish",
			err)
	}

	minRequiredRole := role.Owner
	if !collection.UserRole.Implies(minRequiredRole) {
		return dto.PublishCollectionResponse{}, apierrors.NewForbiddenError(
			fmt.Sprintf("collection %s not published; requires user role: %s",
				nodeID,
				minRequiredRole),
		)
	}

	pennsieveDOIs, _ := GroupByDatasource(collection.DOIs)

	banners := make([]string, 0)
	if len(pennsieveDOIs) > 0 {
		discoverDOIRes, err := params.Container.Discover().GetDatasetsByDOI(pennsieveDOIs)
		if err != nil {
			return dto.PublishCollectionResponse{}, apierrors.NewInternalServerError("error getting DOI info from Discover", err)
		}
		if len(discoverDOIRes.Unpublished) > 0 {
			return dto.PublishCollectionResponse{}, apierrors.NewBadRequestError(fmt.Sprintf("collection contains unpublished DOIs: %s", strings.Join(slices.Collect(maps.Keys(discoverDOIRes.Unpublished)), ", ")))
		}

		banners = collectBanners(pennsieveDOIs, discoverDOIRes.Published)
	}

	userResp, err := params.Container.UsersStore().GetUser(ctx, userClaim.Id)
	if err != nil {
		return dto.PublishCollectionResponse{}, apierrors.NewInternalServerError("error getting user information", err)
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

	internalDiscover, err := params.Container.InternalDiscover(ctx)
	if err != nil {
		return dto.PublishCollectionResponse{}, apierrors.NewInternalServerError("error getting internal Discover dependency", err)
	}
	discoverPubResp, err := internalDiscover.PublishCollection(collection.ID, collection.UserRole, discoverPubReq)
	if err != nil {
		return dto.PublishCollectionResponse{}, apierrors.NewInternalServerError("error publishing to Discover", err)
	}
	params.Container.Logger().Info("publish started on Discover",
		slog.Int64("publishedDatasetId", discoverPubResp.PublishedDatasetID),
		slog.Int64("publishedVersion", discoverPubResp.PublishedVersion),
		slog.String("status", discoverPubResp.Status),
		slog.String("ownerFirstName", discoverPubReq.OwnerFirstName),
		slog.String("ownerLastName", discoverPubReq.OwnerLastName),
		slog.String("ownerOrcid", discoverPubReq.OwnerORCID),
	)

	publishResponse := dto.PublishCollectionResponse{
		PublishedDatasetID: discoverPubResp.PublishedDatasetID,
		PublishedVersion:   discoverPubResp.PublishedVersion,
		Status:             discoverPubResp.Status,
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
