package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/shared/util"
	"io"
	"log/slog"
	"net/http"
	"net/url"
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

func (d *HTTPDiscover) GetDatasetsByDOI(ctx context.Context, dois []string) (DatasetsByDOIResponse, error) {
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
