package service

import "github.com/pennsieve/collections-service/internal/api/dto"

// Some Discover models, such as PublicDatasetDTO live in api/dto because we use them as DTOs as well.
// Here we just have types that are only used internally by us.

type PublishDOICollectionRequest struct {
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	Banners          []string `json:"banners"` // max 4 items
	DOIs             []string `json:"dois"`    // min 1 item
	License          string   `json:"license"`
	Tags             []string `json:"tags"`
	OwnerID          int64    `json:"ownerId"`
	OwnerNodeID      string   `json:"ownerNodeId"`
	OwnerFirstName   string   `json:"ownerFirstName"`
	OwnerLastName    string   `json:"ownerLastName"`
	OwnerORCID       string   `json:"ownerOrcid"`
	CollectionNodeID string   `json:"collectionNodeId"`
}

type PublishDOICollectionResponse struct {
	Name               string            `json:"name"`
	SourceCollectionID int64             `json:"sourceCollectionId"`
	PublishedDatasetID int64             `json:"publishedDatasetId"`
	PublishedVersion   int64             `json:"publishedVersion"`
	Status             dto.PublishStatus `json:"status"`
	PublicID           string            `json:"publicId"`
}

type FinalizeDOICollectionPublishRequest struct {
	PublishedDatasetID int64  `json:"publishedDatasetId"`
	PublishedVersion   int64  `json:"publishedVersion"`
	PublishSuccess     bool   `json:"publishSuccess"`
	FileCount          int    `json:"fileCount"`
	TotalSize          int64  `json:"totalSize"`
	ManifestKey        string `json:"manifestKey"`
	ManifestVersionID  string `json:"manifestVersionId"`
}

type FinalizeDOICollectionPublishResponse struct {
	Status dto.PublishStatus `json:"status"`
}
