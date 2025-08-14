package routes

import (
	"context"
	"fmt"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/publishing"
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/collections-service/internal/api/store/collections"
	"github.com/pennsieve/collections-service/internal/shared/util"
)

func ToDTOPublication(storePublication *collections.Publication, fromDiscover *service.DatasetPublishStatusResponse) *dto.Publication {
	publication := dto.Publication{}

	if fromDiscover != nil {
		publication.PublishedDataset = &dto.PublishedDataset{
			ID:                fromDiscover.PublishedDatasetID,
			Version:           fromDiscover.PublishedVersionCount,
			LastPublishedDate: fromDiscover.LastPublishedDate,
		}
	}

	if storePublication != nil {
		publication.Status = storePublication.Status
		publication.Type = storePublication.Type
	} else {
		publication.Status = publishing.DraftStatus
	}

	return &publication
}

func (p Params) StoreToDTOCollection(ctx context.Context, storeCollection collections.GetCollectionResponse, datasetPublishStatus *service.DatasetPublishStatusResponse) (dto.GetCollectionResponse, error) {
	response := dto.GetCollectionResponse{
		CollectionSummary: dto.CollectionSummary{
			NodeID:      storeCollection.NodeID,
			Name:        storeCollection.Name,
			Description: storeCollection.Description,
			Size:        storeCollection.Size,
			UserRole:    storeCollection.UserRole.String(),
			License:     util.SafeDeref(storeCollection.License),
			Tags:        storeCollection.Tags,
			Publication: ToDTOPublication(storeCollection.Publication, datasetPublishStatus),
		},
	}
	if publication := storeCollection.Publication; publication != nil {
		response.Publication.Status = publication.Status
		response.Publication.Type = publication.Type
	} else {
		response.Publication.Status = publishing.DraftStatus
	}

	mergedContributors := MergedContributors{}

	pennsieveDOIs, _ := GroupByDatasource(storeCollection.DOIs)
	if len(pennsieveDOIs) > 0 {
		discoverResp, err := p.Container.Discover().GetDatasetsByDOI(ctx, pennsieveDOIs)
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
						fmt.Sprintf("error marshalling Discover PublicDataset %s", doi),
						err)
				}
				mergedContributors = mergedContributors.Append(published.Contributors...)
			} else if unpublished, foundUnPub := discoverResp.Unpublished[doi]; foundUnPub {
				datasetDTO, err = dto.NewTombstoneDataset(unpublished)
				if err != nil {
					return dto.GetCollectionResponse{}, apierrors.NewInternalServerError(
						fmt.Sprintf("error marshalling Discover Tombstone %s", doi), err)
				}

			} else {
				// info on the Pennsieve DOI was not returned by Discover. This shouldn't really happen, but who knows.
				// Maybe in future we fall back to doi.org?
				datasetDTO, err = dto.NewTombstoneDataset(dto.Tombstone{
					Status: "UNKNOWN",
					DOI:    doi,
				})
				if err != nil {
					return dto.GetCollectionResponse{}, apierrors.NewInternalServerError(
						fmt.Sprintf("error marshalling Discover Tombstone for missing dataset %s", doi), err)
				}
			}
			response.Datasets = append(response.Datasets, datasetDTO)

		}
	}
	response.DerivedContributors = mergedContributors.Deduplicated()
	return response, nil
}
