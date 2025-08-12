package routes

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/apijson"
	"github.com/pennsieve/collections-service/internal/api/config"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/publishing"
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/collections-service/internal/api/store/collections"
	"github.com/pennsieve/collections-service/internal/api/store/manifests"
	"github.com/pennsieve/collections-service/internal/api/store/users"
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

func TestPublishCollection(t *testing.T) {
	tests := []struct {
		scenario string
		tstFunc  func(t *testing.T, expectationDB *fixtures.ExpectationDB, minio *fixtures.MinIO)
	}{
		{"publish collection", testPublish},
		{"should return a 409 Conflict error if publish already in progress", testPublishNoConcurrent},
		{"should return Bad Request if description is empty", testPublishNoDescription},
		{"should return Bad Request if collection contains unpublished datasets", testPublishContainsTombstones},
		{"should clean up publish status and Discover if SaveManifest fails", testPublishSaveManifestFails},
		{"should clean up S3, publish status, and Discover if Discover finalize fails", testPublishFinalizeFails},
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
	expectedCollection := apitest.NewExpectedCollection().
		WithRandomID().
		WithNodeID().
		WithUser(*callingUser.ID, pgdb.Owner).
		WithRandomLicense().
		WithNTags(2).
		WithPublicDatasets(dataset)
	createCollectionResp := expectationDB.CreateCollection(ctx, t, expectedCollection)

	pennsieveConfig := apitest.PennsieveConfigWithOptions(config.WithPublishBucket(publishBucket))

	expectedPublishedDatasetID := rand.Int63n(5000) + 1
	expectedPublishedVersion := rand.Int63n(20) + 1
	expectedDiscoverPublishStatus := dto.PublishInProgress
	mockPublishDOICollectionResponse := service.PublishDOICollectionResponse{
		PublishedDatasetID: expectedPublishedDatasetID,
		PublishedVersion:   expectedPublishedVersion,
		Status:             expectedDiscoverPublishStatus,
	}

	expectedOrgServiceRole := apitest.ExpectedOrgServiceRole(pennsieveConfig.CollectionsIDSpace.ID)
	expectedDatasetServiceRole := expectedCollection.DatasetServiceRole(role.Owner)

	mockFinalizeDOICollectionResponse := service.FinalizeDOICollectionPublishResponse{Status: dto.PublishSucceeded}

	mockDiscoverMux := mocks.NewDiscoverMux(*pennsieveConfig.JWTSecretKey.Value).
		WithGetDatasetsByDOIFunc(ctx, t, expectedDatasets.GetDatasetsByDOIFunc(t)).
		WithPublishCollectionFunc(
			ctx,
			t,
			expectedCollection.PublishCollectionFunc(
				t,
				mockPublishDOICollectionResponse,
				apitest.VerifyPublishingUser(callingUser),
				apitest.VerifyInternalContributors(apitest.InternalContributor(callingUser)),
			),
			expectedOrgServiceRole,
			expectedDatasetServiceRole,
		).
		WithFinalizeCollectionPublishFunc(ctx, t,
			expectedCollection.FinalizeCollectionPublishFunc(t,
				mockFinalizeDOICollectionResponse,
				apitest.VerifyFinalizeDOICollectionRequest(expectedPublishedDatasetID, expectedPublishedVersion),
			),
			*expectedCollection.NodeID,
			expectedOrgServiceRole,
			expectedDatasetServiceRole)

	mockDiscoverServer := httptest.NewServer(mockDiscoverMux)
	defer mockDiscoverServer.Close()

	pennsieveConfig.DiscoverServiceURL = mockDiscoverServer.URL

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(pennsieveConfig).
		Build()

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PublishCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
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
	assert.Equal(t, mockFinalizeDOICollectionResponse.Status, resp.Status)

	expectedPublishStatus := collectionstest.NewExpectedCompletedPublishStatus(createCollectionResp.ID, *callingUser.ID)

	expectationDB.RequirePublishStatus(ctx, t, expectedPublishStatus, nil)

	manifestKey := publishing.ManifestS3Key(resp.PublishedDatasetID)
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

	assert.Equal(t, expectedCollection.Tags, actualManifest.Keywords)
	assert.Equal(t, *expectedCollection.License, actualManifest.License)

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

	assert.Equal(t, expectedCollection.DOIs.Strings(), actualManifest.References.IDs)

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

	existingPublishStatus := collectionstest.NewInProgressPublishStatus(createCollectionResp.ID, *callingUser.ID)
	expectationDB.CreatePublishStatus(ctx, t, existingPublishStatus)

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).
		Build()

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PublishCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
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

	expectedPublishStatus := collectionstest.NewExpectedInProgressPublishStatus(createCollectionResp.ID, *callingUser.ID)
	expectationDB.RequirePublishStatus(ctx, t, expectedPublishStatus, &existingPublishStatus)

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
		WithPublicDatasets(dataset).
		WithRandomLicense().
		WithNTags(2)
	createCollectionResp := expectationDB.CreateCollection(ctx, t, expectedCollection)

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).
		Build()

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PublishCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
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

	expectedPublishStatus := collectionstest.NewExpectedFailedPublishStatus(createCollectionResp.ID, *callingUser.ID)
	expectationDB.RequirePublishStatus(ctx, t, expectedPublishStatus, nil)

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
		WithRandomLicense().
		WithNTags(3).
		WithPublicDatasets(dataset).
		WithTombstones(tombstone)
	createCollectionResp := expectationDB.CreateCollection(ctx, t, expectedCollection)

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).
		Build()

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PublishCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
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

	expectedPublishStatus := collectionstest.NewExpectedFailedPublishStatus(createCollectionResp.ID, *callingUser.ID)
	expectationDB.RequirePublishStatus(ctx, t, expectedPublishStatus, nil)

}

func testPublishSaveManifestFails(t *testing.T, expectationDB *fixtures.ExpectationDB, _ *fixtures.MinIO) {
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
	expectedCollection := apitest.NewExpectedCollection().
		WithRandomID().
		WithNodeID().
		WithUser(*callingUser.ID, pgdb.Owner).
		WithPublicDatasets(dataset).
		WithRandomLicense().
		WithNTags(4)
	createCollectionResp := expectationDB.CreateCollection(ctx, t, expectedCollection)

	pennsieveConfig := apitest.PennsieveConfigWithOptions()

	expectedPublishedDatasetID := rand.Int63n(5000) + 1
	expectedPublishedVersion := rand.Int63n(20) + 1
	expectedDiscoverPublishStatus := dto.PublishInProgress
	mockPublishDOICollectionResponse := service.PublishDOICollectionResponse{
		PublishedDatasetID: expectedPublishedDatasetID,
		PublishedVersion:   expectedPublishedVersion,
		Status:             expectedDiscoverPublishStatus,
	}

	expectedOrgServiceRole := apitest.ExpectedOrgServiceRole(pennsieveConfig.CollectionsIDSpace.ID)
	expectedDatasetServiceRole := expectedCollection.DatasetServiceRole(role.Owner)

	mockDiscoverMux := mocks.NewDiscoverMux(*pennsieveConfig.JWTSecretKey.Value).
		WithGetDatasetsByDOIFunc(ctx, t, expectedDatasets.GetDatasetsByDOIFunc(t)).
		WithPublishCollectionFunc(
			ctx,
			t,
			expectedCollection.PublishCollectionFunc(
				t,
				mockPublishDOICollectionResponse,
				apitest.VerifyPublishingUser(callingUser),
				apitest.VerifyInternalContributors(apitest.InternalContributor(callingUser)),
			),
			expectedOrgServiceRole,
			expectedDatasetServiceRole,
		).
		WithFinalizeCollectionPublishFunc(ctx, t,
			expectedCollection.FinalizeCollectionPublishFunc(t,
				service.FinalizeDOICollectionPublishResponse{Status: dto.PublishSucceeded},
				apitest.VerifyFailedFinalizeDOICollectionRequest(expectedPublishedDatasetID, expectedPublishedVersion),
			),
			*expectedCollection.NodeID,
			expectedOrgServiceRole,
			expectedDatasetServiceRole)

	mockDiscoverServer := httptest.NewServer(mockDiscoverMux)
	defer mockDiscoverServer.Close()

	pennsieveConfig.DiscoverServiceURL = mockDiscoverServer.URL

	mockManifestStore := mocks.NewManifestStore().WithSaveManifestFunc(func(ctx context.Context, key string, manifest publishing.ManifestV5) (manifests.SaveManifestResponse, error) {
		return manifests.SaveManifestResponse{}, errors.New("unexpected S3 error")
	})

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(pennsieveConfig).
		Build()

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PublishCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
			Build(),
		Container: apitest.NewTestContainer().
			WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
			WithCollectionsStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase).
			WithUsersStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase).
			WithHTTPTestDiscover(mockDiscoverServer.URL).
			WithHTTPTestInternalDiscover(pennsieveConfig).
			WithManifestStore(mockManifestStore),
		Config: apiConfig,
		Claims: &claims,
	}

	_, err := PublishCollection(ctx, params)

	var apiError *apierrors.Error
	require.ErrorAs(t, err, &apiError)

	assert.Equal(t, http.StatusInternalServerError, apiError.StatusCode)

	expectedPublishStatus := collectionstest.NewExpectedFailedPublishStatus(createCollectionResp.ID, *callingUser.ID)

	expectationDB.RequirePublishStatus(ctx, t, expectedPublishStatus, nil)

}

func testPublishFinalizeFails(t *testing.T, expectationDB *fixtures.ExpectationDB, minio *fixtures.MinIO) {
	ctx := context.Background()

	publishBucket := minio.CreatePublishBucket(ctx, t)

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
		WithUser(*callingUser.ID, pgdb.Owner).
		WithPublicDatasets(dataset).
		WithRandomLicense().
		WithNTags(2)

	createCollectionResp := expectationDB.CreateCollection(ctx, t, expectedCollection)

	pennsieveConfig := apitest.PennsieveConfigWithOptions(config.WithPublishBucket(publishBucket))

	expectedPublishedDatasetID := int64(26)
	expectedPublishedVersion := int64(1)
	mockPublishDOICollectionResponse := service.PublishDOICollectionResponse{
		PublishedDatasetID: expectedPublishedDatasetID,
		PublishedVersion:   expectedPublishedVersion,
		Status:             dto.PublishInProgress,
	}

	s3Key := publishing.ManifestS3Key(expectedPublishedDatasetID)

	expectedOrgServiceRole := apitest.ExpectedOrgServiceRole(pennsieveConfig.CollectionsIDSpace.ID)
	expectedDatasetServiceRole := expectedCollection.DatasetServiceRole(role.Owner)

	var actualFinalizeRequests []service.FinalizeDOICollectionPublishRequest
	mockDiscoverMux := mocks.NewDiscoverMux(*pennsieveConfig.JWTSecretKey.Value).
		WithGetDatasetsByDOIFunc(ctx, t, expectedDatasets.GetDatasetsByDOIFunc(t)).
		WithPublishCollectionFunc(
			ctx,
			t,
			expectedCollection.PublishCollectionFunc(
				t,
				mockPublishDOICollectionResponse,
				apitest.VerifyPublishingUser(callingUser),
				apitest.VerifyInternalContributors(apitest.InternalContributor(callingUser)),
			),
			expectedOrgServiceRole,
			expectedDatasetServiceRole,
		).
		WithFinalizeCollectionPublishFunc(ctx, t, func(_ context.Context, _ int64, _ string, _ role.Role, request service.FinalizeDOICollectionPublishRequest) (service.FinalizeDOICollectionPublishResponse, error) {
			actualFinalizeRequests = append(actualFinalizeRequests, request)
			return service.FinalizeDOICollectionPublishResponse{}, errors.New("unexpected Discover error")
		}, *expectedCollection.NodeID, expectedOrgServiceRole, expectedDatasetServiceRole)

	mockDiscoverServer := httptest.NewServer(mockDiscoverMux)
	defer mockDiscoverServer.Close()

	pennsieveConfig.DiscoverServiceURL = mockDiscoverServer.URL

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(pennsieveConfig).
		Build()

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PublishCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
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

	_, err := PublishCollection(ctx, params)
	var apiError *apierrors.Error
	require.ErrorAs(t, err, &apiError)

	assert.Equal(t, http.StatusInternalServerError, apiError.StatusCode)

	expectedPublishStatus := collectionstest.NewExpectedFailedPublishStatus(createCollectionResp.ID, *callingUser.ID)

	expectationDB.RequirePublishStatus(ctx, t, expectedPublishStatus, nil)

	minio.RequireNoObject(ctx, t, publishBucket, s3Key)

	require.Len(t, actualFinalizeRequests, 2)
	assert.True(t, actualFinalizeRequests[0].PublishSuccess)
	assert.False(t, actualFinalizeRequests[1].PublishSuccess)

}

// TestHandlePublishCollection tests that run the Handle wrapper around PublishCollection
func TestHandlePublishCollection(t *testing.T) {
	tests := []struct {
		name    string
		tstFunc func(t *testing.T)
	}{
		{
			"return Bad Request collection license is empty",
			testHandlePublishCollectionEmptyLicense,
		},
		{
			"return Bad Request when collection license is null",
			testHandlePublishCollectionNoLicense,
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

func testHandlePublishCollectionEmptyLicense(t *testing.T) {
	ctx := context.Background()
	callingUser := userstest.SeedUser1

	expectedCollection := apitest.NewExpectedCollection().
		WithRandomID().
		WithNodeID().
		WithUser(callingUser.ID, pgdb.Owner).
		WithLicense("").
		WithNTags(1)

	mockCollectionStore := mocks.NewCollectionsStore().
		WithGetCollectionFunc(expectedCollection.GetCollectionFunc(t)).
		WithStartPublishFunc(expectedCollection.StartPublishFunc(t, callingUser.ID, publishing.PublicationType)).
		WithFinishPublishFunc(expectedCollection.FinishPublishFunc(t, publishing.FailedStatus))

	claims := apitest.DefaultClaims(callingUser)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PublishCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
			Build(),
		Container: apitest.NewTestContainer().WithCollectionsStore(mockCollectionStore),
		Config:    apitest.NewConfigBuilder().WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).Build(),
		Claims:    &claims,
	}
	response, err := Handle(ctx, NewPublishCollectionRouteHandler(), params)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, response.StatusCode)

	assert.Contains(t, response.Body, `missing required license`)
}

func testHandlePublishCollectionNoLicense(t *testing.T) {
	ctx := context.Background()
	callingUser := userstest.SeedUser1
	expectedCollection := apitest.NewExpectedCollection().
		WithRandomID().
		WithNodeID().
		WithUser(callingUser.ID, pgdb.Owner).
		WithNTags(1)

	mockCollectionStore := mocks.NewCollectionsStore().
		WithGetCollectionFunc(expectedCollection.GetCollectionFunc(t)).
		WithStartPublishFunc(expectedCollection.StartPublishFunc(t, callingUser.ID, publishing.PublicationType)).
		WithFinishPublishFunc(expectedCollection.FinishPublishFunc(t, publishing.FailedStatus))

	claims := apitest.DefaultClaims(callingUser)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PublishCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
			Build(),
		Container: apitest.NewTestContainer().WithCollectionsStore(mockCollectionStore),
		Config:    apitest.NewConfigBuilder().WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).Build(),
		Claims:    &claims,
	}
	response, err := Handle(ctx, NewPublishCollectionRouteHandler(), params)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, response.StatusCode)

	assert.Contains(t, response.Body, "missing required license")

}

func testHandlePublishCollectionNoTags(t *testing.T) {
	ctx := context.Background()
	callingUser := userstest.SeedUser1

	expectedCollection := apitest.NewExpectedCollection().
		WithRandomID().
		WithNodeID().
		WithUser(callingUser.ID, pgdb.Owner).
		WithRandomLicense()

	mockCollectionStore := mocks.NewCollectionsStore().
		WithGetCollectionFunc(expectedCollection.GetCollectionFunc(t)).
		WithStartPublishFunc(expectedCollection.StartPublishFunc(t, callingUser.ID, publishing.PublicationType)).
		WithFinishPublishFunc(expectedCollection.FinishPublishFunc(t, publishing.FailedStatus))

	claims := apitest.DefaultClaims(callingUser)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PublishCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
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

	expectedCollection := apitest.NewExpectedCollection().
		WithRandomID().
		WithNodeID().
		WithUser(callingUser.ID, pgdb.Owner).
		WithRandomLicense()
	expectedCollection.Tags = []string{}

	mockCollectionStore := mocks.NewCollectionsStore().
		WithGetCollectionFunc(expectedCollection.GetCollectionFunc(t)).
		WithStartPublishFunc(expectedCollection.StartPublishFunc(t, callingUser.ID, publishing.PublicationType)).
		WithFinishPublishFunc(expectedCollection.FinishPublishFunc(t, publishing.FailedStatus))

	claims := apitest.DefaultClaims(callingUser)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PublishCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
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
			expectedCollection := apitest.NewExpectedCollection().
				WithRandomID().
				WithNodeID().
				WithUser(callingUser.ID, tooLowPerm).
				WithRandomLicense().
				WithNTags(2)

			mockCollectionStore := mocks.NewCollectionsStore().
				WithGetCollectionFunc(expectedCollection.GetCollectionFunc(t, nil))

			params := Params{
				Request: apitest.NewAPIGatewayRequestBuilder(PublishCollectionRouteKey).
					WithClaims(claims).
					WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
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

			expectedCollection := apitest.NewExpectedCollection().
				WithRandomID().
				WithNodeID().
				WithUser(callingUser.ID, okPerm).
				WithPublicDatasets(dataset).
				WithRandomLicense().
				WithNTags(2)

			mockCollectionStore := mocks.NewCollectionsStore().
				WithGetCollectionFunc(expectedCollection.GetCollectionFunc(t, nil)).
				WithStartPublishFunc(expectedCollection.StartPublishFunc(t, callingUser.ID, publishing.PublicationType)).
				WithFinishPublishFunc(expectedCollection.FinishPublishFunc(t, publishing.CompletedStatus))

			expectedPublishedID := int64(14)
			expectedPublishedVersion := int64(1)
			mockPublishDOICollectionResponse := service.PublishDOICollectionResponse{
				PublishedDatasetID: expectedPublishedID,
				PublishedVersion:   expectedPublishedVersion,
				Status:             dto.PublishInProgress,
			}

			mockFinalizeDOICollectionResponse := service.FinalizeDOICollectionPublishResponse{Status: dto.PublishSucceeded}
			expectedManifestS3VersionID := uuid.NewString()

			var capturedManifestTotalSize int64
			mockManifestStore := mocks.NewManifestStore().WithSaveManifestFunc(func(ctx context.Context, key string, manifest publishing.ManifestV5) (manifests.SaveManifestResponse, error) {
				require.Equal(t, publishing.ManifestS3Key(expectedPublishedID), key)
				require.Equal(t, expectedPublishedID, manifest.PennsieveDatasetID)
				require.Equal(t, expectedCollection.Name, manifest.Name)
				require.Equal(t, callingUser.LastName, manifest.Creator.LastName)
				capturedManifestTotalSize = manifest.TotalSize()
				return manifests.SaveManifestResponse{
					S3VersionID: expectedManifestS3VersionID,
				}, nil
			})

			mockInternalDiscover := mocks.NewInternalDiscover().
				WithPublishCollectionFunc(
					expectedCollection.PublishCollectionFunc(t, mockPublishDOICollectionResponse,
						apitest.VerifyPublishingUser(callingUser),
						apitest.VerifyInternalContributors(apitest.InternalContributor(callingUser)),
					),
				).
				WithFinalizeCollectionPublishFunc(
					expectedCollection.FinalizeCollectionPublishFunc(t, mockFinalizeDOICollectionResponse,
						apitest.VerifyFinalizeDOICollectionRequest(expectedPublishedID, expectedPublishedVersion),
						apitest.VerifyFinalizeDOICollectionRequestS3VersionID(expectedManifestS3VersionID),
						apitest.VerifyFinalizeDOICollectionRequestTotalSize(func() int64 {
							return capturedManifestTotalSize
						}),
					),
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

			params := Params{
				Request: apitest.NewAPIGatewayRequestBuilder(PublishCollectionRouteKey).
					WithClaims(claims).
					WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
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

	expectedCollection := apitest.NewExpectedCollection().
		WithRandomID().
		WithNodeID().
		WithUser(callingUser.ID, pgdb.Owner).
		WithDOIs(apitest.NewPennsieveDOI())

	mockCollectionStore := mocks.NewCollectionsStore().
		WithGetCollectionFunc(expectedCollection.GetCollectionFunc(t, nil)).
		WithStartPublishFunc(func(_ context.Context, _ int64, _ int64, _ publishing.Type) error {
			return collections.ErrPublishInProgress
		})

	claims := apitest.DefaultClaims(callingUser)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PublishCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
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
