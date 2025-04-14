package store

import "github.com/pennsieve/pennsieve-go-core/pkg/models/role"

type CreateCollectionResponse struct {
	ID          int64
	CreatorRole role.Role
}

type CollectionResponse struct {
	NodeID      string
	Name        string
	Description string
	Size        int
	BannerDOIs  []string
	UserRole    string
}

type GetCollectionsResponse struct {
	Limit       int
	Offset      int
	Collections []CollectionResponse
	TotalCount  int
}

type GetCollectionResponse struct {
	CollectionResponse
	DOIs []string
}
