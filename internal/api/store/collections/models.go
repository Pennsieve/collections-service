package collections

import (
	"github.com/pennsieve/collections-service/internal/api/datasource"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
)

type CreateCollectionRequest struct {
	NodeID      string
	Name        string
	Description string
	DOIs        []DOI
	UserID      int64
	License     *string
	Tags        []string
}

type CreateCollectionResponse struct {
	ID          int64
	CreatorRole role.Role
}

type CollectionBase struct {
	ID          int64
	NodeID      string
	Name        string
	Description string
	License     *string
	Tags        []string
	Size        int
	UserRole    role.Role
}

type CollectionSummary struct {
	CollectionBase
	BannerDOIs []string
}

type GetCollectionsResponse struct {
	Limit       int
	Offset      int
	Collections []CollectionSummary
	TotalCount  int
}

type DOI struct {
	Value      string
	Datasource datasource.DOIDatasource
}

type DOIs []DOI

func (d DOIs) Strings() []string {
	s := make([]string, len(d))
	for i, doi := range d {
		s[i] = doi.Value
	}
	return s
}

type GetCollectionResponse struct {
	CollectionBase
	DOIs DOIs
}

type DOIUpdate struct {
	Add    []DOI
	Remove []string
}

type UpdateCollectionRequest struct {
	Name        *string
	Description *string
	License     *string
	Tags        []string
	DOIs        DOIUpdate
}
