package routes

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/apijson"
	"github.com/pennsieve/collections-service/internal/api/config"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/publishing"
	"github.com/pennsieve/collections-service/internal/api/store/collections"
	"github.com/pennsieve/collections-service/internal/api/store/manifests"
	"github.com/pennsieve/collections-service/internal/api/store/users"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/apitest"
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
	"strings"
	"testing"
	"time"
)

func TestPublishCollection(t *testing.T) {
	tests := []struct {
		scenario string
		tstFunc  func(t *testing.T, expectationDB *fixtures.ExpectationDB, minio *fixtures.MinIO)
	}{
		{"publish collection", testPublish},
		{"should return a 409 Conflict error if publish already in progress", testPublishNoConcurrent},
		{"should return Bad Request if description is empty", testPublishNoDescription},
		{"should return Bad Request if collection contains unpublished datasets", testPublishContainsTombstones},
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
	createCollectionResp := expectationDB.CreateCollection(ctx, t, expectedCollection)

	expectedPublishStatus := apitest.NewExpectedPublishStatus(
		publishing.CompletedStatus,
		publishing.PublicationType,
		*callingUser.ID).WithCollectionID(createCollectionResp.ID)

	pennsieveConfig := apitest.PennsieveConfigWithOptions(config.WithPublishBucket(publishBucket))

	expectedPublishedDatasetID := rand.Int63n(5000) + 1
	expectedPublishedVersion := rand.Int63n(20) + 1
	expectedDiscoverPublishStatus := uuid.NewString()
	expectedDiscoverFinalizeStatus := uuid.NewString()

	expectedOrgServiceRole := apitest.ExpectedOrgServiceRole(pennsieveConfig.CollectionNamespaceID)
	expectedDatasetServiceRole := expectedCollection.DatasetServiceRole(role.Owner)
	mockDiscoverMux := mocks.NewDiscoverMux(*pennsieveConfig.JWTSecretKey.Value).
		WithGetDatasetsByDOIFunc(ctx, t, expectedDatasets.GetDatasetsByDOIFunc(t)).
		WithPublishCollectionFunc(ctx, t, expectedCollection.PublishCollectionFunc(
			t,
			expectedPublishedDatasetID,
			expectedPublishedVersion,
			expectedDiscoverPublishStatus,
			apitest.VerifyPublishingUser(callingUser),
		),
			expectedOrgServiceRole,
			expectedDatasetServiceRole,
		).
		WithFinalizeCollectionPublishFunc(ctx, t, expectedCollection.FinalizeCollectionPublishFunc(
			t,
			expectedPublishedDatasetID,
			expectedPublishedVersion,
			expectedDiscoverFinalizeStatus),
			expectedOrgServiceRole,
			expectedDatasetServiceRole,
		)

	mockDiscoverServer := httptest.NewServer(mockDiscoverMux)
	defer mockDiscoverServer.Close()

	pennsieveConfig.DiscoverServiceURL = mockDiscoverServer.URL

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(pennsieveConfig).
		Build()

	expectedLicense := "Creative Commons"
	expectedKeywords := []string{"test1, test2"}
	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PublishCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
			WithBody(t, dto.PublishCollectionRequest{
				License: expectedLicense,
				Tags:    expectedKeywords,
			}).
			Build(),
		Container: apitest.NewTestContainer().
			WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
			WithCollectionsStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase).
			WithUsersStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase).
			WithHTTPTestDiscover(mockDiscoverServer.URL).
			WithHTTPTestInternalDiscover(pennsieveConfig).
			WithMinIOManifestStore(ctx, t, apiConfig.PennsieveConfig.PublishBucket),
		Config: apiConfig,
		Claims: &claims,
	}

	resp, err := PublishCollection(ctx, params)
	require.NoError(t, err)

	assert.Equal(t, expectedPublishedDatasetID, resp.PublishedDatasetID)
	assert.Equal(t, expectedPublishedVersion, resp.PublishedVersion)
	assert.Equal(t, expectedDiscoverFinalizeStatus, resp.Status)

	expectationDB.RequirePublishStatus(ctx, t, expectedPublishStatus)

	manifestKey := publishing.S3Key(resp.PublishedDatasetID)
	headManifest := minio.RequireObjectExists(ctx, t, pennsieveConfig.PublishBucket, manifestKey)
	var actualManifest publishing.ManifestV5
	minio.GetObject(ctx, t, pennsieveConfig.PublishBucket, manifestKey, headManifest.VersionId).As(t, &actualManifest)

	assert.Equal(t, expectedPublishedDatasetID, actualManifest.PennsieveDatasetID)
	assert.Equal(t, expectedPublishedVersion, actualManifest.Version)
	assert.Zero(t, actualManifest.Revision)
	assert.Equal(t, expectedCollection.Name, actualManifest.Name)
	assert.Equal(t, expectedCollection.Description, actualManifest.Description)

	expectedCreator := apitest.ToPublishedContributor(callingUser)
	assert.Equal(t, expectedCreator, actualManifest.Creator)
	assert.Len(t, actualManifest.Contributors, 1)
	assert.Equal(t, expectedCreator, actualManifest.Contributors[0])

	assert.Equal(t, expectedKeywords, actualManifest.Keywords)
	assert.Equal(t, expectedLicense, actualManifest.License)

	expectedDatePublished := apijson.Date(time.Now().UTC())
	assert.True(t, expectedDatePublished.Equal(actualManifest.DatePublished),
		"expected date published: %s, actual date published: %s",
		expectedDatePublished, actualManifest.DatePublished)

	assert.NotEmpty(t, actualManifest.ID)

	assert.Equal(t, publishing.ManifestPublisher, actualManifest.Publisher)
	assert.Equal(t, publishing.ManifestContext, actualManifest.Context)
	assert.Equal(t, publishing.ManifestSchemaVersion, actualManifest.SchemaVersion)
	assert.Equal(t, publishing.ManifestType, actualManifest.Type)
	assert.Equal(t, publishing.ManifestPennsieveSchemaVersion, actualManifest.PennsieveSchemaVersion)

	assert.Equal(t, expectedCollection.DOIs.Strings(), actualManifest.References)

	expectedFileManifest := publishing.FileManifest{
		Name:     publishing.ManifestFileName,
		Path:     publishing.ManifestFileName,
		Size:     aws.ToInt64(headManifest.ContentLength),
		FileType: publishing.ManifestFileType,
		// These fields are not set for the manifest's own FileManifest entry
		SourcePackageId: "",
		S3VersionId:     "",
		SHA256:          "",
	}
	require.Len(t, actualManifest.Files, 1)
	assert.Equal(t, expectedFileManifest, actualManifest.Files[0])

}

func testPublishNoConcurrent(t *testing.T, expectationDB *fixtures.ExpectationDB, _ *fixtures.MinIO) {
	ctx := context.Background()

	callingUser := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, callingUser)

	claims := apitest.DefaultClaims(callingUser)

	// The dataset that will be in the collection
	expectedDatasets := apitest.NewExpectedPennsieveDatasets()
	dataset := expectedDatasets.NewPublished()

	// The collection
	expectedCollection := apitest.NewExpectedCollection().WithRandomID().WithNodeID().WithUser(*callingUser.ID, pgdb.Owner).WithPublicDatasets(dataset)
	createCollectionResp := expectationDB.CreateCollection(ctx, t, expectedCollection)

	expectedPublishStatus := apitest.NewExpectedInProgressPublishStatus(*callingUser.ID).
		WithCollectionID(createCollectionResp.ID)
	expectationDB.CreatePublishStatusPreCondition(ctx, t, expectedPublishStatus)

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).
		Build()

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PublishCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
			WithBody(t, dto.PublishCollectionRequest{
				License: "Creative Commons",
				Tags:    []string{"test1, test2"},
			}).
			Build(),
		Container: apitest.NewTestContainer().
			WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
			WithCollectionsStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase),
		Config: apiConfig,
		Claims: &claims,
	}

	_, err := PublishCollection(ctx, params)
	var apiError *apierrors.Error
	require.ErrorAs(t, err, &apiError)

	assert.Equal(t, http.StatusConflict, apiError.StatusCode)
	assert.Contains(t, apiError.UserMessage, "in progress")

	expectationDB.RequirePublishStatus(ctx, t, expectedPublishStatus)

}

func testPublishNoDescription(t *testing.T, expectationDB *fixtures.ExpectationDB, _ *fixtures.MinIO) {
	ctx := context.Background()

	callingUser := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, callingUser)

	claims := apitest.DefaultClaims(callingUser)

	// The dataset that will be in the collection
	expectedDatasets := apitest.NewExpectedPennsieveDatasets()
	dataset := expectedDatasets.NewPublished()

	// The collection
	expectedCollection := apitest.NewExpectedCollection().
		WithRandomID().
		WithNodeID().
		WithDescription("").
		WithUser(*callingUser.ID, pgdb.Owner).
		WithPublicDatasets(dataset)
	createCollectionResp := expectationDB.CreateCollection(ctx, t, expectedCollection)

	expectedPublishStatus := apitest.NewExpectedPublishStatus(publishing.FailedStatus, publishing.PublicationType, *callingUser.ID).
		WithCollectionID(createCollectionResp.ID)

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).
		Build()

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PublishCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
			WithBody(t, dto.PublishCollectionRequest{
				License: "Creative Commons",
				Tags:    []string{"test1, test2"},
			}).
			Build(),
		Container: apitest.NewTestContainer().
			WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
			WithCollectionsStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase),
		Config: apiConfig,
		Claims: &claims,
	}

	_, err := PublishCollection(ctx, params)
	var apiError *apierrors.Error
	require.ErrorAs(t, err, &apiError)

	assert.Equal(t, http.StatusBadRequest, apiError.StatusCode)
	assert.Contains(t, apiError.UserMessage, "description cannot be empty")

	expectationDB.RequirePublishStatus(ctx, t, expectedPublishStatus)

}

func testPublishContainsTombstones(t *testing.T, expectationDB *fixtures.ExpectationDB, _ *fixtures.MinIO) {
	ctx := context.Background()

	callingUser := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, callingUser)

	claims := apitest.DefaultClaims(callingUser)

	// The dataset that will be in the collection
	expectedDatasets := apitest.NewExpectedPennsieveDatasets()
	dataset := expectedDatasets.NewPublished()
	tombstone := expectedDatasets.NewUnpublished()

	// The collection
	expectedCollection := apitest.NewExpectedCollection().
		WithRandomID().
		WithNodeID().
		WithUser(*callingUser.ID, pgdb.Owner).
		WithPublicDatasets(dataset).
		WithTombstones(tombstone)
	createCollectionResp := expectationDB.CreateCollection(ctx, t, expectedCollection)

	expectedPublishStatus := apitest.NewExpectedPublishStatus(publishing.FailedStatus, publishing.PublicationType, *callingUser.ID).
		WithCollectionID(createCollectionResp.ID)

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).
		Build()

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PublishCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
			WithBody(t, dto.PublishCollectionRequest{
				License: "Creative Commons",
				Tags:    []string{"test1, test2"},
			}).
			Build(),
		Container: apitest.NewTestContainer().
			WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
			WithCollectionsStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase).
			WithDiscover(mocks.NewDiscover().WithGetDatasetsByDOIFunc(expectedDatasets.GetDatasetsByDOIFunc(t))),
		Config: apiConfig,
		Claims: &claims,
	}

	_, err := PublishCollection(ctx, params)
	var apiError *apierrors.Error
	require.ErrorAs(t, err, &apiError)

	assert.Equal(t, http.StatusBadRequest, apiError.StatusCode)
	assert.Contains(t, apiError.UserMessage, "unpublished")
	assert.Contains(t, apiError.UserMessage, tombstone.DOI)

	expectationDB.RequirePublishStatus(ctx, t, expectedPublishStatus)

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
		{
			"return Conflict when a publish is already in progress",
			testHandlePublishCollectionPublishAlreadyInProgress,
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
	callingUser := userstest.SeedUser1

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
	callingUser := userstest.SeedUser1

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
	callingUser := userstest.SeedUser1

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
	callingUser := userstest.SeedUser1

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
	callingUser := userstest.SeedUser1

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
	callingUser := userstest.SeedUser1
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
	callingUser := userstest.SeedUser1
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
				WithGetCollectionFunc(expectedCollection.GetCollectionFunc(t)).
				WithStartPublishFunc(expectedCollection.StartPublishFunc(t, callingUser.ID, publishing.PublicationType)).
				WithFinishPublishFunc(expectedCollection.FinishPublishFunc(t, publishing.CompletedStatus))

			expectedPublishedID := int64(14)
			expectedPublishedVersion := int64(1)
			mockInternalDiscover := mocks.NewInternalDiscover().
				WithPublishCollectionFunc(
					expectedCollection.PublishCollectionFunc(t,
						expectedPublishedID,
						expectedPublishedVersion,
						"PublishInProgress",
						apitest.VerifyPublishingUser(callingUser)),
				).
				WithFinalizeCollectionPublishFunc(
					expectedCollection.FinalizeCollectionPublishFunc(
						t,
						expectedPublishedID,
						expectedPublishedVersion,
						"PublishComplete"),
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
				require.Equal(t, expectedPublishedID, manifest.PennsieveDatasetID)
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

func testHandlePublishCollectionPublishAlreadyInProgress(t *testing.T) {
	ctx := context.Background()
	callingUser := userstest.SeedUser1

	publishRequest := dto.PublishCollectionRequest{
		License: "Creative Commons",
		Tags:    []string{"test"},
	}

	expectedCollection := apitest.NewExpectedCollection().
		WithRandomID().
		WithNodeID().
		WithUser(callingUser.ID, pgdb.Owner).
		WithDOIs(apitest.NewPennsieveDOI())

	mockCollectionStore := mocks.NewCollectionsStore().
		WithGetCollectionFunc(expectedCollection.GetCollectionFunc(t)).
		WithStartPublishFunc(func(_ context.Context, _ int64, _ int64, _ publishing.Type) error {
			return collections.ErrPublishInProgress
		})

	claims := apitest.DefaultClaims(callingUser)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PublishCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
			WithBody(t, publishRequest).
			Build(),
		Container: apitest.NewTestContainer().WithCollectionsStore(mockCollectionStore),
		Config:    apitest.NewConfigBuilder().WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).Build(),
		Claims:    &claims,
	}
	response, err := Handle(ctx, NewPublishCollectionRouteHandler(), params)
	require.NoError(t, err)

	assert.Equal(t, http.StatusConflict, response.StatusCode)

	assert.Contains(t, response.Body, "in progress")
}
