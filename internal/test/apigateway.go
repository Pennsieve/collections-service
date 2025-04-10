package test

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/pennsieve/pennsieve-go-core/pkg/authorizer"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/user"
	"github.com/stretchr/testify/require"
)

type APIGatewayRequestBuilder struct {
	r *events.APIGatewayV2HTTPRequest
}

func NewAPIGatewayRequestBuilder(routeKey string) *APIGatewayRequestBuilder {
	return &APIGatewayRequestBuilder{r: &events.APIGatewayV2HTTPRequest{
		RouteKey: routeKey,
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			Authorizer: &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{
				Lambda: make(map[string]interface{}),
			},
		},
	}}
}

func (b *APIGatewayRequestBuilder) WithClaims(claims authorizer.Claims) *APIGatewayRequestBuilder {
	b.r.RequestContext.Authorizer.Lambda = ClaimsToMap(claims)
	return b
}

func (b *APIGatewayRequestBuilder) WithDefaultClaims(seedUser SeedUser) *APIGatewayRequestBuilder {
	return b.WithClaims(DefaultClaims(seedUser))
}

func (b *APIGatewayRequestBuilder) WithBody(t require.TestingT, bodyStruct any) *APIGatewayRequestBuilder {
	bodyBytes, err := json.Marshal(bodyStruct)
	require.NoError(t, err)
	b.r.Body = string(bodyBytes)
	return b
}

func (b *APIGatewayRequestBuilder) WithQueryParam(key string, value string) *APIGatewayRequestBuilder {
	if b.r.QueryStringParameters == nil {
		b.r.QueryStringParameters = make(map[string]string)
	}
	b.r.QueryStringParameters[key] = value
	return b
}

func (b *APIGatewayRequestBuilder) WithIntQueryParam(key string, value int) *APIGatewayRequestBuilder {
	return b.WithQueryParam(key, fmt.Sprintf("%d", value))
}

func (b *APIGatewayRequestBuilder) Build() events.APIGatewayV2HTTPRequest {
	return *b.r
}

func DefaultClaims(seedUser SeedUser) authorizer.Claims {
	return authorizer.Claims{
		UserClaim: &user.Claim{
			Id:           seedUser.ID,
			NodeId:       seedUser.NodeID,
			IsSuperAdmin: seedUser.IsSuperAdmin,
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
