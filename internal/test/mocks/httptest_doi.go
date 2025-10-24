package mocks

import (
	"context"
	"github.com/pennsieve/collections-service/internal/api/service/jwtdiscover"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"strconv"
)

type DOIMux struct {
	*http.ServeMux
	InternalServer
}

func NewDOIMux(jwtSecretKey string) *DOIMux {
	return &DOIMux{
		ServeMux:       http.NewServeMux(),
		InternalServer: InternalServer{jwtSecretKey: jwtSecretKey},
	}
}

func (m *DOIMux) WithGetLatestDOIFunc(ctx context.Context, t require.TestingT, f GetLatestDOIFunc, expectedOrgServiceRole, expectedDatasetServiceRole jwtdiscover.ServiceRole) *DOIMux {
	m.HandleFunc("GET /organizations/{organizationId}/datasets/{datasetId}/doi", func(writer http.ResponseWriter, request *http.Request) {
		test.Helper(t)

		orgIDParam := request.PathValue("organizationId")
		assert.Equal(t, expectedOrgServiceRole.Id, orgIDParam)

		collectionIDParam := request.PathValue("datasetId")
		collectionID, err := strconv.ParseInt(collectionIDParam, 10, 64)
		require.NoError(t, err)

		_, actualDatasetRole := m.RequireExpectedAuthorization(t, collectionIDParam, expectedOrgServiceRole, expectedDatasetServiceRole, request)

		collectionNodeID := expectedDatasetServiceRole.NodeId

		datasetRoleRole, _ := role.RoleFromString(actualDatasetRole.Role)
		finalizeResponse, err := f(ctx, collectionID, collectionNodeID, datasetRoleRole)
		respond(t, writer, finalizeResponse, err)
	})
	return m
}
