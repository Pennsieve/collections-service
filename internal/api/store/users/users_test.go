package users_test

import (
	"context"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/store/users"
	"github.com/pennsieve/collections-service/internal/shared/logging"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/pennsieve/collections-service/internal/test/fixtures"
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
		{"return correct names and ORCID ID", testNamesAndOrcid},
		{"return correct error when user does not exist", testUserNotFound},
		{"handle null fields", testNullableFields},
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

func testNamesAndOrcid(t *testing.T, store *users.PostgresStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	user := apitest.NewTestUser(
		apitest.WithFirstName(uuid.NewString()),
		apitest.WithLastName(uuid.NewString()),
		apitest.WithORCID(uuid.NewString()),
	)
	expectationDB.CreateTestUser(ctx, t, user)

	userResp, err := store.GetUser(ctx, user.GetID())
	require.NoError(t, err)
	assert.Equal(t, user.FirstName, userResp.FirstName)
	assert.Equal(t, user.LastName, userResp.LastName)
	assert.Equal(t, (*user.ORCIDAuthorization).ORCID, *userResp.ORCID)
}

func testUserNotFound(t *testing.T, store *users.PostgresStore, _ *fixtures.ExpectationDB) {
	ctx := context.Background()
	_, err := store.GetUser(ctx, 101)
	assert.ErrorIs(t, err, users.ErrUserNotFound)
}

func testNullableFields(t *testing.T, store *users.PostgresStore, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	user := apitest.NewTestUser()
	expectationDB.CreateTestUser(ctx, t, user)

	userResp, err := store.GetUser(ctx, user.GetID())
	require.NoError(t, err)
	assert.Nil(t, userResp.FirstName)
	assert.Nil(t, userResp.LastName)
	assert.Nil(t, userResp.ORCID)
}
