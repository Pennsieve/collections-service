package api

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/pennsieve/pennsieve-go-core/pkg/authorizer"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/user"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/pennsieve/collections-service/internal/shared/config"
	"github.com/pennsieve/collections-service/internal/test/containertest"
	"github.com/stretchr/testify/assert"
)

func TestAPILambdaHandler(t *testing.T) {
	for scenario, fn := range map[string]func(
		tt *testing.T,
	){
		"default not found response": testDefaultNotFound,
		"no claims":                  testNoClaims,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func testDefaultNotFound(t *testing.T) {
	handler := CollectionsServiceAPIHandler(containertest.NewMockTestContainer(), config.Config{})

	req := events.APIGatewayV2HTTPRequest{
		RouteKey: "GET /unknown",
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			Authorizer: &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{
				Lambda: make(map[string]interface{}),
			},
		},
	}

	WithClaims(&req, ClaimsToMap(DefaultClaims()))

	expectedResponse := events.APIGatewayV2HTTPResponse{
		StatusCode: http.StatusNotFound,
		Body:       "Not found",
	}

	response, err := handler(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, response)
}

func testNoClaims(t *testing.T) {
	handler := CollectionsServiceAPIHandler(containertest.NewMockTestContainer(), config.Config{})

	req := events.APIGatewayV2HTTPRequest{
		RouteKey: "GET /unknown",
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			Authorizer: &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{
				Lambda: make(map[string]interface{}),
			},
		},
	}

	expectedResponse := events.APIGatewayV2HTTPResponse{
		StatusCode: http.StatusUnauthorized,
		Body:       "Unauthorized",
	}

	response, err := handler(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, response)
}

func WithClaims(request *events.APIGatewayV2HTTPRequest, claims map[string]interface{}) {
	requestAuthorizer := request.RequestContext.Authorizer
	if requestAuthorizer == nil {
		requestAuthorizer = &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{}
	}

	requestAuthorizer.Lambda = claims

	request.RequestContext.Authorizer = requestAuthorizer
}

func DefaultClaims() authorizer.Claims {
	return authorizer.Claims{
		UserClaim: &user.Claim{
			Id:           101,
			NodeId:       fmt.Sprintf("N:user:%s", uuid.NewString()),
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
