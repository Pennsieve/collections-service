package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"net/url"
	"sync"

	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/shared/util"
)

type Discover interface {
	GetDatasetsByDOI(ctx context.Context, dois []string) (DatasetsByDOIResponse, error)
}

type HTTPDiscover struct {
	url    string
	logger *slog.Logger
}

func NewHTTPDiscover(discoverURL string, logger *slog.Logger) *HTTPDiscover {
	return &HTTPDiscover{url: discoverURL, logger: logger}
}

// maxRequestURILength: hard cap Discover enforces on the request URI
const maxRequestURILength = 2048

// uriLengthSafetyMargin: headroom reserved below maxRequestURILength to
// absorb minor variations (longer DOIs, slight URL changes, etc.). 100 chars
// is roughly 5 DOIs of headroom.
const uriLengthSafetyMargin = 100

// numDOIBatchWorkers: the number of concurrent workers
const numDOIBatchWorkers = 3

const datasetsByDOIPath = "/datasets/doi"

func (d *HTTPDiscover) GetDatasetsByDOI(ctx context.Context, dois []string) (DatasetsByDOIResponse, error) {

	urlPrefix := fmt.Sprintf("%s%s?", d.url, datasetsByDOIPath)
	queryBudget := maxRequestURILength - len(urlPrefix) - uriLengthSafetyMargin
	batches := batchDOIsByQueryLength(dois, queryBudget)
	if len(batches) == 0 {
		return DatasetsByDOIResponse{}, nil
	}
	if len(batches) == 1 {
		return d.fetchDatasetsByDOI(ctx, batches[0])
	}
	return d.fetchDatasetsByDOIConcurrent(ctx, batches, numDOIBatchWorkers)
}

// fetchDatasetsByDOIConcurrent fans batches out to a pool of workers and merges
// the responses. Fast-fails on the first error via context cancellation.
func (d *HTTPDiscover) fetchDatasetsByDOIConcurrent(ctx context.Context, batches [][]string, numWorkers int) (DatasetsByDOIResponse, error) {
	type batchResult struct {
		resp DatasetsByDOIResponse
		err  error
	}

	if numWorkers > len(batches) {
		numWorkers = len(batches)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	jobs := make(chan []string)
	results := make(chan batchResult, len(batches))

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
				resp, err := d.fetchDatasetsByDOI(ctx, batch)
				results <- batchResult{resp: resp, err: err}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, batch := range batches {
			select {
			case <-ctx.Done():
				return
			case jobs <- batch:
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	var merged DatasetsByDOIResponse
	for res := range results {
		if res.err != nil {
			cancel()
			return DatasetsByDOIResponse{}, res.err
		}
		if len(res.resp.Published) > 0 {
			if merged.Published == nil {
				merged.Published = map[string]dto.PublicDataset{}
			}
			maps.Copy(merged.Published, res.resp.Published)
		}
		if len(res.resp.Unpublished) > 0 {
			if merged.Unpublished == nil {
				merged.Unpublished = map[string]dto.Tombstone{}
			}
			maps.Copy(merged.Unpublished, res.resp.Unpublished)
		}
	}
	return merged, nil
}

func (d *HTTPDiscover) fetchDatasetsByDOI(ctx context.Context, dois []string) (DatasetsByDOIResponse, error) {
	doiQueryParams := url.Values{}
	for _, doi := range dois {
		doiQueryParams.Add("doi", doi)
	}
	requestParams := requestParameters{
		method: http.MethodGet,
		url:    fmt.Sprintf("%s%s?%s", d.url, datasetsByDOIPath, doiQueryParams.Encode()),
	}
	response, err := d.InvokePennsieve(ctx, requestParams)
	if err != nil {
		return DatasetsByDOIResponse{}, err
	}
	defer util.CloseAndWarn(response, d.logger)

	var responseDTO DatasetsByDOIResponse
	if err := util.UnmarshallResponse(response, &responseDTO); err != nil {
		return DatasetsByDOIResponse{}, fmt.Errorf(
			"error unmarshalling response to %s: %w",
			requestParams,
			err)
	}
	return responseDTO, nil
}

// batchDOIsByQueryLength splits dois into batches whose URL-encoded
// "doi=...&doi=..." query strings fit within maxQueryLen characters
func batchDOIsByQueryLength(dois []string, maxQueryLen int) [][]string {
	var batches [][]string
	var current []string
	currentLen := 0
	for _, doi := range dois {
		paramLen := len("doi=") + len(url.QueryEscape(doi))
		if len(current) > 0 {
			paramLen += len("&")
		}
		if len(current) > 0 && currentLen+paramLen > maxQueryLen {
			batches = append(batches, current)
			current = nil
			currentLen = 0
			paramLen = len("doi=") + len(url.QueryEscape(doi))
		}
		current = append(current, doi)
		currentLen += paramLen
	}
	if len(current) > 0 {
		batches = append(batches, current)
	}
	return batches
}

func (d *HTTPDiscover) InvokePennsieve(ctx context.Context, requestParams requestParameters) (*http.Response, error) {
	req, err := newPennsieveRequest(ctx, requestParams)
	if err != nil {
		return nil, fmt.Errorf("error creating %s request: %w", requestParams, err)
	}
	return util.Invoke(req, d.logger)
}

type requestParameters struct {
	method string
	url    string
	body   any
}

func (p requestParameters) String() string {
	return fmt.Sprintf("%s %s", p.method, p.url)
}

func newPennsieveRequest(ctx context.Context, requestParams requestParameters) (*http.Request, error) {
	body, err := makeJSONBody(requestParams.body)
	if err != nil {
		return nil, fmt.Errorf("error for %s request: %w",
			requestParams, err)
	}
	request, err := http.NewRequestWithContext(ctx, requestParams.method, requestParams.url, body)
	if err != nil {
		return nil, fmt.Errorf("error creating %s request: %w", requestParams, err)
	}
	request.Header.Add("accept", util.ApplicationJSON)
	request.Header.Add("Content-Type", util.ApplicationJSON)
	return request, nil
}

func makeJSONBody(structBody any) (io.Reader, error) {
	if structBody == nil {
		return nil, nil
	}
	var buffer bytes.Buffer
	if err := json.NewEncoder(&buffer).Encode(structBody); err != nil {
		return nil, fmt.Errorf("error encoding body: %w", err)
	}
	return &buffer, nil
}

type DatasetsByDOIResponse struct {
	Published   map[string]dto.PublicDataset `json:"published"`
	Unpublished map[string]dto.Tombstone     `json:"unpublished"`
}
