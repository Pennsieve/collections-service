package routes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/store"
	"github.com/pennsieve/collections-service/internal/api/validate"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
	"log/slog"
	"net/http"
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
		if errors.Is(err, store.ErrCollectionNotFound) {
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

	publishResponse := dto.PublishCollectionResponse{}
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
