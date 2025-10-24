package service_test

import (
	"context"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/collections-service/internal/api/service/jwtdiscover"
	"github.com/pennsieve/collections-service/internal/shared/logging"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/pennsieve/collections-service/internal/test/mocks"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPDOI_GetLatestDOI_OK(t *testing.T) {
	ctx := context.Background()

	expectedCollectionID := int64(6)
	expectedCollectionNodeID := uuid.NewString()
	expectedRole := role.Owner

	doiResponse := dto.GetLatestDOIResponse{
		OrganizationID:  apitest.CollectionsIDSpaceID,
		DatasetID:       expectedCollectionID,
		DOI:             apitest.NewPennsieveDOI().Value,
		Title:           uuid.NewString(),
		URL:             uuid.NewString(),
		Publisher:       uuid.NewString(),
		CreatedAt:       uuid.NewString(),
		PublicationYear: 2024,
		State:           uuid.NewString(),
		Creators:        []string{uuid.NewString(), uuid.NewString()},
	}

	jwtSecreteKey := uuid.NewString()
	doiMux := mocks.NewDOIMux(jwtSecreteKey).WithGetLatestDOIFunc(ctx, t,
		func(ctx context.Context, collectionID int64, collectionNodeID string, userRole role.Role) (dto.GetLatestDOIResponse, error) {
			assert.Equal(t, expectedCollectionID, collectionID)
			assert.Equal(t, expectedCollectionNodeID, collectionNodeID)
			assert.Equal(t, expectedRole, userRole)
			return doiResponse, nil
		},
		apitest.ExpectedOrgServiceRole(apitest.CollectionsIDSpaceID),
		jwtdiscover.NewDatasetServiceRole(expectedCollectionID, expectedCollectionNodeID, expectedRole))

	mockServer := httptest.NewServer(doiMux)
	defer mockServer.Close()

	doiService := service.NewHTTPDOI(mockServer.URL, jwtSecreteKey, apitest.CollectionsIDSpaceID, logging.Default)

	response, err := doiService.GetLatestDOI(context.Background(), expectedCollectionID, expectedCollectionNodeID, expectedRole)
	require.NoError(t, err)

	assert.Equal(t, doiResponse, response)

}

func TestHTTPDOI_GetLatestDOI_NotFound(t *testing.T) {

	ctx := context.Background()

	expectedCollectionID := int64(6)
	expectedCollectionNodeID := uuid.NewString()
	expectedRole := role.Owner

	jwtSecretKey := uuid.NewString()
	doiMux := mocks.NewDOIMux(jwtSecretKey).WithGetLatestDOIFunc(ctx, t,
		func(ctx context.Context, collectionID int64, collectionNodeID string, userRole role.Role) (dto.GetLatestDOIResponse, error) {
			return dto.GetLatestDOIResponse{}, mocks.HTTPError{
				StatusCode: http.StatusNotFound,
			}
		},
		apitest.ExpectedOrgServiceRole(apitest.CollectionsIDSpaceID),
		jwtdiscover.NewDatasetServiceRole(expectedCollectionID, expectedCollectionNodeID, expectedRole),
	)

	mockServer := httptest.NewServer(doiMux)
	defer mockServer.Close()

	doiService := service.NewHTTPDOI(mockServer.URL, jwtSecretKey, apitest.CollectionsIDSpaceID, logging.Default)

	_, err := doiService.GetLatestDOI(context.Background(), expectedCollectionID, expectedCollectionNodeID, expectedRole)

	var notFoundErr service.LatestDOINotFoundError
	require.ErrorAs(t, err, &notFoundErr)
	assert.Equal(t, expectedCollectionID, notFoundErr.ID)
	assert.Equal(t, expectedCollectionNodeID, notFoundErr.NodeID)
}
