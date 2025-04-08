package mocks

import (
	"encoding/json"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/stretchr/testify/require"
	"net/http"
)

func ToDiscoverHandlerFunc(t require.TestingT, f GetDatasetsByDOIFunc) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		test.Helper(t)
		require.Equal(t, http.MethodGet, request.Method)
		url := request.URL
		require.Equal(t, "/datasets/doi", url.Path)
		query := url.Query()
		require.Contains(t, query, "doi")
		dois := query["doi"]
		res, err := f(dois)
		require.NoError(t, err)
		resBytes, err := json.Marshal(res)
		require.NoError(t, err)
		_, err = writer.Write(resBytes)
		require.NoError(t, err)
	}
}
