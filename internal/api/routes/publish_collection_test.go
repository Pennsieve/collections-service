package routes

import (
	"context"
	"github.com/google/uuid"
	config2 "github.com/pennsieve/collections-service/internal/api/config"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/publishing"
	"github.com/pennsieve/collections-service/internal/api/store/collections"
	"github.com/pennsieve/collections-service/internal/api/store/manifests"
	"github.com/pennsieve/collections-service/internal/api/store/users"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/pennsieve/collections-service/internal/test/fixtures"
	"github.com/pennsieve/collections-service/internal/test/mocks"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublishCollection(t *testing.T) {
	tests := []struct {
		scenario string
		tstFunc  func(t *testing.T, expectationDB *fixtures.ExpectationDB, minio *fixtures.MinIO)
	}{
		{"publish collection", testPublish},
	}

	ctx := context.Background()
	postgresDBConfig := test.PostgresDBConfig(t)

	for _, tt := range tests {
		t.Run(tt.scenario, func(t *testing.T) {
			db := test.NewPostgresDBFromConfig(t, postgresDBConfig)
			expectationDB := fixtures.NewExpectationDB(db, postgresDBConfig.CollectionsDatabase)
			minio := fixtures.NewMinIOWithDefaultClient(ctx, t)

			t.Cleanup(func() {
				expectationDB.CleanUp(ctx, t)
				minio.CleanUp(ctx, t)
			})

			tt.tstFunc(t, expectationDB, minio)
		})
	}

}

func testPublish(t *testing.T, expectationDB *fixtures.ExpectationDB, minio *fixtures.MinIO) {
	ctx := context.Background()

	publishBucket := minio.CreatePublishBucket(ctx, t)

	callingUser := apitest.NewTestUser(
		apitest.WithFirstName(uuid.NewString()),
		apitest.WithLastName(uuid.NewString()),
		apitest.WithORCID(uuid.NewString()),
	)
	expectationDB.CreateTestUser(ctx, t, callingUser)

	claims := apitest.DefaultClaims(callingUser)

	// The dataset that will be in the collection
	expectedDatasets := apitest.NewExpectedPennsieveDatasets()
	dataset := expectedDatasets.NewPublished()

	// The collection
	expectedCollection := apitest.NewExpectedCollection().WithRandomID().WithNodeID().WithUser(*callingUser.ID, pgdb.Owner).WithPublicDatasets(dataset)
	expectationDB.CreateCollection(ctx, t, expectedCollection)

	pennsieveConfig := apitest.PennsieveConfigWithOptions(config2.WithPublishBucket(publishBucket))

	expectedPublishedDatasetID := rand.Int63n(5000) + 1
	expectedPublishedVersion := rand.Int63n(20) + 1
	expectedPublishStatus := uuid.NewString()

	mockDiscoverMux := mocks.NewDiscoverMux(*pennsieveConfig.JWTSecretKey.Value).
		WithGetDatasetsByDOIFunc(t, expectedDatasets.GetDatasetsByDOIFunc(t)).
		WithPublishCollectionFunc(t, expectedCollection.PublishCollectionFunc(
			t,
			expectedPublishedDatasetID,
			expectedPublishedVersion,
			expectedPublishStatus,
			apitest.VerifyPublishingUser(callingUser),
		),
			apitest.ExpectedOrgServiceRole(pennsieveConfig.CollectionNamespaceID),
			expectedCollection.DatasetServiceRole(role.Owner),
		)

	mockDiscoverServer := httptest.NewServer(mockDiscoverMux)
	defer mockDiscoverServer.Close()

	pennsieveConfig.DiscoverServiceURL = mockDiscoverServer.URL

	config := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(pennsieveConfig).
		Build()

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PublishCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
			WithBody(t, dto.PublishCollectionRequest{
				License: "Creative Commons",
				Tags:    []string{"test"},
			}).
			Build(),
		Container: apitest.NewTestContainer().
			WithPostgresDB(test.NewPostgresDBFromConfig(t, config.PostgresDB)).
			WithContainerStoreFromPostgresDB(config.PostgresDB.CollectionsDatabase).
			WithUsersStoreFromPostgresDB(config.PostgresDB.CollectionsDatabase).
			WithHTTPTestDiscover(mockDiscoverServer.URL).
			WithHTTPTestInternalDiscover(pennsieveConfig).
			WithMinIOManifestStore(ctx, t, config.PennsieveConfig.PublishBucket),
		Config: config,
		Claims: &claims,
	}

	resp, err := PublishCollection(ctx, params)
	require.NoError(t, err)

	assert.Equal(t, expectedPublishedDatasetID, resp.PublishedDatasetID)
	assert.Equal(t, expectedPublishedVersion, resp.PublishedVersion)
	assert.Equal(t, expectedPublishStatus, resp.Status)

	minio.RequireObjectExists(ctx, t, pennsieveConfig.PublishBucket, publishing.S3Key(resp.PublishedDatasetID))

}

// TestHandlePublishCollection tests that run the Handle wrapper around PublishCollection
func TestHandlePublishCollection(t *testing.T) {
	tests := []struct {
		name    string
		tstFunc func(t *testing.T)
	}{
		{"return Bad Request when given no body", testHandlePublishCollectionNoBody},
		{
			"return Bad Request when given empty license",
			testHandlePublishCollectionEmptyLicense,
		},
		{
			"return Bad Request when given a license that is too long",
			testHandlePublishCollectionLicenseTooLong,
		},
		{
			"return Bad Request when given no tags",
			testHandlePublishCollectionNoTags,
		},
		{
			"return Bad Request when given empty tags",
			testHandlePublishCollectionEmptyTags,
		},
		{
			"return Not Found when given a non-existent collection",
			testHandlePublishCollectionNotFound,
		},
		{
			"forbid publish from users without the proper role on the collection",
			testHandlePublishCollectionAuthz,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.tstFunc(t)
		})
	}
}

func testHandlePublishCollectionNoBody(t *testing.T) {
	ctx := context.Background()
	callingUser := apitest.SeedUser1

	mockCollectionStore := mocks.NewCollectionsStore()

	claims := apitest.DefaultClaims(callingUser)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PublishCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, uuid.NewString()).
			Build(),
		Container: apitest.NewTestContainer().WithCollectionsStore(mockCollectionStore),
		Config:    apitest.NewConfigBuilder().WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).Build(),
		Claims:    &claims,
	}
	response, err := Handle(ctx, NewPublishCollectionRouteHandler(), params)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, response.StatusCode)

	assert.Contains(t, response.Body, "missing request body")

}

func testHandlePublishCollectionEmptyLicense(t *testing.T) {
	ctx := context.Background()
	callingUser := apitest.SeedUser1

	publishRequest := dto.PublishCollectionRequest{
		License: "",
		Tags:    []string{"test"},
	}

	mockCollectionStore := mocks.NewCollectionsStore()

	claims := apitest.DefaultClaims(callingUser)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PublishCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, uuid.NewString()).
			WithBody(t, publishRequest).
			Build(),
		Container: apitest.NewTestContainer().WithCollectionsStore(mockCollectionStore),
		Config:    apitest.NewConfigBuilder().WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).Build(),
		Claims:    &claims,
	}
	response, err := Handle(ctx, NewPublishCollectionRouteHandler(), params)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, response.StatusCode)

	assert.Contains(t, response.Body, "license cannot be empty")
}

func testHandlePublishCollectionLicenseTooLong(t *testing.T) {
	ctx := context.Background()
	callingUser := apitest.SeedUser1

	publishRequest := dto.PublishCollectionRequest{
		License: strings.Repeat("a", 256),
		Tags:    []string{"test"},
	}

	mockCollectionStore := mocks.NewCollectionsStore()

	claims := apitest.DefaultClaims(callingUser)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PublishCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, uuid.NewString()).
			WithBody(t, publishRequest).
			Build(),
		Container: apitest.NewTestContainer().WithCollectionsStore(mockCollectionStore),
		Config:    apitest.NewConfigBuilder().WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).Build(),
		Claims:    &claims,
	}
	response, err := Handle(ctx, NewPublishCollectionRouteHandler(), params)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, response.StatusCode)

	assert.Contains(t, response.Body, "license cannot have more than 255 characters")

}

func testHandlePublishCollectionNoTags(t *testing.T) {
	ctx := context.Background()
	callingUser := apitest.SeedUser1

	publishRequest := dto.PublishCollectionRequest{
		License: "Creative Commons something",
	}

	mockCollectionStore := mocks.NewCollectionsStore()

	claims := apitest.DefaultClaims(callingUser)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PublishCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, uuid.NewString()).
			WithBody(t, publishRequest).
			Build(),
		Container: apitest.NewTestContainer().WithCollectionsStore(mockCollectionStore),
		Config:    apitest.NewConfigBuilder().WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).Build(),
		Claims:    &claims,
	}
	response, err := Handle(ctx, NewPublishCollectionRouteHandler(), params)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, response.StatusCode)

	assert.Contains(t, response.Body, "tags array cannot be empty")

}

func testHandlePublishCollectionEmptyTags(t *testing.T) {
	ctx := context.Background()
	callingUser := apitest.SeedUser1

	publishRequest := dto.PublishCollectionRequest{
		License: "Creative Commons something",
		Tags:    []string{},
	}

	mockCollectionStore := mocks.NewCollectionsStore()

	claims := apitest.DefaultClaims(callingUser)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PublishCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, uuid.NewString()).
			WithBody(t, publishRequest).
			Build(),
		Container: apitest.NewTestContainer().WithCollectionsStore(mockCollectionStore),
		Config:    apitest.NewConfigBuilder().WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).Build(),
		Claims:    &claims,
	}
	response, err := Handle(ctx, NewPublishCollectionRouteHandler(), params)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, response.StatusCode)

	assert.Contains(t, response.Body, "tags array cannot be empty")

}

func testHandlePublishCollectionNotFound(t *testing.T) {
	ctx := context.Background()
	callingUser := apitest.SeedUser1
	nonExistentNodeID := uuid.NewString()

	mockCollectionStore := mocks.NewCollectionsStore().WithGetCollectionFunc(func(ctx context.Context, userID int64, nodeID string) (collections.GetCollectionResponse, error) {
		test.Helper(t)
		require.Equal(t, callingUser.ID, userID)
		require.Equal(t, nonExistentNodeID, nodeID)
		return collections.GetCollectionResponse{}, collections.ErrCollectionNotFound
	})

	claims := apitest.DefaultClaims(callingUser)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PublishCollectionRouteKey).
			WithClaims(claims).
			WithBody(t, dto.PublishCollectionRequest{
				License: "Creative Commons",
				Tags:    []string{"test"},
			}).
			WithPathParam(NodeIDPathParamKey, nonExistentNodeID).
			Build(),
		Container: apitest.NewTestContainer().WithCollectionsStore(mockCollectionStore),
		Config:    apitest.NewConfigBuilder().WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).Build(),
		Claims:    &claims,
	}
	response, err := Handle(ctx, NewPublishCollectionRouteHandler(), params)
	require.NoError(t, err)

	assert.Equal(t, http.StatusNotFound, response.StatusCode)

	assert.Contains(t, response.Body, "not found")
	assert.Contains(t, response.Body, nonExistentNodeID)

}

func testHandlePublishCollectionAuthz(t *testing.T) {
	ctx := context.Background()
	callingUser := apitest.SeedUser1
	claims := apitest.DefaultClaims(callingUser)

	for _, tooLowPerm := range []pgdb.DbPermission{pgdb.Guest, pgdb.Read, pgdb.Write, pgdb.Delete, pgdb.Administer} {
		t.Run(tooLowPerm.String(), func(t *testing.T) {
			expectedCollection := apitest.NewExpectedCollection().WithRandomID().WithNodeID().WithUser(callingUser.ID, tooLowPerm)

			mockCollectionStore := mocks.NewCollectionsStore().
				WithGetCollectionFunc(expectedCollection.GetCollectionFunc(t))

			params := Params{
				Request: apitest.NewAPIGatewayRequestBuilder(PublishCollectionRouteKey).
					WithClaims(claims).
					WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
					WithBody(t, dto.PublishCollectionRequest{
						License: "Creative Commons",
						Tags:    []string{"test"},
					}).
					Build(),
				Container: apitest.NewTestContainer().WithCollectionsStore(mockCollectionStore),
				Config:    apitest.NewConfigBuilder().WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).Build(),
				Claims:    &claims,
			}

			resp, err := Handle(ctx, NewPublishCollectionRouteHandler(), params)
			require.NoError(t, err)

			assert.Equal(t, http.StatusForbidden, resp.StatusCode)
			assert.Equal(t, DefaultErrorResponseHeaders(), resp.Headers)
			assert.Contains(t, resp.Body, "errorId")
			assert.Contains(t, resp.Body, "message")
		})
	}

	// only pgdb.Owner can publish
	for _, okPerm := range []pgdb.DbPermission{pgdb.Owner} {
		t.Run(okPerm.String(), func(t *testing.T) {

			expectedDatasets := apitest.NewExpectedPennsieveDatasets()
			dataset := expectedDatasets.NewPublished()

			mockDiscover := mocks.NewDiscover().WithGetDatasetsByDOIFunc(expectedDatasets.GetDatasetsByDOIFunc(t))

			expectedCollection := apitest.NewExpectedCollection().WithRandomID().WithNodeID().WithUser(callingUser.ID, okPerm).WithPublicDatasets(dataset)

			mockCollectionStore := mocks.NewCollectionsStore().
				WithGetCollectionFunc(expectedCollection.GetCollectionFunc(t))

			expectedPublishedID := int64(14)
			mockInternalDiscover := mocks.NewInternalDiscover().WithPublishCollectionFunc(
				expectedCollection.PublishCollectionFunc(t,
					expectedPublishedID,
					1,
					"PublishInProgress",
					apitest.VerifyPublishingUser(callingUser)),
			)

			mockUsersStore := mocks.NewUsersStore().WithGetUserFunc(func(ctx context.Context, userID int64) (users.GetUserResponse, error) {
				t.Helper()
				require.Equal(t, callingUser.ID, userID)
				return users.GetUserResponse{
					FirstName: &callingUser.FirstName,
					LastName:  &callingUser.LastName,
				}, nil
			})

			pennsieveConfig := apitest.PennsieveConfigWithFakeURL()

			mockManifestStore := mocks.NewManifestStore().WithSaveManifestFunc(func(ctx context.Context, key string, manifest publishing.ManifestV5) (manifests.SaveManifestResponse, error) {
				require.Equal(t, publishing.S3Key(expectedPublishedID), key)
				require.Equal(t, expectedPublishedID, manifest.PennsieveDatasetId)
				require.Equal(t, expectedCollection.Name, manifest.Name)
				require.Equal(t, callingUser.LastName, manifest.Creator.LastName)
				return manifests.SaveManifestResponse{
					S3VersionID: uuid.NewString(),
				}, nil
			})

			params := Params{
				Request: apitest.NewAPIGatewayRequestBuilder(PublishCollectionRouteKey).
					WithClaims(claims).
					WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
					WithBody(t, dto.PublishCollectionRequest{
						License: "Creative Commons",
						Tags:    []string{"test"},
					}).
					Build(),
				Container: apitest.NewTestContainer().
					WithCollectionsStore(mockCollectionStore).
					WithDiscover(mockDiscover).
					WithInternalDiscover(mockInternalDiscover).
					WithUsersStore(mockUsersStore).
					WithManifestStore(mockManifestStore),
				Config: apitest.NewConfigBuilder().WithPennsieveConfig(pennsieveConfig).Build(),
				Claims: &claims,
			}

			resp, err := Handle(ctx, NewPublishCollectionRouteHandler(), params)
			require.NoError(t, err)

			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})
	}

}
