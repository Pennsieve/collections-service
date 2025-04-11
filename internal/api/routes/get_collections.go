package routes

import (
	"context"
	"fmt"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"maps"
	"net/http"
	"slices"
)

const DefaultGetCollectionsLimit = 10
const DefaultGetCollectionsOffset = 0

func GetCollections(ctx context.Context, params Params) (dto.GetCollectionsResponse, error) {
	// any errors returned should be *apierrors.Error for correct status codes and better logging
	limit, apiErr := GetIntQueryParam(params.Request.QueryStringParameters, "limit", 0, DefaultGetCollectionsLimit)
	if apiErr != nil {
		return dto.GetCollectionsResponse{}, apiErr
	}
	offset, apiErr := GetIntQueryParam(params.Request.QueryStringParameters, "offset", 0, DefaultGetCollectionsOffset)
	if apiErr != nil {
		return dto.GetCollectionsResponse{}, apiErr
	}
	response := dto.GetCollectionsResponse{
		Limit:  limit,
		Offset: offset,
	}
	collectionsStore := params.Container.CollectionsStore()
	userClaim := params.Claims.UserClaim

	storeResp, err := collectionsStore.GetCollections(ctx, userClaim.Id, limit, offset)
	if err != nil {
		return dto.GetCollectionsResponse{}, apierrors.NewInternalServerError(fmt.Sprintf("error getting collections for user %s", userClaim.NodeId), err)
	}

	response.TotalCount = storeResp.TotalCount

	doiToBanner := map[string]string{}
	for _, storeCollection := range storeResp.Collections {
		for _, doi := range storeCollection.BannerDOIs {
			doiToBanner[doi] = ""
		}
	}

	var dois []string
	dois = slices.AppendSeq(dois, maps.Keys(doiToBanner))

	// For now we are assuming only PennsieveDOIs will be present in collections
	pennsieveDOIs, _ := CategorizeDOIs(params.Config.PennsieveConfig.DOIPrefix, dois)
	if len(pennsieveDOIs) > 0 {
		discoverResp, err := params.Container.Discover().GetDatasetsByDOI(pennsieveDOIs)
		if err != nil {
			return dto.GetCollectionsResponse{}, apierrors.NewInternalServerError(fmt.Sprintf("error looking up DOIs in Discover for user %s", userClaim.NodeId), err)
		}
		for doi, dataset := range discoverResp.Published {
			bannerOpt := dataset.Banner
			if bannerOpt != nil {
				doiToBanner[doi] = *dataset.Banner
			}
		}
	}

	for _, storeCollection := range storeResp.Collections {
		var banners []string
		for _, doi := range storeCollection.BannerDOIs {
			banners = append(banners, doiToBanner[doi])
		}
		collectionDTO := dto.CollectionResponse{
			NodeID:      storeCollection.NodeID,
			Name:        storeCollection.Name,
			Description: storeCollection.Description,
			Banners:     banners,
			Size:        storeCollection.Size,
			UserRole:    storeCollection.UserRole,
		}
		response.Collections = append(response.Collections, collectionDTO)
	}

	return response, nil
}

func NewGetCollectionsRouteHandler() Handler[dto.GetCollectionsResponse] {
	return Handler[dto.GetCollectionsResponse]{
		HandleFunc:        GetCollections,
		SuccessStatusCode: http.StatusOK,
		Headers:           DefaultResponseHeaders(),
	}
}
