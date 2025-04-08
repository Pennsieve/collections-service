package routes

import (
	"context"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/dbmigrate"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/pennsieve/collections-service/internal/test/configtest"
	"github.com/pennsieve/collections-service/internal/test/dbmigratetest"
	"github.com/pennsieve/collections-service/internal/test/fixtures"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCreateCollection(t *testing.T) {
	ctx := context.Background()
	config := configtest.PostgresDBConfig()
	migrator, err := dbmigrate.NewLocalCollectionsMigrator(ctx, dbmigrate.Config{
		PostgresDB:     config,
		VerboseLogging: true,
	})
	require.NoError(t, err)
	require.NoError(t, migrator.Up())
	dbmigratetest.Close(t, migrator)

	for scenario, tstFunc := range map[string]func(t *testing.T, expectationDB *fixtures.ExpectationDB){
		"create collection": testCreateCollection,
	} {
		t.Run(scenario, func(t *testing.T) {
			db := test.NewPostgresDBFromConfig(t, config)

			t.Cleanup(func() {
				require.NoError(t, fixtures.TruncateCollectionsSchema(ctx, db, config.CollectionsDatabase))
			})

			tstFunc(t, fixtures.NewExpectationDB(db, config.CollectionsDatabase))
		})
	}

}

func testCreateCollection(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	t.Skip("need a mock Discover")
	ctx := context.Background()

	publishedDOI1 := test.NewPennsieveDOI()
	banner1 := test.NewBanner()

	publishedDOI2 := test.NewPennsieveDOI()
	banner2 := test.NewBanner()

	callingUser := test.User

	createCollectionRequest := dto.CreateCollectionRequest{
		Name:        uuid.NewString(),
		Description: uuid.NewString(),
		DOIs:        []string{publishedDOI1, publishedDOI2},
	}

	claims := test.DefaultClaims(callingUser)

	config := apitest.NewConfigBuilder().
		WithDockerPostgresDBConfig().
		WithPennsieveConfig(apitest.PennsieveConfigWithFakeHost()).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, config.PostgresDB))

	params := Params{
		Request: test.NewAPIGatewayRequestBuilder("POST /collections").
			WithClaims(claims).
			WithBody(t, createCollectionRequest).
			Build(),
		Container: container,
		Config:    config,
		Claims:    &claims,
	}

	response, err := CreateCollection(ctx, params)
	require.NoError(t, err)

	assert.NotEmpty(t, t, response.NodeID)
	assert.Equal(t, createCollectionRequest.Name, response.Name)
	assert.Equal(t, createCollectionRequest.Description, response.Description)
	assert.Equal(t, len(createCollectionRequest.DOIs), response.Size)
	assert.Equal(t, []string{*banner1, *banner2}, response.Banners)
	assert.Equal(t, role.Owner.String(), response.UserRole)
}

func TestCategorizeDOIs(t *testing.T) {
	pennsieveDOI1 := test.NewPennsieveDOI()
	pennsieveDOI2 := test.NewPennsieveDOI()
	pennsieveDOI3 := test.NewPennsieveDOI()

	externalDOI1 := test.NewExternalDOI()
	externalDOI2 := test.NewExternalDOI()

	type args struct {
		inputDOIs             []string
		expectedPennsieveDOIs []string
		expectedExternalDOIs  []string
	}
	tests := []struct {
		name string
		args args
	}{
		{"no dois",
			args{nil, nil, nil},
		},
		{"no dups",
			args{
				inputDOIs:             []string{pennsieveDOI1, pennsieveDOI2, externalDOI1, pennsieveDOI3, externalDOI2},
				expectedPennsieveDOIs: []string{pennsieveDOI1, pennsieveDOI2, pennsieveDOI3},
				expectedExternalDOIs:  []string{externalDOI1, externalDOI2}},
		},
		{"some dups",
			args{inputDOIs: []string{pennsieveDOI3, pennsieveDOI1, pennsieveDOI2, pennsieveDOI3, pennsieveDOI2},
				expectedPennsieveDOIs: []string{pennsieveDOI3, pennsieveDOI1, pennsieveDOI2},
				expectedExternalDOIs:  nil},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualPennsieve, actualExternal := CategorizeDOIs(test.PennsieveDOIPrefix, tt.args.inputDOIs)
			assert.Equal(t, tt.args.expectedPennsieveDOIs, actualPennsieve)
			assert.Equal(t, tt.args.expectedExternalDOIs, actualExternal)
		})
	}

}
