package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/datasource"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/store/collections"
	"github.com/pennsieve/collections-service/internal/api/validate"
	"log/slog"
	"net/http"
	"strings"
)

const CreateCollectionRouteKey = "POST /"

func CreateCollection(ctx context.Context, params Params) (dto.CreateCollectionResponse, error) {
	requestBody := params.Request.Body
	if len(requestBody) == 0 {
		return dto.CreateCollectionResponse{}, apierrors.NewBadRequestError("missing request body")
	}
	logger := params.Container.Logger()
	if logger.Enabled(ctx, slog.LevelDebug) {
		logger.Debug("create collection request body", slog.String("body", requestBody))
	}
	var createRequest dto.CreateCollectionRequest
	decoder := json.NewDecoder(strings.NewReader(requestBody))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&createRequest); err != nil {
		return dto.CreateCollectionResponse{}, apierrors.NewRequestUnmarshallError(createRequest, err)
	}
	ccParams := createCollectionParams{Params: params}

	if err := ccParams.ValidateCreateRequest(&createRequest); err != nil {
		return dto.CreateCollectionResponse{}, err
	}

	pennsieveDOIs, externalDOIs := CategorizeDOIs(ccParams.Config.PennsieveConfig.DOIPrefix, createRequest.DOIs)
	if len(externalDOIs) > 0 {
		// We may later allow non-Pennsieve DOIs, but for now, this is an error
		return dto.CreateCollectionResponse{}, apierrors.NewBadRequestError(
			fmt.Sprintf("request contains non-Pennsieve DOIs: %s", strings.Join(externalDOIs, ", ")))
	}

	nodeID := uuid.NewString()
	response := dto.CreateCollectionResponse{
		NodeID:      nodeID,
		Name:        createRequest.Name,
		Description: createRequest.Description,
		Size:        len(pennsieveDOIs),
	}
	var doisToAdd []collections.DOI
	if len(pennsieveDOIs) > 0 {
		datasetResults, err := ccParams.Container.Discover().GetDatasetsByDOI(ctx, pennsieveDOIs)
		if err != nil {
			return dto.CreateCollectionResponse{}, apierrors.NewInternalServerError("error looking up DOIs in Discover", err)
		}

		if err := ValidateDiscoverResponse(datasetResults); err != nil {
			return dto.CreateCollectionResponse{}, err
		}

		response.Banners = collectBanners(pennsieveDOIs, datasetResults.Published)

		for _, pennsieveDOI := range pennsieveDOIs {
			doisToAdd = append(doisToAdd, collections.DOI{
				Value:      pennsieveDOI,
				Datasource: datasource.Pennsieve,
			})
		}
	}
	collectionsStore := ccParams.Container.CollectionsStore()

	userID, err := GetUserID(params.Claims.UserClaim)
	if err != nil {
		return dto.CreateCollectionResponse{}, err
	}

	storeResp, err := collectionsStore.CreateCollection(ctx, userID, nodeID, createRequest.Name, createRequest.Description, doisToAdd)
	if err != nil {
		return dto.CreateCollectionResponse{}, apierrors.NewInternalServerError(fmt.Sprintf("error creating collection %s", createRequest.Name), err)
	}

	response.UserRole = storeResp.CreatorRole.String()

	return response, nil
}

func NewCreateCollectionRouteHandler() Handler[dto.CreateCollectionResponse] {
	return Handler[dto.CreateCollectionResponse]{
		HandleFunc:        CreateCollection,
		SuccessStatusCode: http.StatusCreated,
		Headers:           DefaultResponseHeaders(),
	}
}

type createCollectionParams struct {
	Params
}

// ValidateCreateRequest may alter the passed in request by trimming whitespace from request.Name and request.Description.
func (p createCollectionParams) ValidateCreateRequest(request *dto.CreateCollectionRequest) error {
	request.Name = strings.TrimSpace(request.Name)
	request.Description = strings.TrimSpace(request.Description)
	if err := validate.CollectionName(request.Name); err != nil {
		return err
	}
	if err := validate.CollectionDescription(request.Description); err != nil {
		return err
	}
	return nil
}
