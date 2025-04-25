package routes

import (
	"context"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/collections-service/internal/api/store"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/pennsieve/collections-service/internal/test/configtest"
	"github.com/pennsieve/collections-service/internal/test/fixtures"
	"github.com/pennsieve/collections-service/internal/test/mocks"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUpdateCollection(t *testing.T) {
	tests := []struct {
		scenario string
		tstFunc  func(t *testing.T, expectationDB *fixtures.ExpectationDB)
	}{
		{"update collection name", testUpdateCollectionName},
		{"update collection description", testUpdateCollectionDescription},
		{"update collection name and description", testUpdateCollectionNameAndDescription},
		{"remove DOI from collection", testUpdateCollectionRemoveDOI},
		{"remove DOIs from collection", testUpdateCollectionRemoveDOIs},
		{"add DOI to collection", testUpdateCollectionAddDOI},
		{"add DOIs to collection", testUpdateCollectionAddDOIs},
		{"update collection", testUpdateCollection},
		{"update asking to remove a non-existent DOI should succeed", testUpdateCollectionRemoveNonExistentDOI},
		{"update asking to add an already existing DOI should succeed", testUpdateCollectionAddExistingDOI},
		{"update non-existent collection should return ErrCollectionNotFound", testUpdateCollectionNonExistent},
		{"update DOIs on non-existent collection should return ErrCollectionNotFound", testUpdateCollectionNonExistentDOIUpdateOnly},
	}

	ctx := context.Background()
	postgresDBConfig := configtest.PostgresDBConfig()

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

func testUpdateCollectionName(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	expectedDatasets := apitest.NewExpectedPennsieveDatasets()

	user := apitest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	expectedCollection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user.ID, pgdb.Owner).WithDOIs(expectedDatasets.NewPublished(apitest.NewPublicContributor()).DOI)
	createResp := expectationDB.CreateCollection(ctx, t, expectedCollection)
	collectionID := createResp.ID

	newName := uuid.NewString()
	update := dto.PatchCollectionRequest{Name: &newName}

	mockDiscoverServer := httptest.NewServer(mocks.ToDiscoverHandlerFunc(t, expectedDatasets.GetDatasetsByDOIFunc(t)))
	defer mockDiscoverServer.Close()

	claims := apitest.DefaultClaims(user)

	apiConfig := apitest.NewConfigBuilder().
		WithDockerPostgresDBConfig().
		WithPennsieveConfig(apitest.PennsieveConfig(mockDiscoverServer.URL)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
		WithContainerStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase).
		WithHTTPTestDiscover(mockDiscoverServer.URL)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(UpdateCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
			WithBody(t, update).
			Build(),
		Container: container,
		Config:    apiConfig,
		Claims:    &claims,
	}

	updatedCollection, err := UpdateCollection(ctx, params)
	require.NoError(t, err)

	expectedCollection.Name = newName
	assertEqualExpectedGetCollectionResponse(t, expectedCollection, updatedCollection, expectedDatasets)

	expectationDB.RequireCollection(ctx, t, expectedCollection, collectionID)
}

func testUpdateCollectionDescription(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	expectedDatasets := apitest.NewExpectedPennsieveDatasets()

	user := apitest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	expectedCollection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user.ID, pgdb.Owner).
		WithDOIs(
			expectedDatasets.NewPublished(apitest.NewPublicContributor()).DOI,
			expectedDatasets.NewPublished(apitest.NewPublicContributor(apitest.WithOrcid())).DOI,
		)
	createResp := expectationDB.CreateCollection(ctx, t, expectedCollection)
	collectionID := createResp.ID

	newDescription := uuid.NewString()
	update := dto.PatchCollectionRequest{Description: &newDescription}

	mockDiscoverServer := httptest.NewServer(mocks.ToDiscoverHandlerFunc(t, expectedDatasets.GetDatasetsByDOIFunc(t)))
	defer mockDiscoverServer.Close()

	claims := apitest.DefaultClaims(user)

	apiConfig := apitest.NewConfigBuilder().
		WithDockerPostgresDBConfig().
		WithPennsieveConfig(apitest.PennsieveConfig(mockDiscoverServer.URL)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
		WithContainerStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase).
		WithHTTPTestDiscover(mockDiscoverServer.URL)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(UpdateCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
			WithBody(t, update).
			Build(),
		Container: container,
		Config:    apiConfig,
		Claims:    &claims,
	}

	updatedCollection, err := UpdateCollection(ctx, params)
	require.NoError(t, err)

	expectedCollection.Description = newDescription
	assertEqualExpectedGetCollectionResponse(t, expectedCollection, updatedCollection, expectedDatasets)

	expectationDB.RequireCollection(ctx, t, expectedCollection, collectionID)
}

func testUpdateCollectionNameAndDescription(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	expectedDatasets := apitest.NewExpectedPennsieveDatasets()

	user := apitest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	expectedCollection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user.ID, pgdb.Owner).
		WithDOIs(
			expectedDatasets.NewPublished(apitest.NewPublicContributor()).DOI,
			expectedDatasets.NewPublished(apitest.NewPublicContributor(apitest.WithOrcid())).DOI,
		)
	createResp := expectationDB.CreateCollection(ctx, t, expectedCollection)
	collectionID := createResp.ID

	newName := uuid.NewString()
	newDescription := uuid.NewString()
	update := dto.PatchCollectionRequest{Name: &newName, Description: &newDescription}

	mockDiscoverServer := httptest.NewServer(mocks.ToDiscoverHandlerFunc(t, expectedDatasets.GetDatasetsByDOIFunc(t)))
	defer mockDiscoverServer.Close()

	claims := apitest.DefaultClaims(user)

	apiConfig := apitest.NewConfigBuilder().
		WithDockerPostgresDBConfig().
		WithPennsieveConfig(apitest.PennsieveConfig(mockDiscoverServer.URL)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
		WithContainerStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase).
		WithHTTPTestDiscover(mockDiscoverServer.URL)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(UpdateCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
			WithBody(t, update).
			Build(),
		Container: container,
		Config:    apiConfig,
		Claims:    &claims,
	}

	updatedCollection, err := UpdateCollection(ctx, params)
	require.NoError(t, err)

	expectedCollection.Name = newName
	expectedCollection.Description = newDescription
	assertEqualExpectedGetCollectionResponse(t, expectedCollection, updatedCollection, expectedDatasets)

	expectationDB.RequireCollection(ctx, t, expectedCollection, collectionID)
}

func testUpdateCollectionRemoveDOI(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	expectedDatasets := apitest.NewExpectedPennsieveDatasets()

	user := apitest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	doiToKeep1 := expectedDatasets.NewPublished(apitest.NewPublicContributor(apitest.WithMiddleInitial())).DOI
	doiToRemove := expectedDatasets.NewPublished(apitest.NewPublicContributor(apitest.WithOrcid())).DOI
	doiToKeep2 := expectedDatasets.NewPublished(apitest.NewPublicContributor()).DOI

	expectedCollection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user.ID, pgdb.Owner).
		WithDOIs(doiToKeep1, doiToRemove, doiToKeep2)
	createResp := expectationDB.CreateCollection(ctx, t, expectedCollection)
	collectionID := createResp.ID

	update := dto.PatchCollectionRequest{DOIs: &dto.PatchDOIs{Remove: []string{doiToRemove}}}

	mockDiscoverServer := httptest.NewServer(mocks.ToDiscoverHandlerFunc(t, expectedDatasets.GetDatasetsByDOIFunc(t)))
	defer mockDiscoverServer.Close()

	claims := apitest.DefaultClaims(user)

	apiConfig := apitest.NewConfigBuilder().
		WithDockerPostgresDBConfig().
		WithPennsieveConfig(apitest.PennsieveConfig(mockDiscoverServer.URL)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, apiConfig.PostgresDB)).
		WithContainerStoreFromPostgresDB(apiConfig.PostgresDB.CollectionsDatabase).
		WithHTTPTestDiscover(mockDiscoverServer.URL)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(UpdateCollectionRouteKey).
			WithClaims(claims).
			WithPathParam(NodeIDPathParamKey, *expectedCollection.NodeID).
			WithBody(t, update).
			Build(),
		Container: container,
		Config:    apiConfig,
		Claims:    &claims,
	}

	updatedCollection, err := UpdateCollection(ctx, params)
	require.NoError(t, err)

	expectedCollection.SetDOIs(doiToKeep1, doiToKeep2)
	assertEqualExpectedGetCollectionResponse(t, expectedCollection, updatedCollection, expectedDatasets)

	expectationDB.RequireCollection(ctx, t, expectedCollection, collectionID)
}

func testUpdateCollectionRemoveDOIs(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	t.Skip("implement me")
}

func testUpdateCollectionAddDOI(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	t.Skip("implement me")

}

func testUpdateCollectionAddDOIs(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	t.Skip("implement me")

}

func testUpdateCollection(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	t.Skip("implement me")

}

func testUpdateCollectionRemoveNonExistentDOI(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	t.Skip("implement me")

}

func testUpdateCollectionAddExistingDOI(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	t.Skip("implement me")

}

func testUpdateCollectionNonExistent(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	t.Skip("implement me")

}

func testUpdateCollectionNonExistentDOIUpdateOnly(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	t.Skip("implement me")

}

// TestHandleUpdateCollection tests that run the Handle wrapper around UpdateCollection
func TestHandleUpdateCollection(t *testing.T) {
	tests := []struct {
		name    string
		tstFunc func(t *testing.T)
	}{
		{
			"return empty datasets and collaborators arrays instead of null",
			testHandleUpdateCollectionEmptyArrays,
		},
		{
			"return empty arrays in PublicDatasets instead of nulls",
			testHandleUpdateCollectionEmptyArraysInPublicDataset,
		},
		{
			"return Bad Request when given no body",
			testHandleUpdateCollectionNoBody,
		},
		{
			"return Bad Request when given empty name",
			testHandleUpdateCollectionEmptyName,
		},
		{
			"return Bad Request when given a name that is too long",
			testHandleUpdateCollectionNameTooLong,
		},
		{
			"return Bad Request when given a description that is too long",
			testHandleUpdateCollectionDescriptionTooLong,
		},
		{
			"return Not Found when given a non-existent collection",
			testHandleUpdateCollectionNotFound,
		},
		{
			"forbid updates from users without the proper role on the collection",
			testUpdateCollectionAuthz,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.tstFunc(t)
		})
	}
}

func testHandleUpdateCollectionEmptyArrays(t *testing.T) {
	ctx := context.Background()
	callingUser := apitest.SeedUser1

	expectedCollection := apitest.NewExpectedCollection().WithMockID(1).WithNodeID().WithUser(callingUser.ID, pgdb.Owner)

	mockCollectionStore := mocks.NewMockCollectionsStore().
		WithGetCollectionFunc(expectedCollection.GetCollectionFunc(t)).
		WithUpdateCollectionFunc(expectedCollection.UpdateCollectionFunc(t))

	claims := apitest.DefaultClaims(callingUser)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(UpdateCollectionRouteKey).
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

func testHandleUpdateCollectionEmptyArraysInPublicDataset(t *testing.T) {
	ctx := context.Background()
	callingUser := apitest.SeedUser1

	expectedDOI := apitest.NewPennsieveDOI()
	expectedCollection := apitest.NewExpectedCollection().WithMockID(2).WithNodeID().WithUser(callingUser.ID, pgdb.Owner).WithDOIs(expectedDOI)

	mockCollectionStore := mocks.NewMockCollectionsStore().
		WithGetCollectionFunc(expectedCollection.GetCollectionFunc(t)).
		WithUpdateCollectionFunc(expectedCollection.UpdateCollectionFunc(t))

	mockDiscover := mocks.NewMockDiscover().WithGetDatasetsByDOIFunc(func(dois []string) (service.DatasetsByDOIResponse, error) {
		return service.DatasetsByDOIResponse{Published: map[string]dto.PublicDataset{
			expectedDOI: apitest.NewPublicDataset(expectedDOI, apitest.NewBanner(), apitest.NewPublicContributor()),
		}}, nil
	})
	claims := apitest.DefaultClaims(callingUser)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(UpdateCollectionRouteKey).
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

func testHandleUpdateCollectionNoBody(t *testing.T) {
	ctx := context.Background()
	callingUser := apitest.SeedUser1

	mockCollectionStore := mocks.NewMockCollectionsStore()

	claims := apitest.DefaultClaims(callingUser)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(UpdateCollectionRouteKey).
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

func testHandleUpdateCollectionEmptyName(t *testing.T) {
	ctx := context.Background()
	callingUser := apitest.SeedUser1

	mockCollectionStore := mocks.NewMockCollectionsStore()

	claims := apitest.DefaultClaims(callingUser)

	emptyString := ""
	patchRequest := dto.PatchCollectionRequest{
		Name: &emptyString,
	}

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(UpdateCollectionRouteKey).
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

func testHandleUpdateCollectionNameTooLong(t *testing.T) {
	ctx := context.Background()
	callingUser := apitest.SeedUser1

	mockCollectionStore := mocks.NewMockCollectionsStore()

	claims := apitest.DefaultClaims(callingUser)

	tooLongString := strings.Repeat("b", 256)
	patchRequest := dto.PatchCollectionRequest{
		Name: &tooLongString,
	}

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(UpdateCollectionRouteKey).
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

func testHandleUpdateCollectionDescriptionTooLong(t *testing.T) {
	ctx := context.Background()
	callingUser := apitest.SeedUser1

	mockCollectionStore := mocks.NewMockCollectionsStore()

	claims := apitest.DefaultClaims(callingUser)

	tooLongString := strings.Repeat("b", 256)
	patchRequest := dto.PatchCollectionRequest{
		Description: &tooLongString,
	}

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(UpdateCollectionRouteKey).
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

func testHandleUpdateCollectionNotFound(t *testing.T) {
	ctx := context.Background()
	callingUser := apitest.SeedUser1
	nonExistentNodeID := uuid.NewString()

	mockCollectionStore := mocks.NewMockCollectionsStore().WithGetCollectionFunc(func(ctx context.Context, userID int64, nodeID string) (store.GetCollectionResponse, error) {
		test.Helper(t)
		require.Equal(t, callingUser.ID, userID)
		require.Equal(t, nonExistentNodeID, nodeID)
		return store.GetCollectionResponse{}, store.ErrCollectionNotFound
	})

	claims := apitest.DefaultClaims(callingUser)

	params := Params{
		Request: apitest.NewAPIGatewayRequestBuilder(UpdateCollectionRouteKey).
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

func testUpdateCollectionAuthz(t *testing.T) {
	ctx := context.Background()
	callingUser := apitest.SeedUser1
	claims := apitest.DefaultClaims(callingUser)

	for _, tooLowPerm := range []pgdb.DbPermission{pgdb.Guest, pgdb.Read} {
		t.Run(tooLowPerm.String(), func(t *testing.T) {
			expectedCollection := apitest.NewExpectedCollection().WithNodeID().WithUser(callingUser.ID, tooLowPerm)

			mockCollectionStore := mocks.NewMockCollectionsStore().
				WithGetCollectionFunc(expectedCollection.GetCollectionFunc(t))

			params := Params{
				Request: apitest.NewAPIGatewayRequestBuilder(DeleteCollectionRouteKey).
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
			// we're not saving this to a real DB, so no ID is generated for us
			mockCollectionID := int64(123)
			expectedCollection := apitest.NewExpectedCollection().WithMockID(mockCollectionID).WithNodeID().WithUser(callingUser.ID, okPerm)

			mockCollectionStore := mocks.NewMockCollectionsStore().
				WithGetCollectionFunc(expectedCollection.GetCollectionFunc(t)).
				WithUpdateCollectionFunc(expectedCollection.UpdateCollectionFunc(t))

			params := Params{
				Request: apitest.NewAPIGatewayRequestBuilder(DeleteCollectionRouteKey).
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
