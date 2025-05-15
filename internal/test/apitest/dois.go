package apitest

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/datasource"
	"github.com/pennsieve/collections-service/internal/api/store"
)

const ExternalDOIPrefix = "10.9999.9"

func NewExternalDOI() store.DOI {
	return store.DOI{
		Value:      NewDOIWithPrefix(ExternalDOIPrefix),
		Datasource: datasource.External,
	}
}

const PennsieveDOIPrefix = "10.1111"

func NewPennsieveDOI() store.DOI {
	return store.DOI{Value: NewDOIWithPrefix(PennsieveDOIPrefix), Datasource: datasource.Pennsieve}
}

func NewDOIWithPrefix(prefix string) string {
	return fmt.Sprintf("%s/%s", prefix, uuid.NewString())
}
