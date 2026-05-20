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

// maxDOIQueryLength is the maximum length of the query string portion of a
// GetDatasetsByDOI request. Kept comfortably below the default 2048-character
// URI limit
const maxDOIQueryLength = 1800

func (d *HTTPDiscover) GetDatasetsByDOI(ctx context.Context, dois []string) (DatasetsByDOIResponse, error) {
	batches := batchDOIsByQueryLength(dois, maxDOIQueryLength)
	if len(batches) == 0 {
		return DatasetsByDOIResponse{}, nil
	}
	if len(batches) == 1 {
		return d.fetchDatasetsByDOI(ctx, batches[0])
	}
	var merged DatasetsByDOIResponse
	for _, batch := range batches {
		batchResponse, err := d.fetchDatasetsByDOI(ctx, batch)
		// Return fast if any batch request fails
		if err != nil {
			return DatasetsByDOIResponse{}, err
		}
		if len(batchResponse.Published) > 0 {
			if merged.Published == nil {
				merged.Published = map[string]dto.PublicDataset{}
			}
			maps.Copy(merged.Published, batchResponse.Published)
		}
		if len(batchResponse.Unpublished) > 0 {
			if merged.Unpublished == nil {
				merged.Unpublished = map[string]dto.Tombstone{}
			}
			maps.Copy(merged.Unpublished, batchResponse.Unpublished)
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
		url:    fmt.Sprintf("%s/datasets/doi?%s", d.url, doiQueryParams.Encode()),
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
