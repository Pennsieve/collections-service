package routes

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/config"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/pennsieve/collections-service/internal/test/fixtures"
	"github.com/pennsieve/collections-service/internal/test/mocks"
	"github.com/pennsieve/collections-service/internal/test/userstest"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetDOI(t *testing.T) {

	tests := []struct {
		scenario string
		tstFunc  func(t *testing.T, expectationDB *fixtures.ExpectationDB)
	}{
		{"get doi, no collection", testGetDOINoCollection},
		{"get doi, no DOI", testGetDOINoDOI},
		{"get doi", testGetDOI},
	}

	ctx := context.Background()
	postgresDBConfig := test.PostgresDBConfig(t)

	for _, tt := range tests {
		t.Run(tt.scenario, func(t *testing.T) {
			db := test.NewPostgresDBFromConfig(t, postgresDBConfig)
			expectationDB := fixtures.NewExpectationDB(db, postgresDBConfig.CollectionsDatabase)

			t.Cleanup(func() {
				expectationDB.CleanUp(ctx, t)
			})

			tt.tstFunc(t, expectationDB)
		})
	}
}

func testGetDOINoCollection(t *testing.T, db *fixtures.ExpectationDB) {
	ctx := context.Background()

	// use a user with no collections
	callingUser := userstest.SeedUser1
	nonExistentNodeID := uuid.NewString()

	claims := apitest.DefaultClaims(callingUser)

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
		WithCollectionsStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(GetDOIRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, nonExistentNodeID).
			Build(),
		Container: container,
		Config:    apiConfig,
		Claims:    &claims,
	}
	_, err := GetDOI(ctx, params)

	var notFoundErr *apierrors.Error
	require.ErrorAs(t, err, &notFoundErr)

	assert.Equal(t, http.StatusNotFound, notFoundErr.StatusCode)
	assert.Equal(t, fmt.Sprintf("collection %s not found", nonExistentNodeID), notFoundErr.UserMessage)
}

func testGetDOINoDOI(t *testing.T, db *fixtures.ExpectationDB) {
	ctx := context.Background()

	user := userstest.NewTestUser()
	db.CreateTestUser(ctx, t, user)

	claims := apitest.DefaultClaims(user)

	collection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user.ID, pgdb.Owner)
	db.CreateCollection(ctx, t, collection)

	jwtSecretKey := uuid.NewString()
	doiMux := mocks.NewDOIMux(jwtSecretKey).WithGetLatestDOIFunc(ctx, t,
		func(ctx context.Context, collectionID int64, collectionNodeID string, userRole role.Role) (dto.GetLatestDOIResponse, error) {
			assert.Equal(t, *collection.ID, collectionID)
			assert.Equal(t, *collection.NodeID, collectionNodeID)
			return dto.GetLatestDOIResponse{}, mocks.HTTPError{StatusCode: http.StatusNotFound}
		},
		apitest.ExpectedOrgServiceRole(apitest.CollectionsIDSpaceID),
		collection.DatasetServiceRoleForUser(t, user),
	)
	mockDOIService := httptest.NewServer(doiMux)
	defer mockDOIService.Close()

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(apitest.PennsieveConfigWithOptions(
			config.WithDOIServiceURL(mockDOIService.URL),
			config.WithJWTSecretKey(jwtSecretKey),
		)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
		WithCollectionsStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase).
		WithHTTPTestDOI(apiConfig.PennsieveConfig)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(GetDOIRouteKey).
			WithPathParam(NodeIDPathParamKey, *collection.NodeID).
			WithClaims(claims).
			Build(),
		Container: container,
		Config:    apiConfig,
		Claims:    &claims,
	}
	_, err := GetDOI(ctx, params)

	var notFoundErr *apierrors.Error
	require.ErrorAs(t, err, &notFoundErr)

	assert.Equal(t, http.StatusNotFound, notFoundErr.StatusCode)
	assert.Equal(t, fmt.Sprintf("DOI for collection %s not found", *collection.NodeID), notFoundErr.UserMessage)
}

func testGetDOI(t *testing.T, db *fixtures.ExpectationDB) {
	ctx := context.Background()

	user := userstest.NewTestUser()
	db.CreateTestUser(ctx, t, user)

	claims := apitest.DefaultClaims(user)

	collection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user.ID, pgdb.Owner)
	db.CreateCollection(ctx, t, collection)

	doiResponse := dto.GetLatestDOIResponse{
		OrganizationID:  apitest.CollectionsIDSpaceID,
		DatasetID:       *collection.ID,
		DOI:             apitest.NewPennsieveDOI().Value,
		Title:           uuid.NewString(),
		URL:             uuid.NewString(),
		Publisher:       uuid.NewString(),
		CreatedAt:       uuid.NewString(),
		PublicationYear: 2024,
		State:           uuid.NewString(),
		Creators:        []string{uuid.NewString(), uuid.NewString()},
	}

	jwtSecretKey := uuid.NewString()
	doiMux := mocks.NewDOIMux(jwtSecretKey).WithGetLatestDOIFunc(ctx, t,
		collection.GetLatestDOIFunc(t, &doiResponse),
		apitest.ExpectedOrgServiceRole(apitest.CollectionsIDSpaceID),
		collection.DatasetServiceRoleForUser(t, user),
	)

	mockDOIService := httptest.NewServer(doiMux)
	defer mockDOIService.Close()

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(apitest.PennsieveConfigWithOptions(
			config.WithDOIServiceURL(mockDOIService.URL),
			config.WithJWTSecretKey(jwtSecretKey),
		)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
		WithCollectionsStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase).
		WithHTTPTestDOI(apiConfig.PennsieveConfig)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(GetDOIRouteKey).
			WithPathParam(NodeIDPathParamKey, *collection.NodeID).
			WithClaims(claims).
			Build(),
		Container: container,
		Config:    apiConfig,
		Claims:    &claims,
	}
	response, err := GetDOI(ctx, params)
	require.NoError(t, err)

	assert.Equal(t, doiResponse, response)
}
