package api

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/pennsieve/pennsieve-go-core/pkg/authorizer"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/user"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
)

func TestAPILambdaHandler(t *testing.T) {
	for scenario, fn := range map[string]func(
		tt *testing.T,
	){
		"default not found response": testDefaultNotFound,
		"no claims":                  testNoClaims,
		"create collection":          testCreateCollection,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func testDefaultNotFound(t *testing.T) {
	handler := CollectionsServiceAPIHandler(apitest.NewTestContainer(t), apitest.Config())

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
	handler := CollectionsServiceAPIHandler(apitest.NewTestContainer(t), apitest.Config())

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

func testCreateCollection(t *testing.T) {
	t.Skip("working on this")
	createCollectionRequest := dto.CreateCollectionRequest{
		Name:        uuid.NewString(),
		Description: uuid.NewString(),
		DOIs:        []string{test.NewDOI(), test.NewDOI()},
	}
	bodyBytes, err := json.Marshal(createCollectionRequest)
	require.NoError(t, err)

	handler := CollectionsServiceAPIHandler(apitest.NewTestContainer(t), apitest.Config())

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
	assert.Equal(t, http.StatusCreated, response.StatusCode)
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
