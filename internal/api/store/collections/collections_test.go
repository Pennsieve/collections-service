package collections_test

import (
	"context"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/publishing"
	"github.com/pennsieve/collections-service/internal/api/store/collections"
	"github.com/pennsieve/collections-service/internal/shared/logging"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/pennsieve/collections-service/internal/test/apitest/builders/stores/collectionstest"
	"github.com/pennsieve/collections-service/internal/test/fixtures"
	"github.com/pennsieve/collections-service/internal/test/userstest"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestStore(t *testing.T) {
	ctx := context.Background()
	config := test.PostgresDBConfig(t)

	for _, tt := range []struct {
		scenario string
		tstFunc  func(t *testing.T, collectionsStore *collections.PostgresStore, expectationDB *fixtures.ExpectationDB)
	}{
		{"create collection, nil DOIs", testCreateCollectionNilDOIs},
		{"create collection, empty DOIs", testCreateCollectionEmptyDOIs},
		{"create collection, one DOI", testCreateCollectionOneDOI},
		{"create collection, many DOIs", testCreateCollectionManyDOIs},
		{"create collection, empty description", testCreateCollectionEmptyDescription},
		{"get collections, none", testGetCollectionsNone},
		{"get collections", testGetCollections},
		{"get collections, user with no permission on the collection should not see it", testGetCollectionsNoPerms},
		{"get collections, limit and offset", testGetCollectionsLimitOffset},
		{"get collection, none", testGetCollectionNone},
		{"get collection", testGetCollection},
		{"get collection should return publish status if one exists", testGetCollectionPublishStatus},
		{"get collection, user with no permission on the collection should not see it", testGetCollectionNoPerms},

		{"delete collection", testDeleteCollection},
		{"delete non-existent collection", testDeleteCollectionNonExistent},
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
		{"StartPublish should mark the start of a publish event", testStartPublish},
		{"StartPublish should return InProgress error and do no updates if publish already in progress", testStartPublishExistingInProgress},
		{"StartPublish should update an existing complete publish status", testStartPublishExistingComplete},
		{"StartPublish should update an existing failed publish status", testStartPublishExistingFailed},
		{"FinishPublish should update the publish status of a collection", testFinishPublish},
		{"FinishPublish should return an error if no publish status exists", testFinishPublishNoExistingStatus},
	} {

		t.Run(tt.scenario, func(t *testing.T) {
			db := test.NewPostgresDBFromConfig(t, config)
			expectationDB := fixtures.NewExpectationDB(db, config.CollectionsDatabase)
			t.Cleanup(func() {
				expectationDB.CleanUp(ctx, t)
			})

			collectionsStore := collections.NewPostgresStore(db, config.CollectionsDatabase, logging.Default)

			tt.tstFunc(t, collectionsStore, expectationDB)
		})
	}
}

func testCreateCollectionNilDOIs(t *testing.T, store *collections.PostgresStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()
	expectedOwner := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, expectedOwner)
	expectedCollection := apitest.NewExpectedCollection().WithNodeID().WithUser(*expectedOwner.ID, pgdb.Owner)

	resp, err := store.CreateCollection(ctx, *expectedOwner.ID, *expectedCollection.NodeID, expectedCollection.Name, expectedCollection.Description, nil)
	require.NoError(t, err)
	assert.Positive(t, resp.ID)
	assert.Equal(t, role.Owner, resp.CreatorRole)

	expectationDB.RequireCollection(ctx, t, expectedCollection, resp.ID)
}

func testCreateCollectionEmptyDOIs(t *testing.T, collectionsStore *collections.PostgresStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()
	expectedOwner := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, expectedOwner)

	expectedCollection := apitest.NewExpectedCollection().WithNodeID().WithUser(*expectedOwner.ID, pgdb.Owner)

	resp, err := collectionsStore.CreateCollection(ctx, *expectedOwner.ID, *expectedCollection.NodeID, expectedCollection.Name, expectedCollection.Description, []collections.DOI{})
	require.NoError(t, err)
	assert.Positive(t, resp.ID)
	assert.Equal(t, role.Owner, resp.CreatorRole)

	expectationDB.RequireCollection(ctx, t, expectedCollection, resp.ID)
}

func testCreateCollectionOneDOI(t *testing.T, store *collections.PostgresStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()
	expectedOwner := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, expectedOwner)
	expectedCollection := apitest.NewExpectedCollection().WithNodeID().WithUser(*expectedOwner.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI())

	resp, err := store.CreateCollection(ctx, *expectedOwner.ID, *expectedCollection.NodeID, expectedCollection.Name, expectedCollection.Description, expectedCollection.DOIs.AsDOIs())
	require.NoError(t, err)
	assert.Positive(t, resp.ID)
	assert.Equal(t, role.Owner, resp.CreatorRole)

	expectationDB.RequireCollection(ctx, t, expectedCollection, resp.ID)

}

func testCreateCollectionManyDOIs(t *testing.T, store *collections.PostgresStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()
	expectedOwner := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, expectedOwner)
	expectedCollection := apitest.NewExpectedCollection().WithNodeID().WithUser(*expectedOwner.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI())

	resp, err := store.CreateCollection(ctx, *expectedOwner.ID, *expectedCollection.NodeID, expectedCollection.Name, expectedCollection.Description, expectedCollection.DOIs.AsDOIs())
	require.NoError(t, err)
	assert.Positive(t, resp.ID)
	assert.Equal(t, role.Owner, resp.CreatorRole)

	expectationDB.RequireCollection(ctx, t, expectedCollection, resp.ID)

}

func testCreateCollectionEmptyDescription(t *testing.T, store *collections.PostgresStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()
	expectedOwner := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, expectedOwner)
	expectedCollection := apitest.NewExpectedCollection().WithNodeID().WithUser(*expectedOwner.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI())
	expectedCollection.Description = ""

	resp, err := store.CreateCollection(ctx, *expectedOwner.ID, *expectedCollection.NodeID, expectedCollection.Name, expectedCollection.Description, expectedCollection.DOIs.AsDOIs())
	require.NoError(t, err)
	assert.Positive(t, resp.ID)
	assert.Equal(t, role.Owner, resp.CreatorRole)

	expectationDB.RequireCollection(ctx, t, expectedCollection, resp.ID)

}

func testGetCollectionsNone(t *testing.T, store *collections.PostgresStore, _ *fixtures.ExpectationDB) {
	ctx := context.Background()

	// Test with store
	limit, offset := 10, 0
	// use a user with no collections
	response, err := store.GetCollections(ctx, userstest.SeedUser1.ID, limit, offset)
	require.NoError(t, err)

	assert.Equal(t, limit, response.Limit)
	assert.Equal(t, offset, response.Offset)
	assert.Equal(t, 0, response.TotalCount)

	assert.Len(t, response.Collections, 0)
}

func testGetCollections(t *testing.T, store *collections.PostgresStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	// Set up using the ExpectationDB
	user1 := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user1)
	user2 := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user2)

	user1CollectionNoDOI := apitest.NewExpectedCollection().WithNodeID().WithUser(*user1.ID, pgdb.Owner)
	expectationDB.CreateCollection(ctx, t, user1CollectionNoDOI)
	user1CollectionOneDOI := apitest.NewExpectedCollection().WithNodeID().WithUser(*user1.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI())
	expectationDB.CreateCollection(ctx, t, user1CollectionOneDOI)
	user1CollectionFiveDOI := apitest.NewExpectedCollection().WithNodeID().WithUser(*user1.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI())
	expectationDB.CreateCollection(ctx, t, user1CollectionFiveDOI)

	user2Collection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user2.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI())
	expectationDB.CreateCollection(ctx, t, user2Collection)

	// Test with store
	limit, offset := 10, 0
	response, err := store.GetCollections(ctx, *user1.ID, limit, offset)
	require.NoError(t, err)

	assert.Equal(t, limit, response.Limit)
	assert.Equal(t, offset, response.Offset)
	assert.Equal(t, 3, response.TotalCount)

	assert.Len(t, response.Collections, 3)

	// They should be returned in oldest first order
	actualCollection1 := response.Collections[0]
	assertExpectedEqualCollectionSummary(t, user1CollectionNoDOI, actualCollection1)

	actualCollection2 := response.Collections[1]
	assertExpectedEqualCollectionSummary(t, user1CollectionOneDOI, actualCollection2)

	actualCollection3 := response.Collections[2]
	assertExpectedEqualCollectionSummary(t, user1CollectionFiveDOI, actualCollection3)

	// try user2's collections
	user2CollectionResp, err := store.GetCollections(ctx, *user2.ID, limit, offset)
	require.NoError(t, err)

	assert.Equal(t, limit, user2CollectionResp.Limit)
	assert.Equal(t, offset, user2CollectionResp.Offset)
	assert.Equal(t, 1, user2CollectionResp.TotalCount)
	assert.Len(t, user2CollectionResp.Collections, 1)

	actualUser2Collection := user2CollectionResp.Collections[0]
	assertExpectedEqualCollectionSummary(t, user2Collection, actualUser2Collection)

}

func testGetCollectionsNoPerms(t *testing.T, store *collections.PostgresStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	// We will probably only end up with collection users that have pgdb.NoPermission if there is a bug,
	// but it is a possibility and we coded against it, so here is a test

	// Set up using the ExpectationDB
	user1 := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user1)
	user2 := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user2)

	user1CollectionNoDOI := apitest.NewExpectedCollection().WithNodeID().
		WithUser(*user1.ID, pgdb.Owner)
	expectationDB.CreateCollection(ctx, t, user1CollectionNoDOI)

	user1CollectionOneDOI := apitest.NewExpectedCollection().WithNodeID().
		WithUser(*user1.ID, pgdb.Owner).
		WithDOIs(apitest.NewPennsieveDOI())
	expectationDB.CreateCollection(ctx, t, user1CollectionOneDOI)

	user1CollectionFiveDOI := apitest.NewExpectedCollection().WithNodeID().
		WithUser(*user1.ID, pgdb.Owner).
		WithUser(*user2.ID, pgdb.NoPermission).
		WithDOIs(apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI())
	expectationDB.CreateCollection(ctx, t, user1CollectionFiveDOI)

	user2Collection := apitest.NewExpectedCollection().WithNodeID().
		WithUser(*user2.ID, pgdb.Owner).
		WithUser(*user1.ID, pgdb.NoPermission).
		WithDOIs(apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI())
	expectationDB.CreateCollection(ctx, t, user2Collection)

	// Test with store
	limit, offset := 10, 0
	response, err := store.GetCollections(ctx, *user1.ID, limit, offset)
	require.NoError(t, err)

	assert.Equal(t, limit, response.Limit)
	assert.Equal(t, offset, response.Offset)
	assert.Equal(t, 3, response.TotalCount)

	assert.Len(t, response.Collections, 3)

	// They should be returned in oldest first order
	actualCollection1 := response.Collections[0]
	assertExpectedEqualCollectionSummary(t, user1CollectionNoDOI, actualCollection1)

	actualCollection2 := response.Collections[1]
	assertExpectedEqualCollectionSummary(t, user1CollectionOneDOI, actualCollection2)

	actualCollection3 := response.Collections[2]
	assertExpectedEqualCollectionSummary(t, user1CollectionFiveDOI, actualCollection3)

	// try user2's collections
	user2CollectionResp, err := store.GetCollections(ctx, *user2.ID, limit, offset)
	require.NoError(t, err)

	assert.Equal(t, limit, user2CollectionResp.Limit)
	assert.Equal(t, offset, user2CollectionResp.Offset)
	assert.Equal(t, 1, user2CollectionResp.TotalCount)
	assert.Len(t, user2CollectionResp.Collections, 1)

	actualUser2Collection := user2CollectionResp.Collections[0]
	assertExpectedEqualCollectionSummary(t, user2Collection, actualUser2Collection)

}

func testGetCollectionsLimitOffset(t *testing.T, store *collections.PostgresStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	user1 := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user1)

	totalCollections := 11
	var expectedCollections []*apitest.ExpectedCollection
	for i := 0; i < totalCollections; i++ {
		expectedCollection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user1.ID, pgdb.Owner).WithNPennsieveDOIs(i)
		expectationDB.CreateCollection(ctx, t, expectedCollection)
		expectedCollections = append(expectedCollections, expectedCollection)
	}

	limit := 3
	// offsets:        0 3 6 9 12
	// response sizes: 3 3 3 2  0
	offset := 0

	for ; offset < totalCollections; offset += limit {
		resp, err := store.GetCollections(ctx, *user1.ID, limit, offset)
		require.NoError(t, err)

		assert.Equal(t, limit, resp.Limit)
		assert.Equal(t, offset, resp.Offset)
		assert.Equal(t, totalCollections, resp.TotalCount)

		expectedCollectionLen := min(limit, totalCollections-offset)
		if assert.Len(t, resp.Collections, expectedCollectionLen) {
			for i := 0; i < expectedCollectionLen; i++ {
				assertExpectedEqualCollectionSummary(t, expectedCollections[offset+i], resp.Collections[i])
			}
		}
	}

	// now offset >= totalCollections, so the response should have no collections
	// but still have the correct TotalCount.

	emptyResp, err := store.GetCollections(ctx, *user1.ID, limit, offset)
	require.NoError(t, err)

	assert.Equal(t, limit, emptyResp.Limit)
	assert.Equal(t, offset, emptyResp.Offset)
	assert.Equal(t, totalCollections, emptyResp.TotalCount)
	assert.Empty(t, emptyResp.Collections)

}

func testGetCollectionNone(t *testing.T, collectionsStore *collections.PostgresStore, _ *fixtures.ExpectationDB) {
	ctx := context.Background()

	// Test with collectionsStore
	// use a user with no collections
	_, err := collectionsStore.GetCollection(ctx, userstest.SeedUser1.ID, uuid.NewString())
	require.ErrorIs(t, err, collections.ErrCollectionNotFound)
}

func testGetCollection(t *testing.T, store *collections.PostgresStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	// Set up using the ExpectationDB
	user1 := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user1)
	user2 := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user2)

	user1CollectionNoDOI := apitest.NewExpectedCollection().WithNodeID().WithUser(*user1.ID, pgdb.Owner)
	expectationDB.CreateCollection(ctx, t, user1CollectionNoDOI)
	user1CollectionOneDOI := apitest.NewExpectedCollection().WithNodeID().WithUser(*user1.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI())
	expectationDB.CreateCollection(ctx, t, user1CollectionOneDOI)
	user1CollectionFiveDOI := apitest.NewExpectedCollection().WithNodeID().WithUser(*user1.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI())
	expectationDB.CreateCollection(ctx, t, user1CollectionFiveDOI)

	user2Collection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user2.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI())
	expectationDB.CreateCollection(ctx, t, user2Collection)

	// Test with store
	// user1NoDOIs
	user1NoDOIResp, err := store.GetCollection(ctx, *user1.ID, *user1CollectionNoDOI.NodeID)
	require.NoError(t, err)
	assert.NotNil(t, user1NoDOIResp)
	assertExpectedEqualCollectionBase(t, user1CollectionNoDOI, user1NoDOIResp.CollectionBase)
	assert.Empty(t, user1NoDOIResp.DOIs)

	// user1OneDOI
	user1OneDOIResp, err := store.GetCollection(ctx, *user1.ID, *user1CollectionOneDOI.NodeID)
	assert.NoError(t, err)
	assert.NotNil(t, user1CollectionOneDOI)
	assertExpectedEqualCollectionBase(t, user1CollectionOneDOI, user1OneDOIResp.CollectionBase)
	assert.Equal(t, user1CollectionOneDOI.DOIs.AsDOIs(), user1OneDOIResp.DOIs)

	// user1FiveDOI
	user1FiveDOIResp, err := store.GetCollection(ctx, *user1.ID, *user1CollectionFiveDOI.NodeID)
	assert.NoError(t, err)
	assert.NotNil(t, user1CollectionFiveDOI)
	assertExpectedEqualCollectionBase(t, user1CollectionFiveDOI, user1FiveDOIResp.CollectionBase)
	assert.Equal(t, user1CollectionFiveDOI.DOIs.AsDOIs(), user1FiveDOIResp.DOIs)

	// try user2's collections
	user2CollectionResp, err := store.GetCollection(ctx, *user2.ID, *user2Collection.NodeID)
	require.NoError(t, err)
	assert.NotNil(t, user2CollectionResp)
	assertExpectedEqualCollectionBase(t, user2Collection, user2CollectionResp.CollectionBase)
	assert.Equal(t, user2Collection.DOIs.AsDOIs(), user2CollectionResp.DOIs)

}

func testGetCollectionPublishStatus(t *testing.T, store *collections.PostgresStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	// Set up using the ExpectationDB
	user1 := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user1)

	collection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user1.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI())
	expectationDB.CreateCollection(ctx, t, collection)

	expectedPublishStatus := collectionstest.NewCompletedPublishStatus(*collection.ID, *user1.ID)
	expectationDB.CreatePublishStatus(ctx, t, expectedPublishStatus)

	// user1FiveDOI
	getCollectionResp, err := store.GetCollection(ctx, *user1.ID, *collection.NodeID)
	assert.NoError(t, err)
	assert.NotNil(t, collection)
	assertExpectedPublishStatusEqual(t, expectedPublishStatus, getCollectionResp.CollectionBase)

}

func testGetCollectionNoPerms(t *testing.T, collectionsStore *collections.PostgresStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	// Set up using the ExpectationDB
	user1 := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user1)
	user2 := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user2)

	user1CollectionFiveDOI := apitest.NewExpectedCollection().WithNodeID().
		WithUser(*user1.ID, pgdb.Owner).
		WithUser(*user2.ID, pgdb.NoPermission).
		WithDOIs(apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI())
	expectationDB.CreateCollection(ctx, t, user1CollectionFiveDOI)

	user2Collection := apitest.NewExpectedCollection().WithNodeID().
		WithUser(*user2.ID, pgdb.Owner).
		WithUser(*user1.ID, pgdb.NoPermission).
		WithDOIs(apitest.NewPennsieveDOI(), apitest.NewPennsieveDOI())
	expectationDB.CreateCollection(ctx, t, user2Collection)

	// Test with collectionsStore

	// user1FiveDOI
	_, err := collectionsStore.GetCollection(ctx, *user2.ID, *user1CollectionFiveDOI.NodeID)
	assert.ErrorIs(t, err, collections.ErrCollectionNotFound)

	// try user2's collections
	_, err = collectionsStore.GetCollection(ctx, *user1.ID, *user2Collection.NodeID)
	assert.ErrorIs(t, err, collections.ErrCollectionNotFound)

}

func testDeleteCollection(t *testing.T, store *collections.PostgresStore, expectationDB *fixtures.ExpectationDB) {
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

	require.NoError(t, store.DeleteCollection(ctx, idToDelete))

	expectationDB.RequireNoCollection(ctx, t, idToDelete)
	expectationDB.RequireCollection(ctx, t, user1CollectionKeep, keepResp.ID)
	expectationDB.RequireCollection(ctx, t, user2Collection, user2Resp.ID)
}

func testDeleteCollectionNonExistent(t *testing.T, collectionsStore *collections.PostgresStore, _ *fixtures.ExpectationDB) {
	nonExistentCollectionID := int64(99999)
	err := collectionsStore.DeleteCollection(context.Background(), nonExistentCollectionID)
	require.ErrorIs(t, err, collections.ErrCollectionNotFound)
}

func testUpdateCollectionName(t *testing.T, collectionsStore *collections.PostgresStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	user := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	expectedCollection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI())
	createResp := expectationDB.CreateCollection(ctx, t, expectedCollection)
	collectionID := createResp.ID

	newName := uuid.NewString()
	update := collections.UpdateCollectionRequest{
		Name: &newName,
	}
	updatedCollection, err := collectionsStore.UpdateCollection(context.Background(), *user.ID, collectionID, update)
	require.NoError(t, err)

	expectedCollection.Name = newName
	assertExpectedEqualCollectionBase(t, expectedCollection, updatedCollection.CollectionBase)
	assert.Equal(t, expectedCollection.DOIs.AsDOIs(), updatedCollection.DOIs)

	expectationDB.RequireCollection(ctx, t, expectedCollection, collectionID)

}

func testUpdateCollectionDescription(t *testing.T, collectionsStore *collections.PostgresStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	user := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	expectedCollection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI())
	createResp := expectationDB.CreateCollection(ctx, t, expectedCollection)
	collectionID := createResp.ID

	newDescription := uuid.NewString()
	update := collections.UpdateCollectionRequest{
		Description: &newDescription,
	}
	updatedCollection, err := collectionsStore.UpdateCollection(context.Background(), *user.ID, collectionID, update)
	require.NoError(t, err)

	expectedCollection.Description = newDescription
	assertExpectedEqualCollectionBase(t, expectedCollection, updatedCollection.CollectionBase)
	assert.Equal(t, expectedCollection.DOIs.AsDOIs(), updatedCollection.DOIs)

	expectationDB.RequireCollection(ctx, t, expectedCollection, collectionID)

}

func testUpdateCollectionNameAndDescription(t *testing.T, collectionsStore *collections.PostgresStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	user := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	expectedCollection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI())
	createResp := expectationDB.CreateCollection(ctx, t, expectedCollection)
	collectionID := createResp.ID

	newName := uuid.NewString()
	newDescription := uuid.NewString()
	update := collections.UpdateCollectionRequest{
		Name:        &newName,
		Description: &newDescription,
	}
	updatedCollection, err := collectionsStore.UpdateCollection(context.Background(), *user.ID, collectionID, update)
	require.NoError(t, err)

	expectedCollection.Name = newName
	expectedCollection.Description = newDescription

	assertExpectedEqualCollectionBase(t, expectedCollection, updatedCollection.CollectionBase)
	assert.Equal(t, expectedCollection.DOIs.AsDOIs(), updatedCollection.DOIs)

	expectationDB.RequireCollection(ctx, t, expectedCollection, collectionID)

}

func testUpdateCollectionRemoveDOI(t *testing.T, collectionsStore *collections.PostgresStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	user := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	doiToKeep1 := apitest.NewPennsieveDOI()
	doiToRemove := apitest.NewPennsieveDOI()
	doiToKeep2 := apitest.NewPennsieveDOI()

	expectedCollection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user.ID, pgdb.Owner).WithDOIs(doiToKeep1, doiToRemove, doiToKeep2)
	createResp := expectationDB.CreateCollection(ctx, t, expectedCollection)
	collectionID := createResp.ID

	update := collections.UpdateCollectionRequest{
		DOIs: collections.DOIUpdate{
			Remove: []string{doiToRemove.Value},
		},
	}
	updatedCollection, err := collectionsStore.UpdateCollection(context.Background(), *user.ID, collectionID, update)
	require.NoError(t, err)

	expectedCollection.SetDOIs(doiToKeep1, doiToKeep2)
	assertExpectedEqualCollectionBase(t, expectedCollection, updatedCollection.CollectionBase)
	assert.Equal(t, expectedCollection.DOIs.AsDOIs(), updatedCollection.DOIs)

	expectationDB.RequireCollection(ctx, t, expectedCollection, collectionID)

}

func testUpdateCollectionRemoveDOIs(t *testing.T, collectionsStore *collections.PostgresStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	user := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	doiToKeep1 := apitest.NewPennsieveDOI()
	doiToRemove1 := apitest.NewPennsieveDOI()
	doiToKeep2 := apitest.NewPennsieveDOI()
	doiToRemove2 := apitest.NewPennsieveDOI()

	expectedCollection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user.ID, pgdb.Owner).WithDOIs(doiToKeep1, doiToRemove1, doiToKeep2, doiToRemove2)
	createResp := expectationDB.CreateCollection(ctx, t, expectedCollection)
	collectionID := createResp.ID

	update := collections.UpdateCollectionRequest{
		DOIs: collections.DOIUpdate{
			Remove: []string{doiToRemove2.Value, doiToRemove1.Value},
		},
	}
	updatedCollection, err := collectionsStore.UpdateCollection(context.Background(), *user.ID, collectionID, update)
	require.NoError(t, err)

	expectedCollection.SetDOIs(doiToKeep1, doiToKeep2)
	assertExpectedEqualCollectionBase(t, expectedCollection, updatedCollection.CollectionBase)
	assert.Equal(t, expectedCollection.DOIs.AsDOIs(), updatedCollection.DOIs)

	expectationDB.RequireCollection(ctx, t, expectedCollection, collectionID)

}

func testUpdateCollectionAddDOI(t *testing.T, collectionsStore *collections.PostgresStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	user := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	doi1 := apitest.NewPennsieveDOI()
	doi2 := apitest.NewPennsieveDOI()

	expectedCollection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user.ID, pgdb.Owner).WithDOIs(doi1, doi2)
	createResp := expectationDB.CreateCollection(ctx, t, expectedCollection)
	collectionID := createResp.ID

	doiToAdd := apitest.NewPennsieveDOI()
	update := collections.UpdateCollectionRequest{
		DOIs: collections.DOIUpdate{
			Add: []collections.DOI{doiToAdd},
		},
	}
	updatedCollection, err := collectionsStore.UpdateCollection(context.Background(), *user.ID, collectionID, update)
	require.NoError(t, err)

	expectedCollection.SetDOIs(doi1, doi2, doiToAdd)
	assertExpectedEqualCollectionBase(t, expectedCollection, updatedCollection.CollectionBase)
	assert.Equal(t, expectedCollection.DOIs.AsDOIs(), updatedCollection.DOIs)

	expectationDB.RequireCollection(ctx, t, expectedCollection, collectionID)

}

func testUpdateCollectionAddDOIs(t *testing.T, collectionsStore *collections.PostgresStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	user := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	doi1 := apitest.NewPennsieveDOI()
	doi2 := apitest.NewPennsieveDOI()

	expectedCollection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user.ID, pgdb.Owner).WithDOIs(doi1, doi2)
	createResp := expectationDB.CreateCollection(ctx, t, expectedCollection)
	collectionID := createResp.ID

	doiToAdd1 := apitest.NewPennsieveDOI()
	doiToAdd2 := apitest.NewPennsieveDOI()
	update := collections.UpdateCollectionRequest{
		DOIs: collections.DOIUpdate{
			Add: []collections.DOI{doiToAdd1, doiToAdd2},
		},
	}
	updatedCollection, err := collectionsStore.UpdateCollection(context.Background(), *user.ID, collectionID, update)
	require.NoError(t, err)

	expectedCollection.SetDOIs(doi1, doi2, doiToAdd1, doiToAdd2)
	assertExpectedEqualCollectionBase(t, expectedCollection, updatedCollection.CollectionBase)
	assert.Equal(t, expectedCollection.DOIs.AsDOIs(), updatedCollection.DOIs)

	expectationDB.RequireCollection(ctx, t, expectedCollection, collectionID)

}

func testUpdateCollection(t *testing.T, collectionsStore *collections.PostgresStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	user := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	doiToKeep1 := apitest.NewPennsieveDOI()
	doiToRemove1 := apitest.NewPennsieveDOI()
	doiToKeep2 := apitest.NewPennsieveDOI()
	doiToRemove2 := apitest.NewPennsieveDOI()

	expectedCollection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user.ID, pgdb.Owner).WithDOIs(doiToRemove1, doiToKeep1, doiToKeep2, doiToRemove2)
	createResp := expectationDB.CreateCollection(ctx, t, expectedCollection)
	collectionID := createResp.ID

	newName := uuid.NewString()
	newDescription := uuid.NewString()
	newDOI := apitest.NewPennsieveDOI()
	update := collections.UpdateCollectionRequest{
		Name:        &newName,
		Description: &newDescription,
		DOIs: collections.DOIUpdate{
			Add:    []collections.DOI{newDOI},
			Remove: []string{doiToRemove1.Value, doiToRemove2.Value},
		},
	}
	updatedCollection, err := collectionsStore.UpdateCollection(context.Background(), *user.ID, collectionID, update)
	require.NoError(t, err)

	expectedCollection.Name = newName
	expectedCollection.Description = newDescription
	expectedCollection.SetDOIs(doiToKeep1, doiToKeep2, newDOI)

	assertExpectedEqualCollectionBase(t, expectedCollection, updatedCollection.CollectionBase)
	assert.Equal(t, expectedCollection.DOIs.AsDOIs(), updatedCollection.DOIs)

	expectationDB.RequireCollection(ctx, t, expectedCollection, collectionID)

}

func testUpdateCollectionRemoveNonExistentDOI(t *testing.T, collectionsStore *collections.PostgresStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	user := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	doiToKeep1 := apitest.NewPennsieveDOI()
	doiToRemove := apitest.NewPennsieveDOI()
	doiToKeep2 := apitest.NewPennsieveDOI()

	expectedCollection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user.ID, pgdb.Owner).WithDOIs(doiToKeep1, doiToRemove, doiToKeep2)
	createResp := expectationDB.CreateCollection(ctx, t, expectedCollection)
	collectionID := createResp.ID

	nonExistentDOI := apitest.NewPennsieveDOI()
	update := collections.UpdateCollectionRequest{
		DOIs: collections.DOIUpdate{
			Remove: []string{doiToRemove.Value, nonExistentDOI.Value},
		},
	}
	updatedCollection, err := collectionsStore.UpdateCollection(context.Background(), *user.ID, collectionID, update)
	require.NoError(t, err)

	expectedCollection.SetDOIs(doiToKeep1, doiToKeep2)
	assertExpectedEqualCollectionBase(t, expectedCollection, updatedCollection.CollectionBase)
	assert.Equal(t, expectedCollection.DOIs.AsDOIs(), updatedCollection.DOIs)

	expectationDB.RequireCollection(ctx, t, expectedCollection, collectionID)

}

func testUpdateCollectionAddExistingDOI(t *testing.T, collectionsStore *collections.PostgresStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	user := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	doi1 := apitest.NewPennsieveDOI()
	doi2 := apitest.NewPennsieveDOI()

	expectedCollection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user.ID, pgdb.Owner).WithDOIs(doi1, doi2)
	createResp := expectationDB.CreateCollection(ctx, t, expectedCollection)
	collectionID := createResp.ID

	newDOI := apitest.NewPennsieveDOI()
	update := collections.UpdateCollectionRequest{
		DOIs: collections.DOIUpdate{
			Add: []collections.DOI{doi1, newDOI},
		},
	}
	updatedCollection, err := collectionsStore.UpdateCollection(context.Background(), *user.ID, collectionID, update)
	require.NoError(t, err)

	expectedCollection.SetDOIs(doi1, doi2, newDOI)
	assertExpectedEqualCollectionBase(t, expectedCollection, updatedCollection.CollectionBase)
	assert.Equal(t, expectedCollection.DOIs.AsDOIs(), updatedCollection.DOIs)

	expectationDB.RequireCollection(ctx, t, expectedCollection, collectionID)

}

func testUpdateCollectionNonExistent(t *testing.T, collectionsStore *collections.PostgresStore, _ *fixtures.ExpectationDB) {
	nonExistentCollectionID := int64(99999)
	newName := uuid.NewString()
	update := collections.UpdateCollectionRequest{
		Name: &newName,
	}
	_, err := collectionsStore.UpdateCollection(context.Background(), userstest.SeedUser1.ID, nonExistentCollectionID, update)
	require.ErrorIs(t, err, collections.ErrCollectionNotFound)
}

func testUpdateCollectionNonExistentDOIUpdateOnly(t *testing.T, collectionsStore *collections.PostgresStore, _ *fixtures.ExpectationDB) {
	nonExistentCollectionID := int64(99999)
	update := collections.UpdateCollectionRequest{
		DOIs: collections.DOIUpdate{
			Remove: []string{apitest.NewPennsieveDOI().Value},
		},
	}
	_, err := collectionsStore.UpdateCollection(context.Background(), userstest.SeedUser1.ID, nonExistentCollectionID, update)
	require.ErrorIs(t, err, collections.ErrCollectionNotFound)
}

func testStartPublish(t *testing.T, collectionsStore *collections.PostgresStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	user := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	collection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI())
	collectionID := expectationDB.CreateCollection(ctx, t, collection).ID

	require.NoError(t, collectionsStore.StartPublish(ctx, collectionID, *user.ID, publishing.PublicationType))

	expectedPublishStatus := collectionstest.NewExpectedInProgressPublishStatus(collectionID, *user.ID)
	expectationDB.RequirePublishStatus(ctx, t, expectedPublishStatus, nil)
}

func testStartPublishExistingInProgress(t *testing.T, collectionsStore *collections.PostgresStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	user := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	collection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI())
	collectionID := expectationDB.CreateCollection(ctx, t, collection).ID
	existingPublishStatus := collectionstest.NewInProgressPublishStatus(collectionID, *user.ID)
	expectationDB.CreatePublishStatus(ctx, t, existingPublishStatus)

	err := collectionsStore.StartPublish(ctx, collectionID, *user.ID, publishing.PublicationType)
	require.ErrorIs(t, err, collections.ErrPublishInProgress)

	expectedPublishStatus := collectionstest.NewExpectedInProgressPublishStatus(collectionID, *user.ID)

	expectationDB.RequirePublishStatus(ctx, t, expectedPublishStatus, &existingPublishStatus)
}

func testStartPublishExistingComplete(t *testing.T, collectionsStore *collections.PostgresStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	oldUser := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, oldUser)

	user := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	collection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI())
	collectionID := expectationDB.CreateCollection(ctx, t, collection).ID
	existingPublishStatus := collectionstest.NewCompletedPublishStatus(collectionID, *oldUser.ID)

	expectationDB.CreatePublishStatus(ctx, t, existingPublishStatus)

	require.NoError(t, collectionsStore.StartPublish(ctx, collectionID, *user.ID, publishing.PublicationType))

	expectedPublishStatus := collectionstest.NewExpectedInProgressPublishStatus(collectionID, *user.ID)
	expectationDB.RequirePublishStatus(ctx, t, expectedPublishStatus, &existingPublishStatus)
}

func testStartPublishExistingFailed(t *testing.T, collectionsStore *collections.PostgresStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	oldUser := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, oldUser)

	user := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	collection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI())
	collectionID := expectationDB.CreateCollection(ctx, t, collection).ID
	existingPublishStatus := collectionstest.NewFailedPublishStatus(collectionID, *oldUser.ID)

	expectationDB.CreatePublishStatus(ctx, t, existingPublishStatus)

	require.NoError(t, collectionsStore.StartPublish(ctx, collectionID, *user.ID, publishing.PublicationType))

	expectedPublishStatus := collectionstest.NewInProgressPublishStatus(collectionID, *user.ID)
	expectationDB.RequirePublishStatus(ctx, t, expectedPublishStatus, &existingPublishStatus)
}

func testFinishPublish(t *testing.T, collectionsStore *collections.PostgresStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	user := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	collection := apitest.NewExpectedCollection().WithNodeID().WithUser(*user.ID, pgdb.Owner).WithDOIs(apitest.NewPennsieveDOI())
	collectionID := expectationDB.CreateCollection(ctx, t, collection).ID

	existingPublishStatus := collectionstest.NewInProgressPublishStatus(collectionID, *user.ID)
	expectationDB.CreatePublishStatus(ctx, t, existingPublishStatus)

	require.NoError(t, collectionsStore.FinishPublish(ctx, collectionID, publishing.CompletedStatus, true))

	expectedPublishStatus := collectionstest.NewExpectedCompletedPublishStatus(collectionID, *user.ID)
	expectationDB.RequirePublishStatus(ctx, t, expectedPublishStatus, &existingPublishStatus)
}

func testFinishPublishNoExistingStatus(t *testing.T, collectionsStore *collections.PostgresStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	collectionID := int64(99999)

	require.Error(t, collectionsStore.FinishPublish(ctx, collectionID, publishing.CompletedStatus, true))

	expectationDB.RequireNoPublishStatus(ctx, t, collectionID)
}

func assertExpectedEqualCollectionBase(t *testing.T, expected *apitest.ExpectedCollection, actual collections.CollectionBase) {
	t.Helper()
	assert.Equal(t, *expected.NodeID, actual.NodeID)
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.Description, actual.Description)
	assert.Equal(t, expected.Users[0].PermissionBit.ToRole(), actual.UserRole)
	assert.Equal(t, len(expected.DOIs), actual.Size)
}

func assertExpectedPublishStatusEqual(t *testing.T, expected collections.PublishStatus, actual collections.CollectionBase) {
	t.Helper()
	require.NotNil(t, actual.Publication)
	assert.Equal(t, expected.Type, actual.Publication.Type)
	assert.Equal(t, expected.Status, actual.Publication.Status)
}

func assertExpectedEqualCollectionSummary(t *testing.T, expected *apitest.ExpectedCollection, actual collections.CollectionSummary) {
	t.Helper()
	assertExpectedEqualCollectionBase(t, expected, actual.CollectionBase)
	bannerLen := min(collections.MaxBannerDOIsPerCollection, len(expected.DOIs))
	assert.Equal(t, expected.DOIs.Strings()[:bannerLen], actual.BannerDOIs)
}
