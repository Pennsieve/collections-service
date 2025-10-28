package routes

import (
	"context"
	"fmt"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/container"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/collections-service/internal/shared/util"
	"log/slog"
	"net/http"
	"sync"
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
		batchSize  = 80 // how many DOIs per request. >= 90 leads to URL-too-long errors. See discover_benchmark_test.go
		numWorkers = 3  // how many concurrent requests
	)

	discoverService := container.Discover()
	// In reality, len(pennsieveDOIs) <= 40 == FE page size * 4 banner DOIs per collection
	// Testing in discover_benchmark_test.go showed not much point in doing concurrent batches in this case.
	if len(pennsieveDOIs) <= batchSize {
		discoverResp, err := discoverService.GetDatasetsByDOI(ctx, pennsieveDOIs)
		if err != nil {
			return nil, fmt.Errorf("error looking up Datasets in Discover by DOI: %w", err)
		}
		return discoverResp.Published, nil
	}

	// But we do get URL-to-long errors if we request 90 or more DOIs at a time. So
	// to keep things working if someone scripts calls with larger page sizes, we'll
	// batch things here.
	return batchLookupPennsieveDatasets(ctx, discoverService, pennsieveDOIs, batchSize, numWorkers)
}

// batchLookupPennsieveDatasets fetches datasets by DOI in concurrent batches.
// Safe for arbitrary input sizes and avoids URL length limits.
// Typical use: up to ~80 DOIs per batch, 3 workers.
func batchLookupPennsieveDatasets(
	ctx context.Context,
	discover service.Discover,
	dois []string,
	batchSize int,
	numWorkers int,
) (map[string]dto.PublicDataset, error) {

	type batchResult struct {
		data map[string]dto.PublicDataset
		err  error
	}

	jobs := make(chan []string)
	results := make(chan batchResult)

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for w := 0; w < numWorkers; w++ {
		go func() {
			defer wg.Done()
			for batch := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
				}

				resp, err := discover.GetDatasetsByDOI(ctx, batch)
				if err != nil {
					results <- batchResult{err: fmt.Errorf("fetch batch failed: %w", err)}
					continue
				}
				results <- batchResult{data: resp.Published}
			}
		}()
	}

	go func() {
		for i := 0; i < len(dois); i += batchSize {
			end := i + batchSize
			if end > len(dois) {
				end = len(dois)
			}
			jobs <- dois[i:end]
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	doiToDataset := make(map[string]dto.PublicDataset)
	for res := range results {
		if res.err != nil {
			return nil, fmt.Errorf("error fetching datasets from Discover by DOI: %w", res.err)
		}
		for k, v := range res.data {
			doiToDataset[k] = v
		}
	}

	return doiToDataset, nil
}
