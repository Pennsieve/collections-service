package routes

import (
	"context"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/pennsieve/collections-service/internal/test/fixtures"
	"github.com/pennsieve/collections-service/internal/test/mocks"
	"github.com/pennsieve/collections-service/internal/test/userstest"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestDeleteCollection(t *testing.T) {
	tests := []struct {
		scenario string
		tstFunc  func(t *testing.T, expectationDB *fixtures.ExpectationDB)
	}{
		{"delete collection, non-existent collection", testDeleteCollectionNonExistent},
		{"delete collection", testDeleteCollection},
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

func testDeleteCollectionNonExistent(t *testing.T, _ *fixtures.ExpectationDB) {
	callingUser := userstest.SeedUser1
	nonExistentNodeID := uuid.NewString()

	claims := apitest.DefaultClaims(callingUser)

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
		WithContainerStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(DeleteCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, nonExistentNodeID).
			Build(),
		Container: container,
		Config:    apiConfig,
		Claims:    &claims,
	}

	_, err := DeleteCollection(context.Background(), params)

	var apiErr *apierrors.Error
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
	assert.Contains(t, apiErr.UserMessage, nonExistentNodeID)

}

func testDeleteCollection(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	user1 := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user1)
	user2 := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user2)

	user1CollectionDelete := apitest.NewExpectedCollection().WithNodeID().WithUser(*user1.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI())
	createResp := expectationDB.CreateCollection(ctx, t, user1CollectionDelete)
	idToDelete := createResp.ID

	user1CollectionKeep := apitest.NewExpectedCollection().WithNodeID().WithUser(*user1.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI())
	keepResp := expectationDB.CreateCollection(ctx, t, user1CollectionKeep)

	user2Collection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user2.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI())
	user2Resp := expectationDB.CreateCollection(ctx, t, user2Collection)

	claims := apitest.DefaultClaims(user1)

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
		WithContainerStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(DeleteCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *user1CollectionDelete.NodeID).
			Build(),
		Container: container,
		Config:    apiConfig,
		Claims:    &claims,
	}

	_, err := DeleteCollection(ctx, params)

	require.NoError(t, err)

	expectationDB.RequireNoCollection(ctx, t, idToDelete)
	expectationDB.RequireCollection(ctx, t, user1CollectionKeep, keepResp.ID)
	expectationDB.RequireCollection(ctx, t, user2Collection, user2Resp.ID)
}

// TestHandleDeleteCollection tests that run the Handle wrapper around DeleteCollection
func TestHandleDeleteCollection(t *testing.T) {
	tests := []struct {
		scenario string
		tstFunc  func(t *testing.T)
	}{
		{"delete collection, authorization", testDeleteAuthz},
	}

	for _, tt := range tests {
		t.Run(tt.scenario, func(t *testing.T) {
			tt.tstFunc(t)
		})
	}
}

func testDeleteAuthz(t *testing.T) {
	ctx := context.Background()
	callingUser := userstest.SeedUser1
	claims := apitest.DefaultClaims(callingUser)

	// pgdb.Write & pgdb.Delete => role.Editor, which we take to mean perm it add or delete DOIs but
	// not delete the collection itself
	for _, tooLowPerm := range []pgdb.DbPermission{pgdb.Guest, pgdb.Read, pgdb.Write, pgdb.Delete} {
		t.Run(tooLowPerm.String(), func(t *testing.T) {
			expectedCollection := apitest.NewExpectedCollection().WithRandomID().WithNodeID().WithUser(callingUser.ID, tooLowPerm)

			mockCollectionStore := mocks.NewCollectionsStore().
				WithGetCollectionFunc(expectedCollection.GetCollectionFunc(t))

			params := Params{
				Request: apitest.NewAPIGatewayRequestBuilder(DeleteCollectionRouteKey).
					WithClaims(claims).
					WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
					Build(),
				Container: apitest.NewTestContainer().WithCollectionsStore(mockCollectionStore),
				Config:    apitest.NewConfigBuilder().WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).Build(),
				Claims:    &claims,
			}

			resp, err := Handle(ctx, NewDeleteCollectionRouteHandler(), params)
			require.NoError(t, err)

			assert.Equal(t, http.StatusForbidden, resp.StatusCode)
			assert.Equal(t, DefaultErrorResponseHeaders(), resp.Headers)
			assert.Contains(t, resp.Body, "errorId")
			assert.Contains(t, resp.Body, "message")
		})
	}

	for _, okPerm := range []pgdb.DbPermission{pgdb.Administer, pgdb.Owner} {
		t.Run(okPerm.String(), func(t *testing.T) {
			expectedCollection := apitest.NewExpectedCollection().WithRandomID().WithNodeID().WithUser(callingUser.ID, okPerm)

			// we're not saving this to a real DB, so no ID is generated for us
			mockCollectionID := int64(123)
			expectedCollection.ID = &mockCollectionID

			mockCollectionStore := mocks.NewCollectionsStore().
				WithGetCollectionFunc(expectedCollection.GetCollectionFunc(t)).
				WithDeleteCollectionFunc(func(ctx context.Context, collectionID int64) error {
					require.Equal(t, mockCollectionID, collectionID)
					return nil
				})

			params := Params{
				Request: apitest.NewAPIGatewayRequestBuilder(DeleteCollectionRouteKey).
					WithClaims(claims).
					WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
					Build(),
				Container: apitest.NewTestContainer().WithCollectionsStore(mockCollectionStore),
				Config:    apitest.NewConfigBuilder().WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).Build(),
				Claims:    &claims,
			}

			resp, err := Handle(ctx, NewDeleteCollectionRouteHandler(), params)
			require.NoError(t, err)

			assert.Equal(t, http.StatusNoContent, resp.StatusCode)
			assert.Empty(t, resp.Body)
			assert.Empty(t, resp.Headers)
		})
	}

}
