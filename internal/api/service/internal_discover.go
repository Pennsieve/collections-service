package service

import (
	"encoding/json"
	"fmt"
	"github.com/pennsieve/collections-service/internal/api/service/jwtdiscover"
	"github.com/pennsieve/collections-service/internal/shared/util"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/dataset"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/organization"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// Internal Discover stuff has been separated out into its own service and dependency since it depends
// on an SSM parameter. Trying to avoid looking it up unless we need it, so only if the service actually
// needs to call an internal Discover endpoint.

type InternalDiscover interface {
	PublishCollection(collectionID int64, userRole role.Role, request PublishDOICollectionRequest) (PublishDOICollectionResponse, error)
}

type HTTPInternalDiscover struct {
	host                  string
	jwtSecretKey          string
	collectionNamespaceID int64
	logger                *slog.Logger
}

func NewHTTPInternalDiscover(host, jwtSecretKey string, collectionNamespaceID int64, logger *slog.Logger) *HTTPInternalDiscover {
	return &HTTPInternalDiscover{
		host:                  host,
		jwtSecretKey:          jwtSecretKey,
		collectionNamespaceID: collectionNamespaceID,
		logger:                logger,
	}
}

func (d *HTTPInternalDiscover) PublishCollection(collectionID int64, userRole role.Role, request PublishDOICollectionRequest) (PublishDOICollectionResponse, error) {
	requestURL := fmt.Sprintf("%s/collection/%d/publish", d.host, collectionID)
	collectionClaim := &dataset.Claim{
		Role:   userRole,
		NodeId: request.CollectionNodeID,
		IntId:  collectionID,
	}
	response, err := d.InvokePennsieve(collectionClaim, http.MethodPost, requestURL, request)
	if err != nil {
		return PublishDOICollectionResponse{}, err
	}
	defer util.CloseAndWarn(response, d.logger)

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return PublishDOICollectionResponse{}, fmt.Errorf("error reading response from POST %s: %w", requestURL, err)
	}
	var responseDTO PublishDOICollectionResponse
	if err := json.Unmarshal(body, &responseDTO); err != nil {
		rawResponse := string(body)
		return PublishDOICollectionResponse{}, fmt.Errorf(
			"error unmarshalling response [%s] from POST %s: %w",
			rawResponse,
			requestURL,
			err)
	}
	return responseDTO, nil

}

func (d *HTTPInternalDiscover) InvokePennsieve(collectionClaim *dataset.Claim, method string, url string, structBody any) (*http.Response, error) {
	req, err := newPennsieveRequest(method, url, structBody)
	if err != nil {
		return nil, fmt.Errorf("error creating %s %s request: %w", method, url, err)
	}
	if err := d.addAuth(collectionClaim, req); err != nil {
		return nil, err
	}
	return util.Invoke(req, d.logger)
}

func (d *HTTPInternalDiscover) addAuth(collectionClaim *dataset.Claim, request *http.Request) error {
	serviceClaim := jwtdiscover.GenerateServiceClaim(5 * time.Minute).WithOrganizationClaim(OrganizationClaim(d.collectionNamespaceID)).WithDatasetClaim(collectionClaim)
	token, err := serviceClaim.AsToken(d.jwtSecretKey)
	if err != nil {
		return fmt.Errorf("error creating JWT from service claim: %w", err)
	}

	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token.Value))
	return nil
}

func OrganizationClaim(collectionOrgId int64) *organization.Claim {
	return &organization.Claim{
		Role:  pgdb.Owner,
		IntId: collectionOrgId,
	}
}
