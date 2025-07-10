package users_test

import (
	"context"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/store/users"
	"github.com/pennsieve/collections-service/internal/shared/logging"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/fixtures"
	"github.com/pennsieve/collections-service/internal/test/userstest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPostgresStore(t *testing.T) {
	ctx := context.Background()
	config := test.PostgresDBConfig(t)

	for _, tt := range []struct {
		scenario string
		tstFunc  func(t *testing.T, store *users.PostgresStore, expectationDB *fixtures.ExpectationDB)
	}{
		{"GetUser should return correct values", testGetUser},
		{"GetUser should return correct error when user does not exist", testGetUserUserNotFound},
		{"GetUser should handle null fields", testGetUserNullableFields},
	} {
		t.Run(tt.scenario, func(t *testing.T) {
			db := test.NewPostgresDBFromConfig(t, config)
			expectationDB := fixtures.NewExpectationDB(db, config.CollectionsDatabase)
			t.Cleanup(func() {
				expectationDB.CleanUp(ctx, t)
			})

			store := users.NewPostgresStore(db, config.CollectionsDatabase, logging.Default)

			tt.tstFunc(t, store, expectationDB)
		})
	}
}

func testGetUser(t *testing.T, store *users.PostgresStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	user := userstest.NewTestUser(
		userstest.WithFirstName(uuid.NewString()),
		userstest.WithLastName(uuid.NewString()),
		userstest.WithORCID(uuid.NewString()),
		userstest.WithMiddleInitial("F"),
		userstest.WithDegree("Ph.D."),
	)
	expectationDB.CreateTestUser(ctx, t, user)

	userResp, err := store.GetUser(ctx, user.GetID())
	require.NoError(t, err)
	assert.Equal(t, user.FirstName, userResp.FirstName)
	assert.Equal(t, user.LastName, userResp.LastName)
	assert.Equal(t, (*user.ORCIDAuthorization).ORCID, *userResp.ORCID)
	assert.Equal(t, user.MiddleInitial, userResp.MiddleInitial)
	assert.Equal(t, user.Degree, userResp.Degree)
}

func testGetUserUserNotFound(t *testing.T, store *users.PostgresStore, _ *fixtures.ExpectationDB) {
	ctx := context.Background()
	_, err := store.GetUser(ctx, 101)
	assert.ErrorIs(t, err, users.ErrUserNotFound)
}

func testGetUserNullableFields(t *testing.T, store *users.PostgresStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	user := userstest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	userResp, err := store.GetUser(ctx, user.GetID())
	require.NoError(t, err)
	assert.Nil(t, userResp.FirstName)
	assert.Nil(t, userResp.LastName)
	assert.Nil(t, userResp.ORCID)
	assert.Nil(t, userResp.MiddleInitial)
	assert.Nil(t, userResp.Degree)
}
