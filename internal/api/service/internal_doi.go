package service

import (
	"log/slog"
)

type DOI interface {
}

type HTTPDOI struct {
	InternalService
	host                  string
	collectionNamespaceID int64
	logger                *slog.Logger
}

func NewHTTPDOI(host, jwtSecretKey string, collectionNamespaceID int64, logger *slog.Logger) *HTTPDOI {
	return &HTTPDOI{
		InternalService:       InternalService{jwtSecretKey: jwtSecretKey},
		host:                  host,
		collectionNamespaceID: collectionNamespaceID,
		logger:                logger,
	}
}
