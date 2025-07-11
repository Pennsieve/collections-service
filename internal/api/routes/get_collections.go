package routes

import (
	"context"
	"fmt"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/shared/util"
	"net/http"
)

const GetCollectionsRouteKey = "GET /"

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

	// GetCollections only returns collections where the given user has >= Guest permission,
	// so no further authz is required for this route.
	storeResp, err := collectionsStore.GetCollections(ctx, userClaim.Id, limit, offset)
	if err != nil {
		return dto.GetCollectionsResponse{}, apierrors.NewInternalServerError(fmt.Sprintf("error getting collections for user %s", userClaim.NodeId), err)
	}

	response.TotalCount = storeResp.TotalCount

	// Gather all the banner DOIs to eventually look up banners in Discover
	var dois []string
	for _, storeCollection := range storeResp.Collections {
		for _, doi := range storeCollection.BannerDOIs {
			dois = append(dois, doi)
		}
	}

	var doiToPublicDataset map[string]dto.PublicDataset

	// For now we are assuming only PennsieveDOIs will be present in collections
	pennsieveDOIs, _ := CategorizeDOIs(params.Config.PennsieveConfig.DOIPrefix, dois)
	if len(pennsieveDOIs) > 0 {
		discoverResp, err := params.Container.Discover().GetDatasetsByDOI(ctx, pennsieveDOIs)
		if err != nil {
			return dto.GetCollectionsResponse{}, apierrors.NewInternalServerError(fmt.Sprintf("error looking up DOIs in Discover for user %s", userClaim.NodeId), err)
		}
		doiToPublicDataset = discoverResp.Published
	}

	for _, storeCollection := range storeResp.Collections {
		collectionDTO := dto.CollectionSummary{
			NodeID:      storeCollection.NodeID,
			Name:        storeCollection.Name,
			Description: storeCollection.Description,
			License:     util.SafeDeref(storeCollection.License),
			Tags:        storeCollection.Tags,
			Banners:     collectBanners(storeCollection.BannerDOIs, doiToPublicDataset),
			Size:        storeCollection.Size,
			UserRole:    storeCollection.UserRole.String(),
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
