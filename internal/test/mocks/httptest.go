package mocks

import (
	"encoding/json"
	"fmt"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/stretchr/testify/require"
	"net/http"
)

type HTTPError struct {
	StatusCode int
	// Body needs to be json marshalable
	Body any
}

func (e HTTPError) Error() string {
	return fmt.Sprintf("mock error: http status code %d", e.StatusCode)
}

// WriteJSONHTTPResponse json marshals responseBody and sends the bytes with writer.Write.
// Any errors cause the test to fail via t.
func WriteJSONHTTPResponse(t require.TestingT, writer http.ResponseWriter, responseBody any) int {
	resBytes, err := json.Marshal(responseBody)
	require.NoError(t, err)
	n, err := writer.Write(resBytes)
	require.NoError(t, err)
	return n
}

func respond(t require.TestingT, writer http.ResponseWriter, mockResponse any, mockErr error) {
	test.Helper(t)
	var httpResponse any
	switch e := mockErr.(type) {
	case nil:
		httpResponse = mockResponse
	case HTTPError:
		writer.WriteHeader(e.StatusCode)
		httpResponse = e.Body
		if httpResponse == nil {
			httpResponse = fmt.Sprintf(`{"error":%q}`, e.Error())
		}
	default:
		writer.WriteHeader(http.StatusInternalServerError)
		httpResponse = fmt.Sprintf(`{"error":%q}`, e.Error())
	}
	WriteJSONHTTPResponse(t, writer, httpResponse)
}
