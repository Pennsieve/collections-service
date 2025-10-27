package service

import (
	"context"
	"fmt"
	"github.com/pennsieve/collections-service/internal/api/service/jwtdiscover"
	"github.com/pennsieve/collections-service/internal/shared/util"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/dataset"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/organization"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
	"log/slog"
	"net/http"
	"time"
)

type InternalService struct {
	jwtSecretKey string
}

type InternalClaims struct {
	organizationClaim *organization.Claim
	datasetClaim      *dataset.Claim
}

func NewInternalClaims(orgID int64, datasetNodeID string, datasetID int64, userRole role.Role) InternalClaims {
	return InternalClaims{
		organizationClaim: &organization.Claim{
			Role:  pgdb.Owner,
			IntId: orgID,
		},
		datasetClaim: &dataset.Claim{
			Role:   userRole,
			NodeId: datasetNodeID,
			IntId:  datasetID,
		},
	}
}

func (c InternalClaims) ServiceClaim(duration time.Duration) *jwtdiscover.ServiceClaim {
	return jwtdiscover.GenerateServiceClaim(duration).WithOrganizationClaim(c.organizationClaim).WithDatasetClaim(c.datasetClaim)
}

func (d *InternalService) InvokePennsieve(ctx context.Context, logger *slog.Logger, internalClaims InternalClaims, requestParams requestParameters) (*http.Response, error) {
	req, err := newPennsieveRequest(ctx, requestParams)
	if err != nil {
		return nil, fmt.Errorf("error creating %s request: %w", requestParams, err)
	}
	if err := d.addAuth(internalClaims, req); err != nil {
		return nil, err
	}
	return util.Invoke(req, logger)
}

func (d *InternalService) addAuth(internalClaims InternalClaims, request *http.Request) error {
	serviceClaim := internalClaims.ServiceClaim(5 * time.Minute)
	token, err := serviceClaim.AsToken(d.jwtSecretKey)
	if err != nil {
		return fmt.Errorf("error creating JWT from service claim: %w", err)
	}

	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token.Value))
	return nil
}
