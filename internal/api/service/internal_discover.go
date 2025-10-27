package service

import (
	"context"
	"fmt"
	"github.com/pennsieve/collections-service/internal/shared/util"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
	"log/slog"
	"net/http"
)

// Internal Discover stuff has been separated out into its own service and dependency since it depends
// on an SSM parameter. Trying to avoid looking it up unless we need it, so only if the service actually
// needs to call an internal Discover endpoint.

type InternalDiscover interface {
	PublishCollection(ctx context.Context, collectionID int64, userRole role.Role, request PublishDOICollectionRequest) (PublishDOICollectionResponse, error)
	FinalizeCollectionPublish(ctx context.Context, collectionID int64, collectionNodeID string, userRole role.Role, request FinalizeDOICollectionPublishRequest) (FinalizeDOICollectionPublishResponse, error)
	UnpublishCollection(ctx context.Context, collectionID int64, collectionNodeID string, userRole role.Role) (DatasetPublishStatusResponse, error)
	GetCollectionPublishStatus(ctx context.Context, collectionID int64, collectionNodeID string, userRole role.Role) (DatasetPublishStatusResponse, error)
}

type HTTPInternalDiscover struct {
	InternalService
	url                   string
	collectionNamespaceID int64
	logger                *slog.Logger
}

func NewHTTPInternalDiscover(internalDiscoverURL, jwtSecretKey string, collectionNamespaceID int64, logger *slog.Logger) *HTTPInternalDiscover {
	return &HTTPInternalDiscover{
		InternalService:       InternalService{jwtSecretKey: jwtSecretKey},
		url:                   internalDiscoverURL,
		collectionNamespaceID: collectionNamespaceID,
		logger:                logger,
	}
}

func (d *HTTPInternalDiscover) PublishCollection(ctx context.Context, collectionID int64, userRole role.Role, request PublishDOICollectionRequest) (PublishDOICollectionResponse, error) {
	internalClaims := NewInternalClaims(d.collectionNamespaceID, request.CollectionNodeID, collectionID, userRole)
	requestParams := requestParameters{
		method: http.MethodPost,
		url:    fmt.Sprintf("%s/collection/%d/publish", d.url, collectionID),
		body:   request,
	}
	response, err := d.InvokePennsieve(ctx, d.logger, internalClaims, requestParams)
	if err != nil {
		return PublishDOICollectionResponse{}, err
	}
	defer util.CloseAndWarn(response, d.logger)

	var responseDTO PublishDOICollectionResponse
	if err := util.UnmarshallResponse(response, &responseDTO); err != nil {
		return PublishDOICollectionResponse{}, fmt.Errorf(
			"error unmarshalling response to %s: %w",
			requestParams,
			err)
	}
	return responseDTO, nil

}

func (d *HTTPInternalDiscover) FinalizeCollectionPublish(ctx context.Context, collectionID int64, collectionNodeID string, userRole role.Role, request FinalizeDOICollectionPublishRequest) (FinalizeDOICollectionPublishResponse, error) {
	requestParams := requestParameters{
		method: http.MethodPost,
		url:    fmt.Sprintf("%s/collection/%d/finalize", d.url, collectionID),
		body:   request,
	}

	internalClaims := NewInternalClaims(d.collectionNamespaceID, collectionNodeID, collectionID, userRole)

	response, err := d.InvokePennsieve(ctx, d.logger, internalClaims, requestParams)
	if err != nil {
		return FinalizeDOICollectionPublishResponse{}, err
	}
	defer util.CloseAndWarn(response, d.logger)

	var responseDTO FinalizeDOICollectionPublishResponse
	if err := util.UnmarshallResponse(response, &responseDTO); err != nil {
		return FinalizeDOICollectionPublishResponse{}, fmt.Errorf(
			"error unmarshalling response to %s: %w",
			requestParams,
			err)
	}
	return responseDTO, nil
}

func (d *HTTPInternalDiscover) UnpublishCollection(ctx context.Context, collectionID int64, collectionNodeID string, userRole role.Role) (DatasetPublishStatusResponse, error) {
	requestParams := requestParameters{method: http.MethodPost, url: fmt.Sprintf("%s/collection/%d/unpublish", d.url, collectionID)}

	internalClaims := NewInternalClaims(d.collectionNamespaceID, collectionNodeID, collectionID, userRole)

	response, err := d.InvokePennsieve(ctx, d.logger, internalClaims, requestParams)
	if err != nil {
		return DatasetPublishStatusResponse{}, err
	}
	defer util.CloseAndWarn(response, d.logger)

	// Make sure Discover did not return 204 before trying to read the body.
	if response.StatusCode == http.StatusNoContent {
		return DatasetPublishStatusResponse{}, CollectionNeverPublishedError{
			ID:     collectionID,
			NodeID: collectionNodeID,
		}
	}

	var responseDTO DatasetPublishStatusResponse
	if err := util.UnmarshallResponse(response, &responseDTO); err != nil {
		return DatasetPublishStatusResponse{}, fmt.Errorf(
			"error unmarshalling response to %s: %w",
			requestParams,
			err)
	}
	return responseDTO, nil

}

func (d *HTTPInternalDiscover) GetCollectionPublishStatus(ctx context.Context, collectionID int64, collectionNodeID string, userRole role.Role) (DatasetPublishStatusResponse, error) {
	requestParams := requestParameters{
		method: http.MethodGet,
		url: fmt.Sprintf("%s/organizations/%d/datasets/%d",
			d.url,
			d.collectionNamespaceID,
			collectionID),
	}

	internalClaims := NewInternalClaims(d.collectionNamespaceID, collectionNodeID, collectionID, userRole)

	response, err := d.InvokePennsieve(ctx, d.logger, internalClaims, requestParams)
	if err != nil {
		return DatasetPublishStatusResponse{}, err
	}
	defer util.CloseAndWarn(response, d.logger)

	var responseDTO DatasetPublishStatusResponse
	if err := util.UnmarshallResponse(response, &responseDTO); err != nil {
		return DatasetPublishStatusResponse{}, fmt.Errorf(
			"error unmarshalling response to %s: %w",
			requestParams,
			err)
	}
	return responseDTO, nil
}
