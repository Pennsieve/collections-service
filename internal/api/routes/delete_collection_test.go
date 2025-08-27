package routes

import (
	"context"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/publishing"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/pennsieve/collections-service/internal/test/apitest/builders/stores/collectionstest"
	"github.com/pennsieve/collections-service/internal/test/fixtures"
	"github.com/pennsieve/collections-service/internal/test/mocks"
	"github.com/pennsieve/collections-service/internal/test/userstest"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
	"time"
)

func TestDeleteCollection(t *testing.T) {
	tests := []struct {
		scenario string
		tstFunc  func(t *testing.T, expectationDB *fixtures.ExpectationDB)
	}{
		{"delete collection, non-existent collection", testDeleteCollectionNonExistent},
		{"delete collection", testDeleteCollection},
		{"delete collection with publish status", testDeleteCollectionWithPublishStatus},
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
		WithCollectionsStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase)

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
		WithCollectionsStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase)

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

func testDeleteCollectionWithPublishStatus(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	startedAt := time.Now().UTC().AddDate(0, -1, 2)
	finishedAt := startedAt.Add(time.Minute)

	tests := []struct {
		scenario   string
		pubType    publishing.Type
		pubStatus  publishing.Status
		startedAt  time.Time
		finishedAt *time.Time
		allowed    bool
	}{
		{"in progress publication should not be allowed", publishing.PublicationType, publishing.InProgressStatus, startedAt, nil, false},
		{"completed publication should not be allowed", publishing.PublicationType, publishing.CompletedStatus, startedAt, &finishedAt, false},
		{"failed publication should not be allowed", publishing.PublicationType, publishing.FailedStatus, startedAt, &finishedAt, false},
		{"in progress revision should not be allowed", publishing.RevisionType, publishing.InProgressStatus, startedAt, nil, false},
		{"completed revision should not be allowed", publishing.RevisionType, publishing.CompletedStatus, startedAt, &finishedAt, false},
		{"failed revision should not be allowed", publishing.RevisionType, publishing.FailedStatus, startedAt, &finishedAt, false},
		{"in progress removal should not be allowed", publishing.RemovalType, publishing.InProgressStatus, startedAt, nil, false},
		{"completed removal should be allowed", publishing.RemovalType, publishing.CompletedStatus, startedAt, &finishedAt, true},
		{"failed removal should not be allowed", publishing.RemovalType, publishing.FailedStatus, startedAt, &finishedAt, false},
	}

	for _, tt := range tests {
		t.Run(tt.scenario, func(t *testing.T) {
			user := userstest.NewTestUser()
			expectationDB.CreateTestUser(ctx, t, user)

			collection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI())
			createResp := expectationDB.CreateCollection(ctx, t, collection)
			idToDelete := createResp.ID

			existingPublishStatus := collectionstest.NewPublishStatusBuilder(idToDelete, tt.pubType, tt.pubStatus).
				WithUserID(user.ID).
				WithStartedAt(tt.startedAt).
				WithFinishedAt(tt.finishedAt).
				Build()
			expectationDB.CreatePublishStatus(ctx, t, existingPublishStatus)

			claims := apitest.DefaultClaims(user)

			apiConfig := apitest.NewConfigBuilder().
				WithPostgresDBConfig(test.PostgresDBConfig(t)).
				Build()

			container := apitest.NewTestContainer().
				WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
				WithCollectionsStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase)

			params := Params{
				Request: apitest.NewAPIGatewayRequestBuilder(DeleteCollectionRouteKey).
					WithClaims(claims).
					WithPathParam(NodeIDPathParamKey, *collection.NodeID).
					Build(),
				Container: container,
				Config:    apiConfig,
				Claims:    &claims,
			}

			_, err := DeleteCollection(ctx, params)

			if tt.allowed {
				require.NoError(t, err)
				expectationDB.RequireNoCollection(ctx, t, idToDelete)
			} else {
				require.Error(t, err)
				var apiErr *apierrors.Error
				require.ErrorAs(t, err, &apiErr)

				assert.Equal(t, http.StatusConflict, apiErr.StatusCode)

				expectationDB.RequireCollection(ctx, t, collection, idToDelete)
			}
		})
	}

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
				WithGetCollectionFunc(expectedCollection.GetCollectionFunc(t, nil))

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
				WithGetCollectionFunc(expectedCollection.GetCollectionFunc(t, nil)).
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
