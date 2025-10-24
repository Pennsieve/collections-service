package service_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/collections-service/internal/shared/logging"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPDOI_GetLatestDOI_OK(t *testing.T) {

	collectionID := int64(6)
	collectionNodeID := uuid.NewString()

	doiResponse := dto.GetLatestDOIResponse{
		OrganizationID:  apitest.CollectionsIDSpaceID,
		DatasetID:       collectionID,
		DOI:             apitest.NewPennsieveDOI().Value,
		Title:           uuid.NewString(),
		URL:             uuid.NewString(),
		Publisher:       uuid.NewString(),
		CreatedAt:       uuid.NewString(),
		PublicationYear: 2024,
		State:           uuid.NewString(),
		Creators:        []string{uuid.NewString(), uuid.NewString()},
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t,
			fmt.Sprintf("/organizations/%d/datasets/%d/doi", apitest.CollectionsIDSpaceID, collectionID),
			request.RequestURI,
		)
		assert.Equal(t, http.MethodGet, request.Method)
		writer.WriteHeader(http.StatusOK)
		respBytes, err := json.Marshal(doiResponse)
		require.NoError(t, err)
		_, err = writer.Write(respBytes)
		require.NoError(t, err)
	}))
	defer mockServer.Close()

	doiService := service.NewHTTPDOI(mockServer.URL, uuid.NewString(), apitest.CollectionsIDSpaceID, logging.Default)

	response, err := doiService.GetLatestDOI(context.Background(), collectionID, collectionNodeID, role.Owner)
	require.NoError(t, err)

	assert.Equal(t, doiResponse, response)

}

func TestHTTPDOI_GetLatestDOI_NotFound(t *testing.T) {

	collectionID := int64(6)
	collectionNodeID := uuid.NewString()

	mockServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t,
			fmt.Sprintf("/organizations/%d/datasets/%d/doi", apitest.CollectionsIDSpaceID, collectionID),
			request.RequestURI,
		)
		assert.Equal(t, http.MethodGet, request.Method)
		writer.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	doiService := service.NewHTTPDOI(mockServer.URL, uuid.NewString(), apitest.CollectionsIDSpaceID, logging.Default)

	_, err := doiService.GetLatestDOI(context.Background(), collectionID, collectionNodeID, role.Owner)

	var notFoundErr service.LatestDOINotFoundError
	require.ErrorAs(t, err, &notFoundErr)
	assert.Equal(t, collectionID, notFoundErr.ID)
	assert.Equal(t, collectionNodeID, notFoundErr.NodeID)
}
