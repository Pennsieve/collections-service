package mocks

import (
	"encoding/json"
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/collections-service/internal/api/service/jwtdiscover"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
	"github.com/stretchr/testify/require"
	"net/http"
	"strconv"
	"strings"
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

func (m *DiscoverMux) WithGetDatasetsByDOIFunc(t require.TestingT, f GetDatasetsByDOIFunc) *DiscoverMux {
	m.HandleFunc("GET /datasets/doi", func(writer http.ResponseWriter, request *http.Request) {
		test.Helper(t)
		query := request.URL.Query()
		require.Contains(t, query, "doi")
		dois := query["doi"]
		res, err := f(dois)
		require.NoError(t, err)
		resBytes, err := json.Marshal(res)
		require.NoError(t, err)
		_, err = writer.Write(resBytes)
		require.NoError(t, err)
	})
	return m
}

func (m *DiscoverMux) WithPublishCollectionFunc(t require.TestingT, f PublishCollectionFunc, expectedOrgServiceRole, expectedDatasetServiceRole jwtdiscover.ServiceRole) *DiscoverMux {
	m.HandleFunc("POST /collection/{collectionId}/publish", func(writer http.ResponseWriter, request *http.Request) {
		test.Helper(t)

		collectionIDParam := request.PathValue("collectionId")
		collectionID, err := strconv.ParseInt(collectionIDParam, 10, 64)
		require.NoError(t, err)

		var publishRequest service.PublishDOICollectionRequest
		require.NoError(t, json.NewDecoder(request.Body).Decode(&publishRequest))

		authHeader := request.Header.Get("Authorization")
		require.NotEmpty(t, authHeader)
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		require.False(t, tokenString == authHeader, "auth header value %s does not start with 'Bearer '", authHeader)

		orgRole, datasetRole := m.ParseJWT(t, tokenString)

		require.Equal(t, expectedOrgServiceRole, orgRole)
		require.Equal(t, expectedDatasetServiceRole, datasetRole)
		require.Equal(t, collectionIDParam, datasetRole.Id)

		datasetRoleRole, _ := role.RoleFromString(datasetRole.Role)

		publishResponse, err := f(collectionID, datasetRoleRole, publishRequest)
		require.NoError(t, err)

		resBytes, err := json.Marshal(publishResponse)
		require.NoError(t, err)
		_, err = writer.Write(resBytes)
		require.NoError(t, err)
	})
	return m
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
