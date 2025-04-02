package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/store"
	"github.com/pennsieve/collections-service/internal/api/validate"
	"net/http"
)

func CreateCollection(ctx context.Context, params Params) (dto.CollectionResponse, *apierrors.Error) {
	var createRequest dto.CreateCollectionRequest
	if err := json.Unmarshal([]byte(params.Request.Body), &createRequest); err != nil {
		return dto.CollectionResponse{}, apierrors.NewRequestUnmarshallError(createRequest, err)
	}
	ccParams := createCollectionParams{Params: params}

	if err := ccParams.ValidateCreateRequest(createRequest); err != nil {
		return dto.CollectionResponse{}, err
	}

	DeduplicateDOIs(&createRequest)

	collectionsStore := store.NewRDSCollectionsStore(params.Container.PostgresDB(),
		params.Config.PostgresDB.CollectionsDatabase,
		params.Logger)
	nodeID := uuid.NewString()

	storeResp, err := collectionsStore.CreateCollection(ctx, params.Claims.UserClaim.Id, nodeID, createRequest.Name, createRequest.Description, createRequest.DOIs)
	if err != nil {
		return dto.CollectionResponse{},
			apierrors.NewInternalServerError(fmt.Sprintf("error creating collection %s", createRequest.Name), err)
	}
	response := dto.CollectionResponse{
		NodeID:      nodeID,
		Name:        createRequest.Name,
		Description: createRequest.Description,
		Size:        len(createRequest.DOIs),
		UserRole:    storeResp.CreatorRole.String(),
	}
	return response, nil
}

func NewCreateCollectionRouteHandler() Handler[dto.CollectionResponse] {
	return Handler[dto.CollectionResponse]{
		Handle:            CreateCollection,
		SuccessStatusCode: http.StatusCreated,
		Headers:           DefaultResponseHeaders(),
	}
}

type createCollectionParams struct {
	Params
}

func (p createCollectionParams) ValidateCreateRequest(request dto.CreateCollectionRequest) *apierrors.Error {
	if err := validate.CollectionName(request.Name); err != nil {
		return err
	}
	if err := validate.CollectionDescription(request.Description); err != nil {
		return err
	}
	if err := validate.PennsieveDOIPrefix(p.Config.PennsieveConfig.DOIPrefix, request.DOIs...); err != nil {
		return err
	}
	return nil
}

func DeduplicateDOIs(createRequest *dto.CreateCollectionRequest) {
	seenDOIs := map[string]bool{}
	hasDups := false
	var deDuped []string
	// Maybe overly complicated, but trying to maintain order of the dois so that
	// if there are dups, we take the first one
	for _, doi := range createRequest.DOIs {
		if _, seen := seenDOIs[doi]; seen {
			hasDups = true
		} else {
			deDuped = append(deDuped, doi)
			seenDOIs[doi] = true
		}
	}
	if hasDups {
		createRequest.DOIs = deDuped
	}
}
