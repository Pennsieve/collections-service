package api

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/store"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/pennsieve/collections-service/internal/test/mocks"
	"github.com/pennsieve/pennsieve-go-core/pkg/authorizer"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"strings"
	"testing"
)

func TestAPILambdaHandler(t *testing.T) {
	for scenario, fn := range map[string]func(
		tt *testing.T,
	){
		"default not found response": testDefaultNotFound,
		"no claims":                  testNoClaims,
		"create collection bad request: external DOIs":        testCreateCollectionExternalDOIs,
		"create collection bad request: empty name":           testCreateCollectionEmptyName,
		"create collection bad request: name too long":        testCreateCollectionNameTooLong,
		"create collection bad request: description too long": testCreateCollectionDescriptionTooLong,
		"create collection bad request: unpublished DOIs":     testCreateCollectionUnpublishedDOIs,
		"create collection":                                   testCreateCollection,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func testDefaultNotFound(t *testing.T) {
	handler := CollectionsServiceAPIHandler(apitest.NewTestContainer(), apitest.NewConfigBuilder().Build())

	req := events.APIGatewayV2HTTPRequest{
		RouteKey: "GET /unknown",
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			Authorizer: &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{
				Lambda: make(map[string]interface{}),
			},
		},
	}

	WithClaims(&req, ClaimsToMap(DefaultClaims(test.User2)))

	response, err := handler(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, response.StatusCode)
	assert.Contains(t, response.Body, "not found")
	assert.Contains(t, response.Body, `"error_id"`)
}

func testNoClaims(t *testing.T) {
	handler := CollectionsServiceAPIHandler(apitest.NewTestContainer(), apitest.NewConfigBuilder().Build())

	req := events.APIGatewayV2HTTPRequest{
		RouteKey: "POST /collections",
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			Authorizer: &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{
				Lambda: make(map[string]interface{}),
			},
		},
	}

	response, err := handler(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

func testCreateCollectionExternalDOIs(t *testing.T) {
	createCollectionRequest := dto.CreateCollectionRequest{
		Name:        uuid.NewString(),
		Description: uuid.NewString(),
		DOIs:        []string{test.NewExternalDOI(), test.NewPennsieveDOI(), test.NewExternalDOI()},
	}
	bodyBytes, err := json.Marshal(createCollectionRequest)
	require.NoError(t, err)

	handler := CollectionsServiceAPIHandler(
		apitest.NewTestContainer(),
		apitest.NewConfigBuilder().
			WithPennsieveConfig(apitest.PennsieveConfigWithFakeHost()).
			Build())

	req := events.APIGatewayV2HTTPRequest{
		RouteKey: "POST /collections",
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			Authorizer: &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{
				Lambda: make(map[string]interface{}),
			},
		},
		Body: string(bodyBytes),
	}

	WithClaims(&req, ClaimsToMap(DefaultClaims(test.User)))

	response, err := handler(context.Background(), req)

	assert.NoError(t, err)
	assert.Contains(t, response.Body, createCollectionRequest.DOIs[0])
	assert.NotContains(t, response.Body, createCollectionRequest.DOIs[1])
	assert.Contains(t, response.Body, createCollectionRequest.DOIs[2])
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func testCreateCollectionEmptyName(t *testing.T) {
	createCollectionRequest := dto.CreateCollectionRequest{
		Name:        "",
		Description: uuid.NewString(),
	}
	bodyBytes, err := json.Marshal(createCollectionRequest)
	require.NoError(t, err)

	handler := CollectionsServiceAPIHandler(
		apitest.NewTestContainer(),
		apitest.NewConfigBuilder().
			WithPennsieveConfig(apitest.PennsieveConfigWithFakeHost()).
			Build())

	req := events.APIGatewayV2HTTPRequest{
		RouteKey: "POST /collections",
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			Authorizer: &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{
				Lambda: make(map[string]interface{}),
			},
		},
		Body: string(bodyBytes),
	}

	WithClaims(&req, ClaimsToMap(DefaultClaims(test.User)))

	response, err := handler(context.Background(), req)

	assert.NoError(t, err)
	assert.Contains(t, response.Body, "collection name cannot be empty")
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func testCreateCollectionNameTooLong(t *testing.T) {
	createCollectionRequest := dto.CreateCollectionRequest{
		Name:        strings.Repeat("a", 256),
		Description: uuid.NewString(),
	}
	bodyBytes, err := json.Marshal(createCollectionRequest)
	require.NoError(t, err)

	handler := CollectionsServiceAPIHandler(
		apitest.NewTestContainer(),
		apitest.NewConfigBuilder().
			WithPennsieveConfig(apitest.PennsieveConfigWithFakeHost()).
			Build())

	req := events.APIGatewayV2HTTPRequest{
		RouteKey: "POST /collections",
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			Authorizer: &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{
				Lambda: make(map[string]interface{}),
			},
		},
		Body: string(bodyBytes),
	}

	WithClaims(&req, ClaimsToMap(DefaultClaims(test.User)))

	response, err := handler(context.Background(), req)

	assert.NoError(t, err)
	assert.Contains(t, response.Body, "collection name cannot have more than 255 characters")
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func testCreateCollectionDescriptionTooLong(t *testing.T) {
	createCollectionRequest := dto.CreateCollectionRequest{
		Name:        uuid.NewString(),
		Description: strings.Repeat("a", 256),
	}
	bodyBytes, err := json.Marshal(createCollectionRequest)
	require.NoError(t, err)

	handler := CollectionsServiceAPIHandler(
		apitest.NewTestContainer(),
		apitest.NewConfigBuilder().
			WithPennsieveConfig(apitest.PennsieveConfigWithFakeHost()).
			Build())

	req := events.APIGatewayV2HTTPRequest{
		RouteKey: "POST /collections",
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			Authorizer: &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{
				Lambda: make(map[string]interface{}),
			},
		},
		Body: string(bodyBytes),
	}

	WithClaims(&req, ClaimsToMap(DefaultClaims(test.User)))

	response, err := handler(context.Background(), req)

	assert.NoError(t, err)
	assert.Contains(t, response.Body, "collection description cannot have more than 255 characters")
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func testCreateCollectionUnpublishedDOIs(t *testing.T) {
	publishedDOI := test.NewPennsieveDOI()
	unpublishedDOI := test.NewPennsieveDOI()
	expectedStatus := "PublishFailed"
	createCollectionRequest := dto.CreateCollectionRequest{
		Name:        uuid.NewString(),
		Description: uuid.NewString(),
		DOIs:        []string{publishedDOI, unpublishedDOI},
	}
	bodyBytes, err := json.Marshal(createCollectionRequest)
	require.NoError(t, err)

	mockDiscoverService := mocks.NewMockDiscover().
		WithGetDatasetsByDOIFunc(func(dois []string) (dto.DatasetsByDOIResponse, error) {
			test.Helper(t)
			require.Equal(t, []string{publishedDOI, unpublishedDOI}, dois)
			return dto.DatasetsByDOIResponse{
				Published: map[string]dto.PublicDataset{publishedDOI: {
					ID:      1,
					Version: 3,
					DOI:     publishedDOI,
				}},
				Unpublished: map[string]dto.Tombstone{unpublishedDOI: {
					ID:      2,
					Version: 1,
					Status:  expectedStatus,
					DOI:     unpublishedDOI,
				}},
			}, nil
		})

	handler := CollectionsServiceAPIHandler(
		apitest.NewTestContainer().WithDiscover(mockDiscoverService),
		apitest.NewConfigBuilder().
			WithPennsieveConfig(apitest.PennsieveConfigWithFakeHost()).
			Build())

	req := events.APIGatewayV2HTTPRequest{
		RouteKey: "POST /collections",
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			Authorizer: &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{
				Lambda: make(map[string]interface{}),
			},
		},
		Body: string(bodyBytes),
	}

	WithClaims(&req, ClaimsToMap(DefaultClaims(test.User)))

	response, err := handler(context.Background(), req)

	assert.NoError(t, err)
	assert.Contains(t, response.Body, unpublishedDOI)
	assert.Contains(t, response.Body, expectedStatus)
	assert.NotContains(t, response.Body, publishedDOI)
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)

}
func testCreateCollection(t *testing.T) {
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
	bodyBytes, err := json.Marshal(createCollectionRequest)
	require.NoError(t, err)

	mockDiscoverService := mocks.NewMockDiscover().
		WithGetDatasetsByDOIFunc(func(dois []string) (dto.DatasetsByDOIResponse, error) {
			t.Helper()
			require.Equal(t, []string{publishedDOI1, publishedDOI2}, dois)
			return dto.DatasetsByDOIResponse{
				Published: map[string]dto.PublicDataset{
					publishedDOI1: {
						ID:      1,
						Version: 3,
						DOI:     publishedDOI1,
						Banner:  banner1,
					},
					publishedDOI2: {
						ID:      5,
						Version: 1,
						DOI:     publishedDOI2,
						Banner:  banner2,
					}},
			}, nil
		})

	var collectionNodeID string

	mockCollectionsStore := mocks.NewMockCollectionsStore().WithCreateCollectionsFunc(func(_ context.Context, userID int64, nodeID, name, description string, dois []string) (*store.CreateCollectionResponse, error) {
		t.Helper()
		require.Equal(t, callingUser.ID, userID)
		require.NotEmpty(t, nodeID)
		collectionNodeID = nodeID
		require.Equal(t, createCollectionRequest.Name, name)
		require.Equal(t, createCollectionRequest.Description, description)
		require.Equal(t, createCollectionRequest.DOIs, dois)
		return &store.CreateCollectionResponse{
			ID:          1,
			CreatorRole: role.Owner,
		}, nil
	})

	handler := CollectionsServiceAPIHandler(
		apitest.NewTestContainer().WithDiscover(mockDiscoverService).WithCollectionsStore(mockCollectionsStore),
		apitest.NewConfigBuilder().
			WithPennsieveConfig(apitest.PennsieveConfigWithFakeHost()).
			Build())

	req := events.APIGatewayV2HTTPRequest{
		RouteKey: "POST /collections",
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			Authorizer: &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{
				Lambda: make(map[string]interface{}),
			},
		},
		Body: string(bodyBytes),
	}

	WithClaims(&req, ClaimsToMap(DefaultClaims(callingUser)))

	response, err := handler(context.Background(), req)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusCreated, response.StatusCode)

	var responseDTO dto.CollectionResponse
	require.NoError(t, json.Unmarshal([]byte(response.Body), &responseDTO))

	assert.Equal(t, collectionNodeID, responseDTO.NodeID)
	assert.Equal(t, createCollectionRequest.Name, responseDTO.Name)
	assert.Equal(t, createCollectionRequest.Description, responseDTO.Description)
	assert.Equal(t, len(createCollectionRequest.DOIs), responseDTO.Size)
	assert.Equal(t, role.Owner.String(), responseDTO.UserRole)
	assert.Equal(t, []string{*banner1, *banner2}, responseDTO.Banners)
}

func WithClaims(request *events.APIGatewayV2HTTPRequest, claims map[string]interface{}) {
	requestAuthorizer := request.RequestContext.Authorizer
	if requestAuthorizer == nil {
		requestAuthorizer = &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{}
	}

	requestAuthorizer.Lambda = claims

	request.RequestContext.Authorizer = requestAuthorizer
}

func DefaultClaims(seedUser test.SeedUser) authorizer.Claims {
	return authorizer.Claims{
		UserClaim: &user.Claim{
			Id:           seedUser.ID,
			NodeId:       seedUser.NodeID,
			IsSuperAdmin: false,
		},
	}
}

func ClaimsToMap(claims authorizer.Claims) map[string]interface{} {
	asMap := map[string]any{}
	if claims.UserClaim != nil {
		asMap[authorizer.LabelUserClaim] = map[string]any{
			"Id":           float64(claims.UserClaim.Id),
			"NodeId":       claims.UserClaim.NodeId,
			"IsSuperAdmin": claims.UserClaim.IsSuperAdmin,
		}
	}
	if claims.OrgClaim != nil {
		asMap[authorizer.LabelOrganizationClaim] = map[string]any{
			"Role":   float64(claims.OrgClaim.Role),
			"IntId":  float64(claims.OrgClaim.IntId),
			"NodeId": claims.OrgClaim.NodeId,
		}
	}
	if claims.DatasetClaim != nil {
		asMap[authorizer.LabelDatasetClaim] = map[string]any{
			"Role":   float64(claims.DatasetClaim.Role),
			"IntId":  float64(claims.DatasetClaim.IntId),
			"NodeId": claims.DatasetClaim.NodeId,
		}
	}
	return asMap
}
