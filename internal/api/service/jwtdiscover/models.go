package jwtdiscover

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/pennsieve/pennsieve-go-core/pkg/authorizer"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/dataset"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/organization"
	"strconv"
	"strings"
	"time"
)

var signingMethod = jwt.SigningMethodHS256

const OrganizationServiceRoleType = "organization_role"
const DatasetServiceRoleType = "dataset_role"

func GenerateServiceClaim(duration time.Duration) *ServiceClaim {
	issuedTime := jwt.NewNumericDate(time.Now())
	expiresAt := jwt.NewNumericDate(issuedTime.Add(duration))
	return &ServiceClaim{
		Type: authorizer.LabelServiceClaim,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: expiresAt,
			IssuedAt:  issuedTime,
		},
	}
}

type ServiceRole struct {
	Type   string `json:"type"`
	Id     string `json:"id"`
	NodeId string `json:"node_id"`
	Role   string `json:"role"`
}

type ServiceClaim struct {
	Type  string        `json:"type"`
	Roles []ServiceRole `json:"roles"`
	jwt.RegisteredClaims
}

type ServiceToken struct {
	Value string `json:"value"`
}

func (c *ServiceClaim) WithOrganizationClaim(claim *organization.Claim) *ServiceClaim {
	c.Roles = append(c.Roles, ServiceRole{
		Type:   OrganizationServiceRoleType,
		Id:     strconv.FormatInt(claim.IntId, 10),
		NodeId: claim.NodeId,
		Role:   claim.Role.AsRoleString(),
	})
	return c
}

func (c *ServiceClaim) WithDatasetClaim(claim *dataset.Claim) *ServiceClaim {
	c.Roles = append(c.Roles, ServiceRole{
		Type:   DatasetServiceRoleType,
		Id:     strconv.FormatInt(claim.IntId, 10),
		NodeId: claim.NodeId,
		Role:   strings.ToLower(claim.Role.String()),
	})
	return c
}

func (c *ServiceClaim) AsToken(key string) (*ServiceToken, error) {
	var (
		err          error
		secret       []byte
		token        *jwt.Token
		signedString string
	)
	token = jwt.NewWithClaims(signingMethod, c)
	secret = []byte(key)
	signedString, err = token.SignedString(secret)
	if err != nil {
		return nil, err
	}
	return &ServiceToken{Value: signedString}, nil
}

// ParseServiceClaim parses the given tokenString (produced by ServiceClaim.AsToken) and returns the
// extracted ServiceClaim
func ParseServiceClaim(tokenString string, key string) (ServiceClaim, error) {
	var serviceClaim ServiceClaim
	_, err := jwt.ParseWithClaims(tokenString, &serviceClaim, func(token *jwt.Token) (any, error) {
		return []byte(key), nil
	}, jwt.WithValidMethods([]string{signingMethod.Alg()}))
	return serviceClaim, err
}
