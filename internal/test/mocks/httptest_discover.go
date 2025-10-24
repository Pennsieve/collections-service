package mocks

import (
	"context"
	"encoding/json"
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/collections-service/internal/api/service/jwtdiscover"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
	"github.com/stretchr/testify/require"
	"net/http"
	"strconv"
)

func ToDiscoverHandlerFunc(ctx context.Context, t require.TestingT, f GetDatasetsByDOIFunc) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		test.Helper(t)
		require.Equal(t, http.MethodGet, request.Method)
		url := request.URL
		require.Equal(t, "/datasets/doi", url.Path)
		query := url.Query()
		require.Contains(t, query, "doi")
		dois := query["doi"]
		res, err := f(ctx, dois)
		require.NoError(t, err)
		WriteJSONHTTPResponse(t, writer, res)
	}
}

// DiscoverMux holds the mocked handlers for Discover and InternalDiscover. Even though the dependency container separates
// Discover and InternalDiscover, we allow both to be served by a single httptest.Server instance using
// this DiscoverMux
type DiscoverMux struct {
	*http.ServeMux
	InternalServer
}

func NewDiscoverMux(jwtSecretKey string) *DiscoverMux {
	return &DiscoverMux{
		http.NewServeMux(),
		InternalServer{jwtSecretKey},
	}
}

func (m *DiscoverMux) WithGetDatasetsByDOIFunc(ctx context.Context, t require.TestingT, f GetDatasetsByDOIFunc) *DiscoverMux {
	m.HandleFunc("GET /datasets/doi", func(writer http.ResponseWriter, request *http.Request) {
		test.Helper(t)
		query := request.URL.Query()
		require.Contains(t, query, "doi")
		dois := query["doi"]
		res, err := f(ctx, dois)
		respond(t, writer, res, err)
	})
	return m
}

func (m *DiscoverMux) WithPublishCollectionFunc(ctx context.Context, t require.TestingT, f PublishCollectionFunc, expectedOrgServiceRole, expectedDatasetServiceRole jwtdiscover.ServiceRole) *DiscoverMux {
	m.HandleFunc("POST /collection/{collectionId}/publish", func(writer http.ResponseWriter, request *http.Request) {
		test.Helper(t)

		collectionIDParam := request.PathValue("collectionId")
		collectionID, err := strconv.ParseInt(collectionIDParam, 10, 64)
		require.NoError(t, err)

		var publishRequest service.PublishDOICollectionRequest
		require.NoError(t, json.NewDecoder(request.Body).Decode(&publishRequest))

		_, actualDatasetRole := m.RequireExpectedAuthorization(t, collectionIDParam, expectedOrgServiceRole, expectedDatasetServiceRole, request)
		datasetRoleRole, _ := role.RoleFromString(actualDatasetRole.Role)

		publishResponse, err := f(ctx, collectionID, datasetRoleRole, publishRequest)
		respond(t, writer, publishResponse, err)
	})
	return m
}

func (m *DiscoverMux) WithFinalizeCollectionPublishFunc(ctx context.Context, t require.TestingT, f FinalizeCollectionPublishFunc, expectedCollectionNodeID string, expectedOrgServiceRole, expectedDatasetServiceRole jwtdiscover.ServiceRole) *DiscoverMux {
	m.HandleFunc("POST /collection/{collectionId}/finalize", func(writer http.ResponseWriter, request *http.Request) {
		test.Helper(t)

		collectionIDParam := request.PathValue("collectionId")
		collectionID, err := strconv.ParseInt(collectionIDParam, 10, 64)
		require.NoError(t, err)

		var finalizeRequest service.FinalizeDOICollectionPublishRequest
		require.NoError(t, json.NewDecoder(request.Body).Decode(&finalizeRequest))

		_, actualDatasetRole := m.RequireExpectedAuthorization(t, collectionIDParam, expectedOrgServiceRole, expectedDatasetServiceRole, request)
		require.Equal(t, expectedCollectionNodeID, actualDatasetRole.NodeId)

		datasetRoleRole, _ := role.RoleFromString(actualDatasetRole.Role)
		finalizeResponse, err := f(ctx, collectionID, expectedCollectionNodeID, datasetRoleRole, finalizeRequest)
		respond(t, writer, finalizeResponse, err)
	})
	return m
}

func (m *DiscoverMux) WithGetCollectionPublishStatusFunc(ctx context.Context, t require.TestingT, f GetCollectionPublishStatusFunc, expectedCollectionNodeID string, expectedOrgServiceRole, expectedDatasetServiceRole jwtdiscover.ServiceRole) *DiscoverMux {
	m.HandleFunc("GET /organizations/{organizationId}/datasets/{datasetId}", func(writer http.ResponseWriter, request *http.Request) {
		test.Helper(t)
		collectionIDParam := request.PathValue("datasetId")
		collectionID, err := strconv.ParseInt(collectionIDParam, 10, 64)
		require.NoError(t, err)

		require.Equal(t, expectedOrgServiceRole.Id, request.PathValue("organizationId"))

		_, actualDatasetRole := m.RequireExpectedAuthorization(t, collectionIDParam, expectedOrgServiceRole, expectedDatasetServiceRole, request)
		require.Equal(t, expectedCollectionNodeID, actualDatasetRole.NodeId)

		datasetRoleRole, _ := role.RoleFromString(actualDatasetRole.Role)
		publishStatusResponse, err := f(ctx, collectionID, expectedCollectionNodeID, datasetRoleRole)
		respond(t, writer, publishStatusResponse, err)
	})
	return m
}

func (m *DiscoverMux) WithUnpublishCollectionFunc(ctx context.Context, t require.TestingT, f UnpublishCollectionFunc, expectedOrgServiceRole, expectedDatasetServiceRole jwtdiscover.ServiceRole) *DiscoverMux {
	m.HandleFunc("POST /collection/{collectionId}/unpublish", func(writer http.ResponseWriter, request *http.Request) {
		test.Helper(t)

		collectionIDParam := request.PathValue("collectionId")
		collectionID, err := strconv.ParseInt(collectionIDParam, 10, 64)
		require.NoError(t, err)

		_, actualDatasetRole := m.RequireExpectedAuthorization(t, collectionIDParam, expectedOrgServiceRole, expectedDatasetServiceRole, request)
		datasetRoleRole, _ := role.RoleFromString(actualDatasetRole.Role)

		unpublishResponse, err := f(ctx, collectionID, actualDatasetRole.NodeId, datasetRoleRole)
		respond(t, writer, unpublishResponse, err)
	})
	return m
}
