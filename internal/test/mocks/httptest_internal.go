package mocks

import (
	"github.com/pennsieve/collections-service/internal/api/service/jwtdiscover"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/stretchr/testify/require"
	"net/http"
	"strings"
)

type InternalServer struct {
	jwtSecretKey string
}

func RequireExpectedAuthorization(t require.TestingT, collectionIDParam string, expectedOrgServiceRole, expectedDatasetServiceRole, actualOrgRole, actualDatasetRole jwtdiscover.ServiceRole) {
	require.Equal(t, expectedOrgServiceRole, actualOrgRole)
	require.Equal(t, expectedDatasetServiceRole, actualDatasetRole)
	require.Equal(t, collectionIDParam, actualDatasetRole.Id)
	return
}

func (m InternalServer) RequireExpectedAuthorization(t require.TestingT, collectionIDParam string, expectedOrgServiceRole, expectedDatasetServiceRole jwtdiscover.ServiceRole, request *http.Request) (actualOrgRole jwtdiscover.ServiceRole, actualDatasetRole jwtdiscover.ServiceRole) {
	authHeader := request.Header.Get("Authorization")
	require.NotEmpty(t, authHeader)
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	require.False(t, tokenString == authHeader, "auth header value %s does not start with 'Bearer '", authHeader)

	actualOrgRole, actualDatasetRole = m.ParseJWT(t, tokenString)
	RequireExpectedAuthorization(t, collectionIDParam, expectedOrgServiceRole, expectedDatasetServiceRole, actualOrgRole, actualDatasetRole)

	return
}

func (m InternalServer) ParseJWT(t require.TestingT, tokenString string) (orgRole jwtdiscover.ServiceRole, datasetRole jwtdiscover.ServiceRole) {
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
