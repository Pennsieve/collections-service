package routes

import (
	"context"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/collections-service/internal/dbmigrate"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/pennsieve/collections-service/internal/test/configtest"
	"github.com/pennsieve/collections-service/internal/test/dbmigratetest"
	"github.com/pennsieve/collections-service/internal/test/fixtures"
	"github.com/pennsieve/collections-service/internal/test/mocks"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
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
		"create collection; no DTOs":              testCreateCollectionNoDTOs,
		"create collection; two DTOs":             testCreateCollectionTwoDTOs,
		"create collection; five DTOs":            testCreateCollectionFiveDTOs,
		"create collection; some missing banners": testCreateCollectionSomeMissingBanners,
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

func testCreateCollectionNoDTOs(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	callingUser := test.User

	expectedCollection := fixtures.NewExpectedCollection().
		WithUser(callingUser.ID, pgdb.Owner)

	createCollectionRequest := dto.CreateCollectionRequest{
		Name:        expectedCollection.Name,
		Description: expectedCollection.Description,
	}

	claims := test.DefaultClaims(callingUser)

	config := apitest.NewConfigBuilder().
		WithDockerPostgresDBConfig().
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, config.PostgresDB)).
		WithContainerStoreFromPostgresDB(config.PostgresDB.CollectionsDatabase)

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
	assert.Zero(t, response.Size)
	assert.Equal(t, role.Owner.String(), response.UserRole)

	expectationDB.RequireCollectionByNodeID(ctx, t, expectedCollection, response.NodeID)
}

func testCreateCollectionTwoDTOs(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	callingUser := test.User

	publishedDOI1 := test.NewPennsieveDOI()
	banner1 := test.NewBanner()

	publishedDOI2 := test.NewPennsieveDOI()
	banner2 := test.NewBanner()

	expectedCollection := fixtures.NewExpectedCollection().
		WithUser(callingUser.ID, pgdb.Owner).
		WithDOIs(publishedDOI1, publishedDOI2)

	createCollectionRequest := dto.CreateCollectionRequest{
		Name:        expectedCollection.Name,
		Description: expectedCollection.Description,
		DOIs:        expectedCollection.DOIs.Strings(),
	}

	mockDiscoverServer := httptest.NewServer(mocks.ToDiscoverHandlerFunc(t, func(dois []string) (service.DatasetsByDOIResponse, error) {
		t.Helper()
		require.Equal(t, []string{publishedDOI1, publishedDOI2}, dois)
		return service.DatasetsByDOIResponse{
			Published: map[string]dto.PublicDataset{
				publishedDOI1: test.NewPublicDataset(publishedDOI1, banner1),
				publishedDOI2: test.NewPublicDataset(publishedDOI2, banner2)},
		}, nil
	}))
	defer mockDiscoverServer.Close()

	claims := test.DefaultClaims(callingUser)

	config := apitest.NewConfigBuilder().
		WithDockerPostgresDBConfig().
		WithPennsieveConfig(apitest.PennsieveConfig(mockDiscoverServer.URL)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, config.PostgresDB)).
		WithHTTPTestDiscover(mockDiscoverServer.URL).
		WithContainerStoreFromPostgresDB(config.PostgresDB.CollectionsDatabase)

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

	expectationDB.RequireCollectionByNodeID(ctx, t, expectedCollection, response.NodeID)
}

func testCreateCollectionFiveDTOs(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	callingUser := test.User

	publishedDOI1 := test.NewPennsieveDOI()
	banner1 := test.NewBanner()

	publishedDOI2 := test.NewPennsieveDOI()
	banner2 := test.NewBanner()

	publishedDOI3 := test.NewPennsieveDOI()
	banner3 := test.NewBanner()

	publishedDTO4 := test.NewPennsieveDOI()
	banner4 := test.NewBanner()

	publishedDTO5 := test.NewPennsieveDOI()
	banner5 := test.NewBanner()

	expectedCollection := fixtures.NewExpectedCollection().
		WithUser(callingUser.ID, pgdb.Owner).
		WithDOIs(publishedDOI1, publishedDOI2, publishedDOI3, publishedDTO4, publishedDTO5)

	createCollectionRequest := dto.CreateCollectionRequest{
		Name:        expectedCollection.Name,
		Description: expectedCollection.Description,
		DOIs:        expectedCollection.DOIs.Strings(),
	}

	mockDiscoverServer := httptest.NewServer(mocks.ToDiscoverHandlerFunc(t, func(dois []string) (service.DatasetsByDOIResponse, error) {
		t.Helper()
		require.Equal(t, []string{publishedDOI1, publishedDOI2, publishedDOI3, publishedDTO4, publishedDTO5}, dois)
		return service.DatasetsByDOIResponse{
			Published: map[string]dto.PublicDataset{
				publishedDOI1: test.NewPublicDataset(publishedDOI1, banner1),
				publishedDOI2: test.NewPublicDataset(publishedDOI2, banner2),
				publishedDOI3: test.NewPublicDataset(publishedDOI3, banner3),
				publishedDTO4: test.NewPublicDataset(publishedDTO4, banner4),
				publishedDTO5: test.NewPublicDataset(publishedDTO5, banner5),
			},
		}, nil
	}))
	defer mockDiscoverServer.Close()

	claims := test.DefaultClaims(callingUser)

	config := apitest.NewConfigBuilder().
		WithDockerPostgresDBConfig().
		WithPennsieveConfig(apitest.PennsieveConfig(mockDiscoverServer.URL)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, config.PostgresDB)).
		WithHTTPTestDiscover(mockDiscoverServer.URL).
		WithContainerStoreFromPostgresDB(config.PostgresDB.CollectionsDatabase)

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
	assert.Equal(t, []string{*banner1, *banner2, *banner3, *banner4}, response.Banners)
	assert.Equal(t, role.Owner.String(), response.UserRole)

	expectationDB.RequireCollectionByNodeID(ctx, t, expectedCollection, response.NodeID)

}

func testCreateCollectionSomeMissingBanners(t *testing.T, expectationDB *fixtures.ExpectationDB) {
	ctx := context.Background()

	callingUser := test.User

	publishedDOI1 := test.NewPennsieveDOI()
	var banner1 *string = nil

	publishedDOI2 := test.NewPennsieveDOI()
	banner2 := test.NewBanner()

	publishedDOI3 := test.NewPennsieveDOI()
	var banner3 *string = nil

	publishedDTO4 := test.NewPennsieveDOI()
	banner4 := test.NewBanner()

	publishedDTO5 := test.NewPennsieveDOI()
	var banner5 *string = nil

	expectedCollection := fixtures.NewExpectedCollection().
		WithUser(callingUser.ID, pgdb.Owner).
		WithDOIs(publishedDOI1, publishedDOI2, publishedDOI3, publishedDTO4, publishedDTO5)

	createCollectionRequest := dto.CreateCollectionRequest{
		Name:        expectedCollection.Name,
		Description: expectedCollection.Description,
		DOIs:        expectedCollection.DOIs.Strings(),
	}

	mockDiscoverServer := httptest.NewServer(mocks.ToDiscoverHandlerFunc(t, func(dois []string) (service.DatasetsByDOIResponse, error) {
		t.Helper()
		require.Equal(t, []string{publishedDOI1, publishedDOI2, publishedDOI3, publishedDTO4, publishedDTO5}, dois)
		return service.DatasetsByDOIResponse{
			Published: map[string]dto.PublicDataset{
				publishedDOI1: test.NewPublicDataset(publishedDOI1, banner1),
				publishedDOI2: test.NewPublicDataset(publishedDOI2, banner2),
				publishedDOI3: test.NewPublicDataset(publishedDOI3, banner3),
				publishedDTO4: test.NewPublicDataset(publishedDTO4, banner4),
				publishedDTO5: test.NewPublicDataset(publishedDTO5, banner5),
			},
		}, nil
	}))
	defer mockDiscoverServer.Close()

	claims := test.DefaultClaims(callingUser)

	config := apitest.NewConfigBuilder().
		WithDockerPostgresDBConfig().
		WithPennsieveConfig(apitest.PennsieveConfig(mockDiscoverServer.URL)).
		Build()

	container := apitest.NewTestContainer().
		WithPostgresDB(test.NewPostgresDBFromConfig(t, config.PostgresDB)).
		WithHTTPTestDiscover(mockDiscoverServer.URL).
		WithContainerStoreFromPostgresDB(config.PostgresDB.CollectionsDatabase)

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
	assert.Equal(t, []string{"", *banner2, "", *banner4}, response.Banners)
	assert.Equal(t, role.Owner.String(), response.UserRole)

	expectationDB.RequireCollectionByNodeID(ctx, t, expectedCollection, response.NodeID)

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
