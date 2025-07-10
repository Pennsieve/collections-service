package routes

import (
	"context"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/collections-service/internal/api/store/collections"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/pennsieve/collections-service/internal/test/fixtures"
	"github.com/pennsieve/collections-service/internal/test/mocks"
	"github.com/pennsieve/collections-service/internal/test/userstest"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPatchCollection(t *testing.T) {
	tests := []struct {
		scenario string
		tstFunc  func(t *testing.T, expectationDB *fixtures.ExpectationDB)
	}{
		{"update collection name", testPatchCollectionName},
		{"update collection description", testPatchCollectionDescription},
		{"update collection name and description", testPatchCollectionNameAndDescription},
		{"remove DOIs from collection", testPatchCollectionRemoveDOIs},
		{"add DOIs to collection", testPatchCollectionAddDOIs},
		{"update collection", testPatchCollection},
		{"update asking to add an unpublished DOI should fail", testPatchCollectionAddUnpublished},
		{"update asking to remove a non-existent DOI should succeed", testPatchCollectionRemoveNonExistentDOI},
		{"update asking to add an already existing DOI should succeed", testPatchCollectionAddExistingDOI},
		{"update non-existent collection should return ErrCollectionNotFound", testPatchCollectionNonExistent},
		{"update DOIs on non-existent collection should return ErrCollectionNotFound", testPatchCollectionNonExistentDOIUpdateOnly},
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

func testPatchCollectionName(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	expectedDatasets := apitest.NewExpectedPennsieveDatasets()

	user := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	expectedCollection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user.ID, pgdb.Owner).WithPublicDatasets(expectedDatasets.NewPublished(apitest.NewPublicContributor()))
	createResp := expectationDB.CreateCollection(ctx, t, expectedCollection)
	collectionID := createResp.ID

	newName := uuid.NewString()
	update := dto.PatchCollectionRequest{Name: &newName}

	mockDiscoverServer := httptest.NewServer(mocks.ToDiscoverHandlerFunc(ctx, t, expectedDatasets.GetDatasetsByDOIFunc(t)))
	defer mockDiscoverServer.Close()

	claims := apitest.DefaultClaims(user)

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(apitest.PennsieveConfig(mockDiscoverServer.URL)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
		WithCollectionsStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase).
		WithHTTPTestDiscover(mockDiscoverServer.URL)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PatchCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
			WithBody(t, update).
			Build(),
		Container: container,
		Config:    apiConfig,
		Claims:    &claims,
	}

	updatedCollection, err := PatchCollection(ctx, params)
	require.NoError(t, err)

	expectedCollection.Name = newName
	assertEqualExpectedGetCollectionResponse(t, expectedCollection, updatedCollection, expectedDatasets)

	expectationDB.RequireCollection(ctx, t, expectedCollection, collectionID)
}

func testPatchCollectionDescription(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	expectedDatasets := apitest.NewExpectedPennsieveDatasets()

	user := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	expectedCollection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user.ID, pgdb.Owner).
		WithPublicDatasets(
			expectedDatasets.NewPublished(apitest.NewPublicContributor()),
			expectedDatasets.NewPublished(apitest.NewPublicContributor(apitest.WithOrcid())),
		)
	createResp := expectationDB.CreateCollection(ctx, t, expectedCollection)
	collectionID := createResp.ID

	newDescription := uuid.NewString()
	update := dto.PatchCollectionRequest{Description: &newDescription}

	mockDiscoverServer := httptest.NewServer(mocks.ToDiscoverHandlerFunc(ctx, t, expectedDatasets.GetDatasetsByDOIFunc(t)))
	defer mockDiscoverServer.Close()

	claims := apitest.DefaultClaims(user)

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(apitest.PennsieveConfig(mockDiscoverServer.URL)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
		WithCollectionsStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase).
		WithHTTPTestDiscover(mockDiscoverServer.URL)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PatchCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
			WithBody(t, update).
			Build(),
		Container: container,
		Config:    apiConfig,
		Claims:    &claims,
	}

	updatedCollection, err := PatchCollection(ctx, params)
	require.NoError(t, err)

	expectedCollection.Description = newDescription
	assertEqualExpectedGetCollectionResponse(t, expectedCollection, updatedCollection, expectedDatasets)

	expectationDB.RequireCollection(ctx, t, expectedCollection, collectionID)
}

func testPatchCollectionNameAndDescription(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	expectedDatasets := apitest.NewExpectedPennsieveDatasets()

	user := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	expectedCollection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user.ID, pgdb.Owner).
		WithPublicDatasets(
			expectedDatasets.NewPublished(apitest.NewPublicContributor()),
			expectedDatasets.NewPublished(apitest.NewPublicContributor(apitest.WithOrcid())),
		)
	createResp := expectationDB.CreateCollection(ctx, t, expectedCollection)
	collectionID := createResp.ID

	newName := uuid.NewString()
	newDescription := uuid.NewString()
	update := dto.PatchCollectionRequest{Name: &newName, Description: &newDescription}

	mockDiscoverServer := httptest.NewServer(mocks.ToDiscoverHandlerFunc(ctx, t, expectedDatasets.GetDatasetsByDOIFunc(t)))
	defer mockDiscoverServer.Close()

	claims := apitest.DefaultClaims(user)

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(apitest.PennsieveConfig(mockDiscoverServer.URL)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
		WithCollectionsStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase).
		WithHTTPTestDiscover(mockDiscoverServer.URL)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PatchCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
			WithBody(t, update).
			Build(),
		Container: container,
		Config:    apiConfig,
		Claims:    &claims,
	}

	updatedCollection, err := PatchCollection(ctx, params)
	require.NoError(t, err)

	expectedCollection.Name = newName
	expectedCollection.Description = newDescription
	assertEqualExpectedGetCollectionResponse(t, expectedCollection, updatedCollection, expectedDatasets)

	expectationDB.RequireCollection(ctx, t, expectedCollection, collectionID)
}

func testPatchCollectionRemoveDOIs(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	expectedDatasets := apitest.NewExpectedPennsieveDatasets()

	user := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	datasetToRemove1 := expectedDatasets.NewPublished(apitest.NewPublicContributor())
	datasetToKeep1 := expectedDatasets.NewPublished(apitest.NewPublicContributor(apitest.WithMiddleInitial()))
	datasetToRemove2 := expectedDatasets.NewPublished(apitest.NewPublicContributor(apitest.WithOrcid()))
	datasetToKeep2 := expectedDatasets.NewPublished(apitest.NewPublicContributor())

	expectedCollection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user.ID, pgdb.Owner).
		WithPublicDatasets(datasetToRemove1, datasetToKeep1, datasetToRemove2, datasetToKeep2)
	createResp := expectationDB.CreateCollection(ctx, t, expectedCollection)
	collectionID := createResp.ID

	update := dto.PatchCollectionRequest{DOIs: &dto.PatchDOIs{Remove: []string{datasetToRemove2.DOI, datasetToRemove1.DOI}}}

	mockDiscoverServer := httptest.NewServer(mocks.ToDiscoverHandlerFunc(ctx, t, expectedDatasets.GetDatasetsByDOIFunc(t)))
	defer mockDiscoverServer.Close()

	claims := apitest.DefaultClaims(user)

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(apitest.PennsieveConfig(mockDiscoverServer.URL)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
		WithCollectionsStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase).
		WithHTTPTestDiscover(mockDiscoverServer.URL)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PatchCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
			WithBody(t, update).
			Build(),
		Container: container,
		Config:    apiConfig,
		Claims:    &claims,
	}

	updatedCollection, err := PatchCollection(ctx, params)
	require.NoError(t, err)

	expectedCollection.SetPublicDatasets(datasetToKeep1, datasetToKeep2)
	assertEqualExpectedGetCollectionResponse(t, expectedCollection, updatedCollection, expectedDatasets)

	expectationDB.RequireCollection(ctx, t, expectedCollection, collectionID)
}

func testPatchCollectionAddDOIs(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	expectedDatasets := apitest.NewExpectedPennsieveDatasets()

	user := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	datasetToAdd1 := expectedDatasets.NewPublished(apitest.NewPublicContributor())
	dataset1 := expectedDatasets.NewPublished(apitest.NewPublicContributor(apitest.WithMiddleInitial()))
	datasetToAdd2 := expectedDatasets.NewPublished(apitest.NewPublicContributor(apitest.WithOrcid()))
	dataset2 := expectedDatasets.NewPublished(apitest.NewPublicContributor())

	expectedCollection := apitest.NewExpectedCollection().WithRandomID().WithNodeID().WithUser(*user.ID, pgdb.Owner).
		WithPublicDatasets(dataset1, dataset2)
	createResp := expectationDB.CreateCollection(ctx, t, expectedCollection)
	collectionID := createResp.ID

	update := dto.PatchCollectionRequest{DOIs: &dto.PatchDOIs{Add: []string{datasetToAdd1.DOI, datasetToAdd2.DOI}}}

	mockDiscoverServer := httptest.NewServer(mocks.ToDiscoverHandlerFunc(ctx, t, expectedDatasets.GetDatasetsByDOIFunc(t)))
	defer mockDiscoverServer.Close()

	claims := apitest.DefaultClaims(user)

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(apitest.PennsieveConfig(mockDiscoverServer.URL)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
		WithCollectionsStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase).
		WithHTTPTestDiscover(mockDiscoverServer.URL)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PatchCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
			WithBody(t, update).
			Build(),
		Container: container,
		Config:    apiConfig,
		Claims:    &claims,
	}

	updatedCollection, err := PatchCollection(ctx, params)
	require.NoError(t, err)

	expectedCollection.SetPublicDatasets(dataset1, dataset2, datasetToAdd1, datasetToAdd2)
	assertEqualExpectedGetCollectionResponse(t, expectedCollection, updatedCollection, expectedDatasets)

	expectationDB.RequireCollection(ctx, t, expectedCollection, collectionID)
}

func testPatchCollection(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	expectedDatasets := apitest.NewExpectedPennsieveDatasets()

	user := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	datasetToAdd1 := expectedDatasets.NewPublished(apitest.NewPublicContributor())
	dataset1 := expectedDatasets.NewPublished(apitest.NewPublicContributor(apitest.WithMiddleInitial()))
	datasetToAdd2 := expectedDatasets.NewPublished(apitest.NewPublicContributor(apitest.WithOrcid()))
	dataset2 := expectedDatasets.NewPublished(apitest.NewPublicContributor())
	datasetToRemove := expectedDatasets.NewPublished(apitest.NewPublicContributor(apitest.WithOrcid()))

	expectedCollection := apitest.NewExpectedCollection().WithRandomID().WithNodeID().WithUser(*user.ID, pgdb.Owner).
		WithPublicDatasets(dataset1, dataset2, datasetToRemove)
	createResp := expectationDB.CreateCollection(ctx, t, expectedCollection)
	collectionID := createResp.ID

	newName := uuid.NewString()
	newDescription := uuid.NewString()
	update := dto.PatchCollectionRequest{
		Name:        &newName,
		Description: &newDescription,
		DOIs: &dto.PatchDOIs{
			Remove: []string{datasetToRemove.DOI},
			Add:    []string{datasetToAdd1.DOI, datasetToAdd2.DOI},
		},
	}

	mockDiscoverServer := httptest.NewServer(mocks.ToDiscoverHandlerFunc(ctx, t, expectedDatasets.GetDatasetsByDOIFunc(t)))
	defer mockDiscoverServer.Close()

	claims := apitest.DefaultClaims(user)

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(apitest.PennsieveConfig(mockDiscoverServer.URL)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
		WithCollectionsStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase).
		WithHTTPTestDiscover(mockDiscoverServer.URL)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PatchCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
			WithBody(t, update).
			Build(),
		Container: container,
		Config:    apiConfig,
		Claims:    &claims,
	}

	updatedCollection, err := PatchCollection(ctx, params)
	require.NoError(t, err)

	expectedCollection.Name = newName
	expectedCollection.Description = newDescription
	expectedCollection.SetPublicDatasets(dataset1, dataset2, datasetToAdd1, datasetToAdd2)
	assertEqualExpectedGetCollectionResponse(t, expectedCollection, updatedCollection, expectedDatasets)

	expectationDB.RequireCollection(ctx, t, expectedCollection, collectionID)
}

func testPatchCollectionAddUnpublished(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	expectedDatasets := apitest.NewExpectedPennsieveDatasets()

	user := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	publishedToAdd := expectedDatasets.NewPublished(apitest.NewPublicContributor())
	published1 := expectedDatasets.NewPublished(apitest.NewPublicContributor(apitest.WithMiddleInitial()))
	unpublishedToAdd := expectedDatasets.NewUnpublished()
	published2 := expectedDatasets.NewPublished(apitest.NewPublicContributor())

	expectedCollection := apitest.NewExpectedCollection().WithRandomID().WithNodeID().WithUser(*user.ID, pgdb.Owner).
		WithPublicDatasets(published1, published2)
	createResp := expectationDB.CreateCollection(ctx, t, expectedCollection)
	collectionID := createResp.ID

	update := dto.PatchCollectionRequest{DOIs: &dto.PatchDOIs{Add: []string{publishedToAdd.DOI, unpublishedToAdd.DOI}}}

	mockDiscoverServer := httptest.NewServer(mocks.ToDiscoverHandlerFunc(ctx, t, expectedDatasets.GetDatasetsByDOIFunc(t)))
	defer mockDiscoverServer.Close()

	claims := apitest.DefaultClaims(user)

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(apitest.PennsieveConfig(mockDiscoverServer.URL)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
		WithCollectionsStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase).
		WithHTTPTestDiscover(mockDiscoverServer.URL)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PatchCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
			WithBody(t, update).
			Build(),
		Container: container,
		Config:    apiConfig,
		Claims:    &claims,
	}

	_, err := PatchCollection(ctx, params)
	require.Error(t, err)

	var badRequest *apierrors.Error
	require.ErrorAs(t, err, &badRequest)
	assert.Equal(t, http.StatusBadRequest, badRequest.StatusCode)
	assert.Contains(t, badRequest.UserMessage, unpublishedToAdd.DOI)
	assert.Contains(t, badRequest.UserMessage, unpublishedToAdd.Status)

	// collection should be unchanged
	expectationDB.RequireCollection(ctx, t, expectedCollection, collectionID)
}

func testPatchCollectionRemoveNonExistentDOI(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	expectedDatasets := apitest.NewExpectedPennsieveDatasets()

	user := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	datasetToRemove1 := expectedDatasets.NewPublished(apitest.NewPublicContributor())
	datasetToKeep1 := expectedDatasets.NewPublished(apitest.NewPublicContributor(apitest.WithMiddleInitial()))
	datasetToRemove2 := expectedDatasets.NewPublished(apitest.NewPublicContributor(apitest.WithOrcid()))
	datasetToKeep2 := expectedDatasets.NewPublished(apitest.NewPublicContributor())

	expectedCollection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user.ID, pgdb.Owner).
		WithPublicDatasets(datasetToRemove1, datasetToKeep1, datasetToRemove2, datasetToKeep2)
	createResp := expectationDB.CreateCollection(ctx, t, expectedCollection)
	collectionID := createResp.ID

	update := dto.PatchCollectionRequest{
		// include a third DOI that is not part of the collection
		DOIs: &dto.PatchDOIs{Remove: []string{datasetToRemove2.DOI, datasetToRemove1.DOI, uuid.NewString()}},
	}

	mockDiscoverServer := httptest.NewServer(mocks.ToDiscoverHandlerFunc(ctx, t, expectedDatasets.GetDatasetsByDOIFunc(t)))
	defer mockDiscoverServer.Close()

	claims := apitest.DefaultClaims(user)

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(apitest.PennsieveConfig(mockDiscoverServer.URL)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
		WithCollectionsStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase).
		WithHTTPTestDiscover(mockDiscoverServer.URL)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PatchCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
			WithBody(t, update).
			Build(),
		Container: container,
		Config:    apiConfig,
		Claims:    &claims,
	}

	updatedCollection, err := PatchCollection(ctx, params)
	require.NoError(t, err)

	expectedCollection.SetPublicDatasets(datasetToKeep1, datasetToKeep2)
	assertEqualExpectedGetCollectionResponse(t, expectedCollection, updatedCollection, expectedDatasets)

	expectationDB.RequireCollection(ctx, t, expectedCollection, collectionID)

}

func testPatchCollectionAddExistingDOI(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	expectedDatasets := apitest.NewExpectedPennsieveDatasets()

	user := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	datasetToAdd1 := expectedDatasets.NewPublished(apitest.NewPublicContributor())
	dataset1 := expectedDatasets.NewPublished(apitest.NewPublicContributor(apitest.WithMiddleInitial()))
	datasetToAdd2 := expectedDatasets.NewPublished(apitest.NewPublicContributor(apitest.WithOrcid()))
	dataset2 := expectedDatasets.NewPublished(apitest.NewPublicContributor())

	expectedCollection := apitest.NewExpectedCollection().WithRandomID().WithNodeID().WithUser(*user.ID, pgdb.Owner).
		WithPublicDatasets(dataset1, dataset2)
	createResp := expectationDB.CreateCollection(ctx, t, expectedCollection)
	collectionID := createResp.ID

	update := dto.PatchCollectionRequest{
		DOIs: &dto.PatchDOIs{
			// include one of the DOIs that are already in the collection
			Add: []string{datasetToAdd1.DOI, dataset1.DOI, datasetToAdd2.DOI},
		},
	}

	mockDiscoverServer := httptest.NewServer(mocks.ToDiscoverHandlerFunc(ctx, t, expectedDatasets.GetDatasetsByDOIFunc(t)))
	defer mockDiscoverServer.Close()

	claims := apitest.DefaultClaims(user)

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(apitest.PennsieveConfig(mockDiscoverServer.URL)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
		WithCollectionsStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase).
		WithHTTPTestDiscover(mockDiscoverServer.URL)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PatchCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
			WithBody(t, update).
			Build(),
		Container: container,
		Config:    apiConfig,
		Claims:    &claims,
	}

	updatedCollection, err := PatchCollection(ctx, params)
	require.NoError(t, err)

	expectedCollection.SetPublicDatasets(dataset1, dataset2, datasetToAdd1, datasetToAdd2)
	assertEqualExpectedGetCollectionResponse(t, expectedCollection, updatedCollection, expectedDatasets)

	expectationDB.RequireCollection(ctx, t, expectedCollection, collectionID)
}

func testPatchCollectionNonExistent(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	user := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	nonExistentNodeID := uuid.NewString()

	newName := uuid.NewString()
	update := dto.PatchCollectionRequest{Name: &newName}

	claims := apitest.DefaultClaims(user)

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
		WithCollectionsStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase)
	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PatchCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, nonExistentNodeID).
			WithBody(t, update).
			Build(),
		Container: container,
		Config:    apiConfig,
		Claims:    &claims,
	}

	_, err := PatchCollection(ctx, params)
	var notFoundError *apierrors.Error
	require.ErrorAs(t, err, &notFoundError)
	assert.Equal(t, http.StatusNotFound, notFoundError.StatusCode)

}

func testPatchCollectionNonExistentDOIUpdateOnly(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	user := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	nonExistentNodeID := uuid.NewString()

	doi := uuid.NewString()
	update := dto.PatchCollectionRequest{DOIs: &dto.PatchDOIs{Add: []string{doi}}}

	claims := apitest.DefaultClaims(user)

	apiConfig := apitest.NewConfigBuilder().
		WithPostgresDBConfig(test.PostgresDBConfig(t)).
		WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
		WithCollectionsStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase)
	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PatchCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, nonExistentNodeID).
			WithBody(t, update).
			Build(),
		Container: container,
		Config:    apiConfig,
		Claims:    &claims,
	}

	_, err := PatchCollection(ctx, params)
	var notFoundError *apierrors.Error
	require.ErrorAs(t, err, &notFoundError)
	assert.Equal(t, http.StatusNotFound, notFoundError.StatusCode)
}

func TestGetUpdateRequestAddDOIs(t *testing.T) {
	doiToAdd1 := apitest.NewPennsieveDOI()
	doi1 := apitest.NewPennsieveDOI()
	doiToAdd2 := apitest.NewPennsieveDOI()
	doi2 := apitest.NewPennsieveDOI()

	expectedCollection := apitest.NewExpectedCollection().WithRandomID().WithNodeID().WithUser(userstest.SeedUser1.ID, pgdb.Owner).
		WithDOIs(doi1, doi2)

	patchCollectionRequest := dto.PatchCollectionRequest{DOIs: &dto.PatchDOIs{Add: []string{doiToAdd1.Value, doiToAdd2.Value}}}

	updateRequest, err := GetUpdateRequest(apitest.PennsieveDOIPrefix, patchCollectionRequest, expectedCollection.ToGetCollectionResponse(t, userstest.SeedUser1.ID))
	require.NoError(t, err)

	assert.Nil(t, updateRequest.Name)
	assert.Nil(t, updateRequest.Description)
	assert.Empty(t, updateRequest.DOIs.Remove)
	assert.Equal(t, []collections.DOI{doiToAdd1, doiToAdd2}, updateRequest.DOIs.Add)

}

// TestHandlePatchCollection tests that run the Handle wrapper around PatchCollection
func TestHandlePatchCollection(t *testing.T) {
	tests := []struct {
		name    string
		tstFunc func(t *testing.T)
	}{
		{
			"return empty datasets and collaborators arrays instead of null",
			testHandlePatchCollectionEmptyArrays,
		},
		{
			"return empty arrays in PublicDatasets instead of nulls",
			testHandlePatchCollectionEmptyArraysInPublicDataset,
		},
		{
			"return Bad Request when given no body",
			testHandlePatchCollectionNoBody,
		},
		{
			"return Bad Request when given empty name",
			testHandlePatchCollectionEmptyName,
		},
		{
			"return Bad Request when given a name that is too long",
			testHandlePatchCollectionNameTooLong,
		},
		{
			"return Bad Request when given a description that is too long",
			testHandlePatchCollectionDescriptionTooLong,
		},
		{
			"return Not Found when given a non-existent collection",
			testHandlePatchCollectionNotFound,
		},
		{
			"forbid updates from users without the proper role on the collection",
			testHandlePatchCollectionAuthz,
		},
		{
			"return Bad Request when given a collection DOI to add",
			testRejectAddingCollectionDOI,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.tstFunc(t)
		})
	}
}

func testHandlePatchCollectionEmptyArrays(t *testing.T) {
	ctx := context.Background()
	callingUser := userstest.SeedUser1

	expectedCollection := apitest.NewExpectedCollection().WithMockID(1).WithNodeID().WithUser(callingUser.ID, pgdb.Owner)

	mockCollectionStore := mocks.NewCollectionsStore().
		WithGetCollectionFunc(expectedCollection.GetCollectionFunc(t)).
		WithUpdateCollectionFunc(expectedCollection.UpdateCollectionFunc(t))

	claims := apitest.DefaultClaims(callingUser)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PatchCollectionRouteKey).
			WithClaims(claims).
			WithBody(t, dto.PatchCollectionRequest{}).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
			Build(),
		Container: apitest.NewTestContainer().WithCollectionsStore(mockCollectionStore),
		Config:    apitest.NewConfigBuilder().WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).Build(),
		Claims:    &claims,
	}
	response, err := Handle(ctx, NewPatchCollectionRouteHandler(), params)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, response.StatusCode)

	assert.NotContains(t, response.Body, `"banners":null`)
	assert.Contains(t, response.Body, `"banners":[]`)

	assert.NotContains(t, response.Body, `"derivedContributors":null`)
	assert.Contains(t, response.Body, `"derivedContributors":[]`)

	assert.NotContains(t, response.Body, `"datasets":null`)
	assert.Contains(t, response.Body, `"datasets":[]`)

}

func testHandlePatchCollectionEmptyArraysInPublicDataset(t *testing.T) {
	ctx := context.Background()
	callingUser := userstest.SeedUser1

	expectedDOI := apitest.NewPennsieveDOI()
	expectedCollection := apitest.NewExpectedCollection().WithMockID(2).WithNodeID().WithUser(callingUser.ID, pgdb.Owner).WithDOIs(expectedDOI)

	mockCollectionStore := mocks.NewCollectionsStore().
		WithGetCollectionFunc(expectedCollection.GetCollectionFunc(t)).
		WithUpdateCollectionFunc(expectedCollection.UpdateCollectionFunc(t))

	mockDiscover := mocks.NewDiscover().WithGetDatasetsByDOIFunc(func(ctx context.Context, dois []string) (service.DatasetsByDOIResponse, error) {
		return service.DatasetsByDOIResponse{Published: map[string]dto.PublicDataset{
			expectedDOI.Value: apitest.NewPublicDataset(expectedDOI.Value, apitest.NewBanner(), apitest.NewPublicContributor()),
		}}, nil
	})
	claims := apitest.DefaultClaims(callingUser)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PatchCollectionRouteKey).
			WithClaims(claims).
			WithBody(t, dto.PatchCollectionRequest{}).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
			Build(),
		Container: apitest.NewTestContainer().WithCollectionsStore(mockCollectionStore).WithDiscover(mockDiscover),
		Config:    apitest.NewConfigBuilder().WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).Build(),
		Claims:    &claims,
	}
	response, err := Handle(ctx, NewPatchCollectionRouteHandler(), params)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, response.StatusCode)

	assert.NotContains(t, response.Body, `"modelCount":null`)
	assert.Contains(t, response.Body, `"modelCount":[]`)

}

func testHandlePatchCollectionNoBody(t *testing.T) {
	ctx := context.Background()
	callingUser := userstest.SeedUser1

	mockCollectionStore := mocks.NewCollectionsStore()

	claims := apitest.DefaultClaims(callingUser)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PatchCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, uuid.NewString()).
			Build(),
		Container: apitest.NewTestContainer().WithCollectionsStore(mockCollectionStore),
		Config:    apitest.NewConfigBuilder().WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).Build(),
		Claims:    &claims,
	}
	response, err := Handle(ctx, NewPatchCollectionRouteHandler(), params)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, response.StatusCode)

	assert.Contains(t, response.Body, "missing request body")

}

func testHandlePatchCollectionEmptyName(t *testing.T) {
	ctx := context.Background()
	callingUser := userstest.SeedUser1

	mockCollectionStore := mocks.NewCollectionsStore()

	claims := apitest.DefaultClaims(callingUser)

	emptyString := ""
	patchRequest := dto.PatchCollectionRequest{
		Name: &emptyString,
	}

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PatchCollectionRouteKey).
			WithClaims(claims).
			WithBody(t, patchRequest).
			WithPathParam(NodeIDPathParamKey, uuid.NewString()).
			Build(),
		Container: apitest.NewTestContainer().WithCollectionsStore(mockCollectionStore),
		Config:    apitest.NewConfigBuilder().WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).Build(),
		Claims:    &claims,
	}
	response, err := Handle(ctx, NewPatchCollectionRouteHandler(), params)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, response.StatusCode)

	assert.Contains(t, response.Body, "collection name cannot be empty")

}

func testHandlePatchCollectionNameTooLong(t *testing.T) {
	ctx := context.Background()
	callingUser := userstest.SeedUser1

	mockCollectionStore := mocks.NewCollectionsStore()

	claims := apitest.DefaultClaims(callingUser)

	tooLongString := strings.Repeat("b", 256)
	patchRequest := dto.PatchCollectionRequest{
		Name: &tooLongString,
	}

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PatchCollectionRouteKey).
			WithClaims(claims).
			WithBody(t, patchRequest).
			WithPathParam(NodeIDPathParamKey, uuid.NewString()).
			Build(),
		Container: apitest.NewTestContainer().WithCollectionsStore(mockCollectionStore),
		Config:    apitest.NewConfigBuilder().WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).Build(),
		Claims:    &claims,
	}
	response, err := Handle(ctx, NewPatchCollectionRouteHandler(), params)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, response.StatusCode)

	assert.Contains(t, response.Body, "collection name cannot have more than 255 characters")

}

func testHandlePatchCollectionDescriptionTooLong(t *testing.T) {
	ctx := context.Background()
	callingUser := userstest.SeedUser1

	mockCollectionStore := mocks.NewCollectionsStore()

	claims := apitest.DefaultClaims(callingUser)

	tooLongString := strings.Repeat("b", 256)
	patchRequest := dto.PatchCollectionRequest{
		Description: &tooLongString,
	}

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PatchCollectionRouteKey).
			WithClaims(claims).
			WithBody(t, patchRequest).
			WithPathParam(NodeIDPathParamKey, uuid.NewString()).
			Build(),
		Container: apitest.NewTestContainer().WithCollectionsStore(mockCollectionStore),
		Config:    apitest.NewConfigBuilder().WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).Build(),
		Claims:    &claims,
	}
	response, err := Handle(ctx, NewPatchCollectionRouteHandler(), params)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, response.StatusCode)

	assert.Contains(t, response.Body, "collection description cannot have more than 255 characters")

}

func testHandlePatchCollectionNotFound(t *testing.T) {
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
		Request: apitest.NewAPIGatewayRequestBuilder(PatchCollectionRouteKey).
			WithClaims(claims).
			WithBody(t, dto.PatchCollectionRequest{}).
			WithPathParam(NodeIDPathParamKey, nonExistentNodeID).
			Build(),
		Container: apitest.NewTestContainer().WithCollectionsStore(mockCollectionStore),
		Config:    apitest.NewConfigBuilder().WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).Build(),
		Claims:    &claims,
	}
	response, err := Handle(ctx, NewPatchCollectionRouteHandler(), params)
	require.NoError(t, err)

	assert.Equal(t, http.StatusNotFound, response.StatusCode)

	assert.Contains(t, response.Body, "not found")
	assert.Contains(t, response.Body, nonExistentNodeID)

}

func testHandlePatchCollectionAuthz(t *testing.T) {
	ctx := context.Background()
	callingUser := userstest.SeedUser1
	claims := apitest.DefaultClaims(callingUser)

	for _, tooLowPerm := range []pgdb.DbPermission{pgdb.Guest, pgdb.Read} {
		t.Run(tooLowPerm.String(), func(t *testing.T) {
			expectedCollection := apitest.NewExpectedCollection().WithRandomID().WithNodeID().WithUser(callingUser.ID, tooLowPerm)

			mockCollectionStore := mocks.NewCollectionsStore().
				WithGetCollectionFunc(expectedCollection.GetCollectionFunc(t))

			params := Params{
				Request: apitest.NewAPIGatewayRequestBuilder(PatchCollectionRouteKey).
					WithClaims(claims).
					WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
					WithBody(t, dto.PatchCollectionRequest{}).
					Build(),
				Container: apitest.NewTestContainer().WithCollectionsStore(mockCollectionStore),
				Config:    apitest.NewConfigBuilder().WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).Build(),
				Claims:    &claims,
			}

			resp, err := Handle(ctx, NewPatchCollectionRouteHandler(), params)
			require.NoError(t, err)

			assert.Equal(t, http.StatusForbidden, resp.StatusCode)
			assert.Equal(t, DefaultErrorResponseHeaders(), resp.Headers)
			assert.Contains(t, resp.Body, "errorId")
			assert.Contains(t, resp.Body, "message")
		})
	}

	// pgdb.Write & pgdb.Delete => role.Editor, which we take to mean both have perm to add and delete DOIs
	for _, okPerm := range []pgdb.DbPermission{pgdb.Write, pgdb.Delete, pgdb.Administer, pgdb.Owner} {
		t.Run(okPerm.String(), func(t *testing.T) {
			expectedCollection := apitest.NewExpectedCollection().WithRandomID().WithNodeID().WithUser(callingUser.ID, okPerm)

			mockCollectionStore := mocks.NewCollectionsStore().
				WithGetCollectionFunc(expectedCollection.GetCollectionFunc(t)).
				WithUpdateCollectionFunc(expectedCollection.UpdateCollectionFunc(t))

			params := Params{
				Request: apitest.NewAPIGatewayRequestBuilder(PatchCollectionRouteKey).
					WithClaims(claims).
					WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
					WithBody(t, dto.PatchCollectionRequest{}).
					Build(),
				Container: apitest.NewTestContainer().WithCollectionsStore(mockCollectionStore),
				Config:    apitest.NewConfigBuilder().WithPennsieveConfig(apitest.PennsieveConfigWithFakeURL()).Build(),
				Claims:    &claims,
			}

			resp, err := Handle(ctx, NewPatchCollectionRouteHandler(), params)
			require.NoError(t, err)

			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})
	}

}

func testRejectAddingCollectionDOI(t *testing.T) {
	ctx := context.Background()

	callingUser := userstest.SeedUser1

	expectedCollection := apitest.NewExpectedCollection().
		WithRandomID().
		WithNodeID().
		WithUser(callingUser.ID, pgdb.Owner)

	mockCollectionStore := mocks.NewCollectionsStore().WithGetCollectionFunc(expectedCollection.GetCollectionFunc(t))

	expectedDatasets := apitest.NewExpectedPennsieveDatasets()
	researchDataset := expectedDatasets.NewPublishedWithOptions(apitest.WithDatasetType("research"))
	releaseDataset := expectedDatasets.NewPublishedWithOptions(apitest.WithDatasetType("release"))
	collectionDataset := expectedDatasets.NewPublishedWithOptions(apitest.WithDatasetType(dto.CollectionDatasetType))

	patchCollectionRequest := dto.PatchCollectionRequest{
		DOIs: &dto.PatchDOIs{Add: []string{researchDataset.DOI, releaseDataset.DOI, collectionDataset.DOI}},
	}

	claims := apitest.DefaultClaims(callingUser)

	mockDiscoverServer := httptest.NewServer(mocks.ToDiscoverHandlerFunc(ctx, t, expectedDatasets.GetDatasetsByDOIFunc(t)))
	defer mockDiscoverServer.Close()

	config := apitest.NewConfigBuilder().
		WithPennsieveConfig(apitest.PennsieveConfig(mockDiscoverServer.URL)).
		Build()

	container := apitest.NewTestContainer().
		WithCollectionsStore(mockCollectionStore).
		WithHTTPTestDiscover(mockDiscoverServer.URL)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(PatchCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
			WithBody(t, patchCollectionRequest).
			Build(),
		Container: container,
		Config:    config,
		Claims:    &claims,
	}

	response, err := Handle(ctx, NewPatchCollectionRouteHandler(), params)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, response.StatusCode)

	assert.Contains(t, response.Body, collectionDataset.DOI)
}
