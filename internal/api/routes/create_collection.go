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
	collectionsStore := store.NewRDSCollectionsStore(params.Container.PostgresDB(), params.Config.PostgresDB.CollectionsDatabase)
	nodeID := uuid.NewString()

	if err := collectionsStore.CreateCollection(ctx, nodeID, createRequest.Name, createRequest.Description, createRequest.DOIs); err != nil {
		return dto.CollectionResponse{},
			apierrors.NewInternalServerError(fmt.Sprintf("error creating collection %s", createRequest.Name), err)
	}
	response := dto.CollectionResponse{
		NodeID:      nodeID,
		Name:        createRequest.Name,
		Description: createRequest.Description,
		Size:        len(createRequest.DOIs),
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
