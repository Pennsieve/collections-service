package routes

import (
	"context"
	"errors"
	"fmt"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/container"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/shared/util"
	"log/slog"
	"net/http"
	"sync"
	"time"
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
	params.Container.AddLoggingContext(
		slog.String("userNodeId", userClaim.NodeId))

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

	// For now we are assuming only PennsieveDOIs will be present in collections
	var doiToPublicDataset map[string]dto.PublicDataset
	pennsieveDOIs, _ := CategorizeDOIs(params.Config.PennsieveConfig.DOIPrefix, dois)
	if len(pennsieveDOIs) > 0 {
		doiToPublicDataset, err = lookupPennsieveDatasets(ctx, params.Container, pennsieveDOIs)
		if err != nil {
			return dto.GetCollectionsResponse{}, apierrors.NewInternalServerError(fmt.Sprintf("error looking up DOIs in Discover for user %s", userClaim.NodeId), err)
		}
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
			Publication: ToDTOPublication(storeCollection.Publication, nil),
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

func lookupPennsieveDatasets(ctx context.Context, container container.DependencyContainer, pennsieveDOIs []string) (map[string]dto.PublicDataset, error) {
	const (
		batchSize  = 10 // how many DOIs per request
		numWorkers = 5  // how many concurrent requests
	)

	discoverService := container.Discover()
	doiToPublicDataset := make(map[string]dto.PublicDataset)
	var errs []error

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	jobs := make(chan []string)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errsMu sync.Mutex

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for batch := range jobs {
				discoverResp, err := discoverService.GetDatasetsByDOI(ctx, batch)
				if err != nil {
					errsMu.Lock()
					errs = append(errs, fmt.Errorf("batch %v: %w", batch, err))
					errsMu.Unlock()
					continue // keep going even if one batch fails
				}

				mu.Lock()
				for doi, ds := range discoverResp.Published {
					doiToPublicDataset[doi] = ds
				}
				mu.Unlock()
			}
		}()
	}

	go func() {
		defer close(jobs)
		for i := 0; i < len(pennsieveDOIs); i += batchSize {
			end := i + batchSize
			if end > len(pennsieveDOIs) {
				end = len(pennsieveDOIs)
			}
			select {
			case jobs <- pennsieveDOIs[i:end]:
			case <-ctx.Done():
				return
			}
		}
	}()

	start := time.Now()
	wg.Wait()
	elapsed := time.Since(start)

	if len(errs) > 0 {
		combinedErr := errors.Join(errs...)
		return nil, combinedErr
	}
	container.Logger().Info("looked up DOIs in Discover",
		slog.Int("totalDOIs", len(pennsieveDOIs)),
		slog.Int("batchSize", batchSize),
		slog.Int("numWorkers", numWorkers),
		slog.String("totalTime", elapsed.Truncate(time.Millisecond).String()),
		slog.Float64("DOIs/sec", float64(len(pennsieveDOIs))/elapsed.Seconds()),
	)
	return doiToPublicDataset, nil
}
