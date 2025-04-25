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

var UpdateCollectionRouteKey = fmt.Sprintf("PATCH /{%s}", NodeIDPathParamKey)

func UpdateCollection(ctx context.Context, params Params) (dto.GetCollectionResponse, error) {
	nodeID := params.Request.PathParameters[NodeIDPathParamKey]
	if len(nodeID) == 0 {
		return dto.GetCollectionResponse{}, apierrors.NewBadRequestError(fmt.Sprintf(`missing %q path parameter`, NodeIDPathParamKey))
	}

	userClaim := params.Claims.UserClaim
	params.Container.AddLoggingContext(
		slog.String(NodeIDPathParamKey, nodeID),
		slog.String("userNodeId", userClaim.NodeId))

	requestBody := params.Request.Body
	if len(requestBody) == 0 {
		return dto.GetCollectionResponse{}, apierrors.NewBadRequestError("missing request body")
	}
	if logger := params.Container.Logger(); logger.Enabled(ctx, slog.LevelDebug) {
		logger.Debug("update collection request body", slog.String("body", requestBody))
	}

	var patchRequest dto.PatchCollectionRequest
	decoder := json.NewDecoder(strings.NewReader(requestBody))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&patchRequest); err != nil {
		return dto.GetCollectionResponse{}, apierrors.NewRequestUnmarshallError(patchRequest, err)
	}

	if err := ValidatePatchRequest(&patchRequest); err != nil {
		return dto.GetCollectionResponse{}, err
	}

	currentState, err := params.Container.CollectionsStore().GetCollection(ctx, userClaim.Id, nodeID)
	if err != nil {
		if errors.Is(err, store.ErrCollectionNotFound) {
			return dto.GetCollectionResponse{}, apierrors.NewCollectionNotFoundError(nodeID)
		}
		return dto.GetCollectionResponse{}, apierrors.NewInternalServerError(
			"error querying store for collection to update",
			err)
	}

	minRequiredRole := role.Editor
	if !currentState.UserRole.Implies(minRequiredRole) {
		return dto.GetCollectionResponse{}, apierrors.NewForbiddenError(
			fmt.Sprintf("collection %s not updated; requires user role: %s",
				nodeID,
				minRequiredRole),
		)
	}

	updateCollectionRequest, err := params.GetUpdateRequest(patchRequest, currentState)
	if err != nil {
		return dto.GetCollectionResponse{}, err
	}

	updateCollectionResponse, err := params.Container.CollectionsStore().UpdateCollection(ctx, userClaim.Id, currentState.ID, updateCollectionRequest)
	if err != nil {
		if errors.Is(err, store.ErrCollectionNotFound) {
			return dto.GetCollectionResponse{}, apierrors.NewCollectionNotFoundError(nodeID)
		}
		return dto.GetCollectionResponse{}, apierrors.NewInternalServerError(
			"error updating collection",
			err)
	}
	return params.StoreToDTOCollection(updateCollectionResponse)
}

func NewPatchCollectionRouteHandler() Handler[dto.GetCollectionResponse] {
	return Handler[dto.GetCollectionResponse]{
		HandleFunc:        UpdateCollection,
		SuccessStatusCode: http.StatusOK,
		Headers:           DefaultResponseHeaders(),
	}
}

func ValidatePatchRequest(request *dto.PatchCollectionRequest) error {
	if request.Name != nil {
		trimmedName := strings.TrimSpace(*request.Name)
		request.Name = &trimmedName
		if err := validate.CollectionName(*request.Name); err != nil {
			return err
		}

	}
	if request.Description != nil {
		trimmedDescription := strings.TrimSpace(*request.Description)
		request.Description = &trimmedDescription
		if err := validate.CollectionDescription(*request.Description); err != nil {
			return err
		}
	}
	return nil

}

func (p Params) GetUpdateRequest(patchRequest dto.PatchCollectionRequest, currentState store.GetCollectionResponse) (store.UpdateCollectionRequest, error) {
	storeRequest := store.UpdateCollectionRequest{}
	if patchRequest.Name != nil && *patchRequest.Name != currentState.Name {
		storeRequest.Name = patchRequest.Name
	}
	if patchRequest.Description != nil && *patchRequest.Description != currentState.Description {
		storeRequest.Description = patchRequest.Description
	}

	if patchRequest.DOIs == nil {
		return storeRequest, nil
	}

	existingDOIs := map[string]bool{}
	for _, doi := range currentState.DOIs {
		existingDOIs[doi] = true
	}

	for _, toDelete := range patchRequest.DOIs.Remove {
		if _, exists := existingDOIs[toDelete]; exists {
			storeRequest.DOIs.Remove = append(storeRequest.DOIs.Remove, toDelete)
		}
	}

	pennsieveDOIsToAdd, externalDOIs := CategorizeDOIs(p.Config.PennsieveConfig.DOIPrefix, patchRequest.DOIs.Add)
	if len(externalDOIs) > 0 {
		// We may later allow non-Pennsieve DOIs, but for now, this is an error
		return store.UpdateCollectionRequest{}, apierrors.NewBadRequestError(
			fmt.Sprintf("request contains non-Pennsieve DOIs: %s", strings.Join(externalDOIs, ", ")))
	}

	for _, toAdd := range pennsieveDOIsToAdd {
		if _, exists := existingDOIs[toAdd]; exists {
			storeRequest.DOIs.Add = append(storeRequest.DOIs.Add, toAdd)
		}
	}
	return storeRequest, nil
}
