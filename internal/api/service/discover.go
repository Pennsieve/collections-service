package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/shared/util"
	"io"
	"net/http"
	"net/url"
)

type Discover interface {
	GetDatasetsByDOI(dois []string) (dto.DatasetsByDOIResponse, error)
}

type HTTPDiscover struct {
	host string
}

func NewHTTPDiscover(discoverHost string) *HTTPDiscover {
	return &HTTPDiscover{host: discoverHost}
}

func (d *HTTPDiscover) GetDatasetsByDOI(dois []string) (dto.DatasetsByDOIResponse, error) {
	doiQueryParams := url.Values{}
	for _, doi := range dois {
		doiQueryParams.Add("doi", doi)
	}
	requestURL := fmt.Sprintf("%s/datasets/doi?%s", d.host, doiQueryParams.Encode())
	response, err := InvokePennsieve(http.MethodGet, requestURL, nil)
	if err != nil {
		return dto.DatasetsByDOIResponse{}, err
	}
	defer util.CloseAndWarn(response)

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return dto.DatasetsByDOIResponse{}, fmt.Errorf("error reading response from GET %s: %w", requestURL, err)
	}
	var responseDTO dto.DatasetsByDOIResponse
	if err := json.Unmarshal(body, &responseDTO); err != nil {
		rawResponse := string(body)
		return dto.DatasetsByDOIResponse{}, fmt.Errorf(
			"error unmarshalling response [%s] from GET %s: %w",
			rawResponse,
			requestURL,
			err)
	}
	return responseDTO, nil
}

func InvokePennsieve(method string, url string, structBody any) (*http.Response, error) {
	req, err := newPennsieveRequest(method, url, structBody)
	if err != nil {
		return nil, fmt.Errorf("error creating %s %s request: %w", method, url, err)
	}
	return util.Invoke(req)
}

func newPennsieveRequest(method string, url string, structBody any) (*http.Request, error) {
	body, err := makeJSONBody(structBody)
	if err != nil {
		return nil, fmt.Errorf("error for %s %s request: %w",
			method, url, err)
	}
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error creating %s %s request: %w", method, url, err)
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
