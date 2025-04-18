package store

import "github.com/pennsieve/pennsieve-go-core/pkg/models/role"

type CreateCollectionResponse struct {
	ID          int64
	CreatorRole role.Role
}

type CollectionBase struct {
	NodeID      string
	Name        string
	Description string
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

type GetCollectionResponse struct {
	CollectionBase
	DOIs []string
}
