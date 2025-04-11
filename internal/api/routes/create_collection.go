package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/validate"
	"net/http"
	"strings"
)

const CreateCollectionRouteKey = "POST /"

func CreateCollection(ctx context.Context, params Params) (dto.CollectionResponse, error) {
	if len(params.Request.Body) == 0 {
		return dto.CollectionResponse{}, apierrors.NewBadRequestError("no request body")
	}
	var createRequest dto.CreateCollectionRequest
	if err := json.Unmarshal([]byte(params.Request.Body), &createRequest); err != nil {
		return dto.CollectionResponse{}, apierrors.NewRequestUnmarshallError(createRequest, err)
	}
	ccParams := createCollectionParams{Params: params}

	if err := ccParams.ValidateCreateRequest(createRequest); err != nil {
		return dto.CollectionResponse{}, err
	}

	pennsieveDOIs, externalDOIs := CategorizeDOIs(ccParams.Config.PennsieveConfig.DOIPrefix, createRequest.DOIs)
	if len(externalDOIs) > 0 {
		// We may later allow non-Pennsieve DOIs, but for now, this is an error
		return dto.CollectionResponse{}, apierrors.NewBadRequestError(
			fmt.Sprintf("request contains non-Pennsieve DOIs: %s", strings.Join(externalDOIs, ", ")))
	}

	nodeID := uuid.NewString()
	response := dto.CollectionResponse{
		NodeID:      nodeID,
		Name:        createRequest.Name,
		Description: createRequest.Description,
		Size:        len(pennsieveDOIs),
	}
	if len(pennsieveDOIs) > 0 {
		datasetResults, err := ccParams.Container.Discover().GetDatasetsByDOI(pennsieveDOIs)
		if err != nil {
			return dto.CollectionResponse{}, apierrors.NewInternalServerError("error looking up DOIs in Discover", err)
		}

		if len(datasetResults.Unpublished) > 0 {
			var details []string
			for _, unpublished := range datasetResults.Unpublished {
				details = append(details, fmt.Sprintf("%s status is %s", unpublished.DOI, unpublished.Status))
			}
			return dto.CollectionResponse{}, apierrors.NewBadRequestError(fmt.Sprintf("request contains unpublished DOIs: %s", strings.Join(details, ", ")))
		}

		response.Banners = collectBanners(createRequest.DOIs, datasetResults.Published)

	}
	collectionsStore := ccParams.Container.CollectionsStore()

	storeResp, err := collectionsStore.CreateCollection(ctx, params.Claims.UserClaim.Id, nodeID, createRequest.Name, createRequest.Description, pennsieveDOIs)
	if err != nil {
		return dto.CollectionResponse{},
			apierrors.NewInternalServerError(fmt.Sprintf("error creating collection %s", createRequest.Name), err)
	}

	response.UserRole = storeResp.CreatorRole.String()

	return response, nil
}

func NewCreateCollectionRouteHandler() Handler[dto.CollectionResponse] {
	return Handler[dto.CollectionResponse]{
		HandleFunc:        CreateCollection,
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
	return nil
}
