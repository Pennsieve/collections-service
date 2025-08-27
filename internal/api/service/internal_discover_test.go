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

// TestUnpublishCollectionNoContent tests the case where Discover /collection/{id}/unpublish returns No Content
// because no published collection is found.
func TestUnpublishCollectionNoContent(t *testing.T) {
	collectionID := int64(5)
	collectionNodeID := uuid.NewString()

	mockServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, fmt.Sprintf("/collection/%d/unpublish", collectionID), request.RequestURI)
		assert.Equal(t, http.MethodPost, request.Method)
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer mockServer.Close()

	discover := service.NewHTTPInternalDiscover(mockServer.URL, uuid.NewString(), apitest.CollectionsIDSpaceID, logging.Default)

	_, err := discover.UnpublishCollection(context.Background(), collectionID, collectionNodeID, role.Owner)
	var neverPublishedErr service.CollectionNeverPublishedError

	require.Error(t, err)
	require.Error(t, err)
	require.ErrorAs(t, err, &neverPublishedErr)
	assert.Equal(t, collectionID, neverPublishedErr.ID)
	assert.Equal(t, collectionNodeID, neverPublishedErr.NodeID)
}

// TestUnpublishCollectionContent tests the case where Discover /collection/{id}/unpublish returns OK
// because a published collection is found and unpublished
func TestUnpublishCollectionContent(t *testing.T) {
	collectionNodeID := uuid.NewString()

	discoverResp := service.DatasetPublishStatusResponse{
		Name:                  uuid.NewString(),
		SourceOrganizationID:  int(apitest.CollectionsIDSpaceID),
		SourceDatasetID:       5,
		PublishedDatasetID:    301,
		PublishedVersionCount: 1,
		Status:                dto.Unpublished,
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, fmt.Sprintf("/collection/%d/unpublish", discoverResp.SourceDatasetID), request.RequestURI)
		assert.Equal(t, http.MethodPost, request.Method)
		writer.WriteHeader(http.StatusOK)
		respBytes, err := json.Marshal(discoverResp)
		require.NoError(t, err)
		_, err = writer.Write(respBytes)
		require.NoError(t, err)
	}))
	defer mockServer.Close()

	discover := service.NewHTTPInternalDiscover(mockServer.URL, uuid.NewString(), apitest.CollectionsIDSpaceID, logging.Default)

	resp, err := discover.UnpublishCollection(context.Background(), int64(discoverResp.SourceDatasetID), collectionNodeID, role.Owner)

	require.NoError(t, err)

	assert.Equal(t, discoverResp, resp)
}
