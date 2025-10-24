package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/shared/util"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
	"log/slog"
	"net/http"
)

type DOI interface {
	GetLatestDOI(ctx context.Context, collectionID int64, collectionNodeID string, userRole role.Role) (dto.GetLatestDOIResponse, error)
}

type HTTPDOI struct {
	InternalService
	url                   string
	collectionNamespaceID int64
	logger                *slog.Logger
}

func NewHTTPDOI(doiServiceURL, jwtSecretKey string, collectionNamespaceID int64, logger *slog.Logger) *HTTPDOI {
	return &HTTPDOI{
		InternalService:       InternalService{jwtSecretKey: jwtSecretKey},
		url:                   doiServiceURL,
		collectionNamespaceID: collectionNamespaceID,
		logger:                logger,
	}
}

func (h *HTTPDOI) GetLatestDOI(ctx context.Context, collectionID int64, collectionNodeID string, userRole role.Role) (dto.GetLatestDOIResponse, error) {
	internalClaims := NewInternalClaims(h.collectionNamespaceID, collectionNodeID, collectionID, userRole)
	requestParams := requestParameters{
		method: http.MethodGet,
		url:    fmt.Sprintf("%s/organizations/%d/datasets/%d/doi", h.url, h.collectionNamespaceID, collectionID),
	}
	response, err := h.InvokePennsieve(ctx, h.logger, internalClaims, requestParams)
	if err != nil {
		var e *util.HTTPError
		switch {
		case errors.As(err, &e):
			if e.StatusCode() == http.StatusNotFound {
				return dto.GetLatestDOIResponse{}, LatestDOINotFoundError{
					ID:     collectionID,
					NodeID: collectionNodeID,
				}
			}
			return dto.GetLatestDOIResponse{}, err
		default:
			return dto.GetLatestDOIResponse{}, err
		}
	}
	defer util.CloseAndWarn(response, h.logger)

	var latestDOI dto.GetLatestDOIResponse
	if err := util.UnmarshallResponse(response, &latestDOI); err != nil {
		return latestDOI, fmt.Errorf("error unmarshalling response to request %s: %w", requestParams, err)
	}
	return latestDOI, nil
}
