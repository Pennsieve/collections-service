package routes

import (
	"context"
	"fmt"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"log/slog"
	"net/http"
)

const nodeIDPathParamKey = "nodeId"

var GetCollectionRouteKey = fmt.Sprintf("GET /{%s}", nodeIDPathParamKey)

func GetCollection(ctx context.Context, params Params) (dto.GetCollectionResponse, error) {
	nodeID := params.Request.PathParameters[nodeIDPathParamKey]
	if len(nodeID) == 0 {
		return dto.GetCollectionResponse{}, apierrors.NewBadRequestError(fmt.Sprintf(`missing %q path parameter`, nodeIDPathParamKey))
	}
	userClaim := params.Claims.UserClaim
	params.Container.AddLoggingContext(
		slog.String(nodeIDPathParamKey, nodeID),
		slog.String("userNodeId", userClaim.NodeId))

	storeResp, err := params.Container.CollectionsStore().GetCollection(ctx, userClaim.Id, nodeID)
	if err != nil {
		return dto.GetCollectionResponse{}, apierrors.NewInternalServerError(
			"error querying store for collection",
			err)
	}
	if storeResp == nil {
		return dto.GetCollectionResponse{}, apierrors.NewCollectionNotFoundError(nodeID)
	}

	response := dto.GetCollectionResponse{
		CollectionResponse: dto.CollectionResponse{
			NodeID:      nodeID,
			Name:        storeResp.Name,
			Description: storeResp.Description,
			Size:        storeResp.Size,
			UserRole:    storeResp.UserRole,
		},
	}

	mergedContributors := MergedContributors{}

	pennsieveDOIs, _ := CategorizeDOIs(params.Config.PennsieveConfig.DOIPrefix, storeResp.DOIs)
	if len(pennsieveDOIs) > 0 {
		discoverResp, err := params.Container.Discover().GetDatasetsByDOI(pennsieveDOIs)
		if err != nil {
			return dto.GetCollectionResponse{}, apierrors.NewInternalServerError(
				"error querying Discover for datasets in collection",
				err)
		}

		response.Banners = collectBanners(pennsieveDOIs, discoverResp.Published)

		for _, doi := range pennsieveDOIs {
			var datasetDTO dto.Dataset
			if published, foundPub := discoverResp.Published[doi]; foundPub {
				datasetDTO, err = dto.NewPennsieveDataset(published)
				if err != nil {
					return dto.GetCollectionResponse{}, apierrors.NewInternalServerError(
						"error marshalling Discover PublicDataset",
						err)
				}
				mergedContributors = mergedContributors.Append(published.Contributors...)
			} else if unpublished, foundUnPub := discoverResp.Unpublished[doi]; foundUnPub {
				datasetDTO, err = dto.NewTombstoneDataset(unpublished)
				if err != nil {
					return dto.GetCollectionResponse{}, apierrors.NewInternalServerError(
						"error marshalling Discover Tombstone", err)
				}

			} else {
				// info on the doi was not returned. This shouldn't really happen
				return dto.GetCollectionResponse{}, apierrors.NewInternalServerError(
					fmt.Sprintf("Discover did not return information on Pennsieve DOI %s", doi),
					nil)
			}
			response.Datasets = append(response.Datasets, datasetDTO)

		}
	}
	response.DerivedContributors = mergedContributors.Deduplicated()
	return response, nil
}

func NewGetCollectionRouteHandler() Handler[dto.GetCollectionResponse] {
	return Handler[dto.GetCollectionResponse]{
		HandleFunc:        GetCollection,
		SuccessStatusCode: http.StatusOK,
		Headers:           DefaultResponseHeaders(),
	}
}
