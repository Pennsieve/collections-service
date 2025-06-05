package service

// Some Discover models, such as PublicDatasetDTO live in api/dto because we use them as DTOs as well.
// Here we just have types that are only used internally by us.

type PublishDOICollectionRequest struct {
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	Banners          []string `json:"banners"` // max 4 items
	DOIs             []string `json:"dois"`    // min 1 item
	License          string   `json:"license"`
	Tags             []string `json:"tags"`
	OwnerID          int32    `json:"ownerId"`
	OwnerNodeID      string   `json:"ownerNodeId"`
	OwnerFirstName   string   `json:"ownerFirstName"`
	OwnerLastName    string   `json:"ownerLastName"`
	OwnerOrcid       string   `json:"ownerOrcid"`
	CollectionNodeID string   `json:"collectionNodeId"`
}
