package mocks

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/collections-service/internal/api/service/jwtdiscover"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
	"github.com/stretchr/testify/require"
	"net/http"
	"strconv"
	"strings"
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
		resBytes, err := json.Marshal(res)
		require.NoError(t, err)
		_, err = writer.Write(resBytes)
		require.NoError(t, err)
	}
}

// DiscoverMux holds the mocked handlers for Discover. Even though the dependency container separates
// Discover and InternalDiscover, we allow both to be served by a single httptest.Server instance using
// this DiscoverMux
type DiscoverMux struct {
	*http.ServeMux
	jwtSecretKey string
}

func NewDiscoverMux(jwtSecretKey string) *DiscoverMux {
	return &DiscoverMux{
		http.NewServeMux(),
		jwtSecretKey,
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

		collectionIDParam, collectionID := parseID(t, request, "collectionId")

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

		collectionIDParam, collectionID := parseID(t, request, "collectionId")

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
		collectionIDParam, collectionID := parseID(t, request, "datasetId")

		require.Equal(t, expectedOrgServiceRole.Id, request.PathValue("organizationId"))

		_, actualDatasetRole := m.RequireExpectedAuthorization(t, collectionIDParam, expectedOrgServiceRole, expectedDatasetServiceRole, request)
		require.Equal(t, expectedCollectionNodeID, actualDatasetRole.NodeId)

		datasetRoleRole, _ := role.RoleFromString(actualDatasetRole.Role)
		publishStatusResponse, err := f(ctx, collectionID, expectedCollectionNodeID, datasetRoleRole)
		respond(t, writer, publishStatusResponse, err)
	})
	return m
}

func parseID(t require.TestingT, request *http.Request, idKey string) (idParam string, idValue int32) {
	idParam = request.PathValue(idKey)
	id64, err := strconv.ParseInt(idParam, 10, 32)
	require.NoError(t, err)
	idValue = int32(id64)
	return
}

func respond(t require.TestingT, writer http.ResponseWriter, mockResponse any, mockErr error) {
	test.Helper(t)
	var httpResponse any
	switch e := mockErr.(type) {
	case nil:
		httpResponse = mockResponse
	case *apierrors.Error:
		writer.WriteHeader(e.StatusCode)
		httpResponse = e
	default:
		writer.WriteHeader(http.StatusInternalServerError)
		httpResponse = fmt.Sprintf(`{"error":%q}`, e.Error())
	}
	resBytes, err := json.Marshal(httpResponse)
	require.NoError(t, err)
	_, err = writer.Write(resBytes)
	require.NoError(t, err)
}

func (m *DiscoverMux) RequireExpectedAuthorization(t require.TestingT, collectionIDParam string, expectedOrgServiceRole, expectedDatasetServiceRole jwtdiscover.ServiceRole, request *http.Request) (actualOrgRole jwtdiscover.ServiceRole, actualDatasetRole jwtdiscover.ServiceRole) {
	authHeader := request.Header.Get("Authorization")
	require.NotEmpty(t, authHeader)
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	require.False(t, tokenString == authHeader, "auth header value %s does not start with 'Bearer '", authHeader)

	actualOrgRole, actualDatasetRole = m.ParseJWT(t, tokenString)

	require.Equal(t, expectedOrgServiceRole, actualOrgRole)
	require.Equal(t, expectedDatasetServiceRole, actualDatasetRole)
	require.Equal(t, collectionIDParam, actualDatasetRole.Id)
	return
}

func (m *DiscoverMux) ParseJWT(t require.TestingT, tokenString string) (orgRole jwtdiscover.ServiceRole, datasetRole jwtdiscover.ServiceRole) {
	test.Helper(t)

	serviceClaim, err := jwtdiscover.ParseServiceClaim(tokenString, m.jwtSecretKey)
	require.NoError(t, err)

	serviceRoles := serviceClaim.Roles
	require.Len(t, serviceRoles, 2)
	for _, serviceRole := range serviceRoles {
		switch serviceRole.Type {
		case jwtdiscover.OrganizationServiceRoleType:
			orgRole = serviceRole
		case jwtdiscover.DatasetServiceRoleType:
			datasetRole = serviceRole
		default:
			require.FailNow(t, "unexpected role type in service claim", serviceRole.Type)
		}
	}

	return
}
