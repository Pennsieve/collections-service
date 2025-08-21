package routes

import (
	"context"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/publishing"
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/pennsieve/collections-service/internal/test/apitest/builders/stores/collectionstest"
	"github.com/pennsieve/collections-service/internal/test/fixtures"
	"github.com/pennsieve/collections-service/internal/test/mocks"
	"github.com/pennsieve/collections-service/internal/test/userstest"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestUnpublishCollection(t *testing.T) {
	tests := []struct {
		scenario string
		tstFunc  func(t *testing.T, expectationDB *fixtures.ExpectationDB)
	}{
		{"unpublishing a never published collection should fail", testUnpublishNeverPublished},
		{"unpublish collection with publish status", testUnpublish},
		{"should clean up publish status if Discover unpublish fails", testCleanupOnDiscoverError},
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

func testUnpublishNeverPublished(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	callingUser := userstest.NewTestUser(
		userstest.WithFirstName(uuid.NewString()),
		userstest.WithLastName(uuid.NewString()),
		userstest.WithORCID(uuid.NewString()),
		userstest.WithMiddleInitial("F"),
		userstest.WithDegree("B.S."),
	)
	expectationDB.CreateTestUser(ctx, t, callingUser)

	claims := apitest.DefaultClaims(callingUser)

	// The dataset that will be in the collection
	expectedDatasets := apitest.NewExpectedPennsieveDatasets()
	dataset := expectedDatasets.NewPublished()

	// The collection
	expectedCollection := apitest.NewExpectedCollection().WithRandomID().WithNodeID().WithUser(*callingUser.ID, pgdb.Owner).WithPublicDatasets(dataset)
	expectationDB.CreateCollection(ctx, t, expectedCollection)

	pennsieveConfig := apitest.PennsieveConfigWithOptions()

	apiConfig := apitest.NewConfigBuilder().
		WithPennsieveConfig(pennsieveConfig).
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		Build()

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(UnpublishCollectionRouteKey).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
			WithClaims(claims).
			Build(),
		Container: apitest.NewTestContainer().
			WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
			WithCollectionsStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase),
		Config: apiConfig,
		Claims: &claims,
	}

	_, err := UnpublishCollection(ctx, params)

	require.Error(t, err)

	var apiError *apierrors.Error
	require.ErrorAs(t, err, &apiError)
	assert.Equal(t, http.StatusConflict, apiError.StatusCode)
	expectationDB.RequireNoPublishStatus(ctx, t, *expectedCollection.ID)
}

func testUnpublish(t *testing.T, expectationDB *fixtures.ExpectationDB) {
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
		{"completed publication should be allowed", publishing.PublicationType, publishing.CompletedStatus, startedAt, &finishedAt, true},
		{"failed publication should be allowed", publishing.PublicationType, publishing.FailedStatus, startedAt, &finishedAt, true},
		{"in progress revision should not be allowed", publishing.RevisionType, publishing.InProgressStatus, startedAt, nil, false},
		{"completed revision should be allowed", publishing.RevisionType, publishing.CompletedStatus, startedAt, &finishedAt, true},
		{"failed revision should be allowed", publishing.RevisionType, publishing.FailedStatus, startedAt, &finishedAt, true},
		{"in progress removal should not be allowed", publishing.RemovalType, publishing.InProgressStatus, startedAt, nil, false},
		{"completed removal should not be allowed", publishing.RemovalType, publishing.CompletedStatus, startedAt, &finishedAt, false},
		{"failed removal should be allowed", publishing.RemovalType, publishing.FailedStatus, startedAt, &finishedAt, true},
	}

	for _, tt := range tests {
		t.Run(tt.scenario, func(t *testing.T) {

			callingUser := userstest.NewTestUser(
				userstest.WithFirstName(uuid.NewString()),
				userstest.WithLastName(uuid.NewString()),
				userstest.WithORCID(uuid.NewString()),
				userstest.WithMiddleInitial("F"),
				userstest.WithDegree("B.S."),
			)
			expectationDB.CreateTestUser(ctx, t, callingUser)

			claims := apitest.DefaultClaims(callingUser)

			// The dataset that will be in the collection
			expectedDatasets := apitest.NewExpectedPennsieveDatasets()
			dataset := expectedDatasets.NewPublished()

			// The collection
			expectedCollection := apitest.NewExpectedCollection().WithRandomID().WithNodeID().WithUser(*callingUser.ID, pgdb.Owner).WithPublicDatasets(dataset)
			expectationDB.CreateCollection(ctx, t, expectedCollection)

			existingPublishStatus := collectionstest.NewPublishStatusBuilder(*expectedCollection.ID, tt.pubType, tt.pubStatus).
				WithUserID(callingUser.ID).
				WithStartedAt(tt.startedAt).
				WithFinishedAt(tt.finishedAt).
				Build()

			expectationDB.CreatePublishStatus(ctx, t, existingPublishStatus)

			pennsieveConfig := apitest.PennsieveConfigWithOptions()

			// not realistic values, but non-zero to test values are being passed through to our response.
			mockDatasetPublishStatusResponse := service.DatasetPublishStatusResponse{
				PublishedDatasetID:    rand.Intn(5000) + 1,
				PublishedVersionCount: rand.Intn(10),
				Status:                dto.Unpublished,
			}

			expectedOrgServiceRole := apitest.ExpectedOrgServiceRole(pennsieveConfig.CollectionsIDSpace.ID)
			expectedDatasetServiceRole := expectedCollection.DatasetServiceRole(role.Owner)

			mockDiscoverMux := mocks.NewDiscoverMux(*pennsieveConfig.JWTSecretKey.Value).
				WithUnpublishCollectionFunc(ctx, t, expectedCollection.UnpublishCollectionFunc(t, mockDatasetPublishStatusResponse), expectedOrgServiceRole, expectedDatasetServiceRole)

			mockDiscoverServer := httptest.NewServer(mockDiscoverMux)
			defer mockDiscoverServer.Close()

			pennsieveConfig.DiscoverServiceURL = mockDiscoverServer.URL

			apiConfig := apitest.NewConfigBuilder().
				WithPennsieveConfig(pennsieveConfig).
				WithPostgresDBConfig(test.PostgresDBConfig(t)).
				Build()

			params := Params{
				Request: apitest.NewAPIGatewayRequestBuilder(UnpublishCollectionRouteKey).
					WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
					WithClaims(claims).
					Build(),
				Container: apitest.NewTestContainer().
					WithHTTPTestInternalDiscover(pennsieveConfig).
					WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
					WithCollectionsStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase),
				Config: apiConfig,
				Claims: &claims,
			}

			resp, err := UnpublishCollection(ctx, params)

			if tt.allowed {
				require.NoError(t, err)

				assert.Equal(t, mockDatasetPublishStatusResponse.PublishedDatasetID, resp.PublishedDatasetID)
				assert.Equal(t, mockDatasetPublishStatusResponse.PublishedVersionCount, resp.PublishedVersion)
				assert.Equal(t, mockDatasetPublishStatusResponse.Status, resp.Status)

				expectedPublishStatus := collectionstest.NewPublishStatusBuilder(
					*expectedCollection.ID,
					publishing.RemovalType,
					publishing.CompletedStatus).
					WithUserID(callingUser.ID).Build()

				expectationDB.RequirePublishStatus(ctx, t, expectedPublishStatus, &existingPublishStatus)

			} else {
				require.Error(t, err)

				var apiError *apierrors.Error
				require.ErrorAs(t, err, &apiError)
				assert.Equal(t, http.StatusConflict, apiError.StatusCode)
				expectationDB.RequirePublishStatus(ctx, t, existingPublishStatus, nil)
			}
		})
	}
}

func testCleanupOnDiscoverError(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	callingUser := userstest.NewTestUser(
		userstest.WithFirstName(uuid.NewString()),
		userstest.WithLastName(uuid.NewString()),
		userstest.WithORCID(uuid.NewString()),
		userstest.WithMiddleInitial("F"),
		userstest.WithDegree("B.S."),
	)
	expectationDB.CreateTestUser(ctx, t, callingUser)

	claims := apitest.DefaultClaims(callingUser)

	// The dataset that will be in the collection
	expectedDatasets := apitest.NewExpectedPennsieveDatasets()
	dataset := expectedDatasets.NewPublished()

	// The collection
	expectedCollection := apitest.NewExpectedCollection().WithRandomID().WithNodeID().WithUser(*callingUser.ID, pgdb.Owner).WithPublicDatasets(dataset)
	expectationDB.CreateCollection(ctx, t, expectedCollection)

	existingPublishStatus := collectionstest.NewCompletedPublishStatus(*expectedCollection.ID, *callingUser.ID)
	expectationDB.CreatePublishStatus(ctx, t, existingPublishStatus)

	pennsieveConfig := apitest.PennsieveConfigWithOptions()

	apiConfig := apitest.NewConfigBuilder().
		WithPennsieveConfig(pennsieveConfig).
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		Build()

	mockInternalDiscover := mocks.NewInternalDiscover().WithUnpublishCollectionFunc(func(_ context.Context, collectionID int64, collectionNodeID string, userRole role.Role) (service.DatasetPublishStatusResponse, error) {
		return service.DatasetPublishStatusResponse{}, service.CollectionNeverPublishedError{
			ID:     collectionID,
			NodeID: collectionNodeID,
		}
	})

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(UnpublishCollectionRouteKey).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
			WithClaims(claims).
			Build(),
		Container: apitest.NewTestContainer().
			WithInternalDiscover(mockInternalDiscover).
			WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
			WithCollectionsStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase),
		Config: apiConfig,
		Claims: &claims,
	}

	_, err := UnpublishCollection(ctx, params)

	require.Error(t, err)

	var apiError *apierrors.Error
	require.ErrorAs(t, err, &apiError)
	assert.Equal(t, http.StatusConflict, apiError.StatusCode)

	expectedPublishStatus := collectionstest.NewPublishStatusBuilder(*expectedCollection.ID, publishing.RemovalType, publishing.FailedStatus).WithUserID(callingUser.ID).Build()
	expectationDB.RequirePublishStatus(ctx, t, expectedPublishStatus, &existingPublishStatus)
}
