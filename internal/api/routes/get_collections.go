package routes

import (
	"context"
	"fmt"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/validate"
	"net/http"
	"strconv"
)

const DefaultGetCollectionsLimit = 10
const DefaultGetCollectionsOffset = 0

func GetCollections(ctx context.Context, params Params) (dto.CollectionsResponse, error) {
	limit, err := getIntQueryParam(params.Request.QueryStringParameters, "limit", 1, DefaultGetCollectionsLimit)
	if err != nil {
		return dto.CollectionsResponse{}, err
	}
	offset, err := getIntQueryParam(params.Request.QueryStringParameters, "offset", 0, DefaultGetCollectionsOffset)
	if err != nil {
		return dto.CollectionsResponse{}, err
	}
	response := dto.CollectionsResponse{
		Limit:  limit,
		Offset: offset,
	}
	collectionsStore := params.Container.CollectionsStore()
	userClaim := params.Claims.UserClaim

	storeResp, storeErr := collectionsStore.GetCollections(ctx, userClaim.Id, limit, offset)
	if storeErr != nil {
		return dto.CollectionsResponse{}, apierrors.NewInternalServerError(fmt.Sprintf("error getting collections for user %s", userClaim.NodeId), storeErr)
	}
	response.TotalCount = storeResp.TotalCount

	return response, nil
}

func NewGetCollectionsRouteHandler() Handler[dto.CollectionsResponse] {
	return Handler[dto.CollectionsResponse]{
		HandleFunc:        GetCollections,
		SuccessStatusCode: http.StatusOK,
		Headers:           DefaultResponseHeaders(),
	}
}

func getIntQueryParam(queryParams map[string]string, key string, requiredMin int, defaultValue int) (int, *apierrors.Error) {
	if strVal, present := queryParams[key]; present {
		value, err := strconv.Atoi(strVal)
		if err != nil {
			return 0, apierrors.NewBadRequestErrorWithCause(fmt.Sprintf("value of [%s] must be an integer", key), err)
		}
		if err := validate.IntQueryParamValue(key, value, requiredMin); err != nil {
			return 0, err
		}
		return value, nil
	}
	return defaultValue, nil
}
