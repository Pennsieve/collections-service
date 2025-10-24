package routes

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/config"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/pennsieve/collections-service/internal/test/fixtures"
	"github.com/pennsieve/collections-service/internal/test/userstest"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
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

	mockDOIService := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t,
			fmt.Sprintf("/organizations/%d/datasets/%d/doi", apitest.CollectionsIDSpaceID, *collection.ID),
			request.RequestURI,
		)
		assert.Equal(t, http.MethodGet, request.Method)
		writer.WriteHeader(http.StatusNotFound)
	}))
	defer mockDOIService.Close()

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(apitest.PennsieveConfigWithOptions(config.WithDOIServiceURL(mockDOIService.URL))).
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
	// TODO make this test
}
