package service

import (
	"encoding/json"
	"github.com/pennsieve/collections-service/internal/api/dto"
)

// Some Discover models, such as PublicDatasetDTO live in api/dto because we use them as DTOs as well.
// Here we just have types that are only used internally by us.

type PublishDOICollectionRequest struct {
	// Required Values

	Name             string                `json:"name"`
	Description      string                `json:"description"`
	Banners          []string              `json:"banners"` // max 4 items
	DOIs             []string              `json:"dois"`    // min 1 item
	OwnerID          int64                 `json:"ownerId"`
	License          string                `json:"license"`
	Contributors     []InternalContributor `json:"contributors"`
	Tags             []string              `json:"tags"`
	OwnerNodeID      string                `json:"ownerNodeId"`
	OwnerFirstName   string                `json:"ownerFirstName"`
	OwnerLastName    string                `json:"ownerLastName"`
	OwnerORCID       string                `json:"ownerOrcid"`
	CollectionNodeID string                `json:"collectionNodeId"`

	// Optional Values have been left out for now. Can be added as they come up
}

func (r PublishDOICollectionRequest) MarshalJSON() ([]byte, error) {
	if r.Banners == nil {
		r.Banners = []string{}
	}
	if r.DOIs == nil {
		r.DOIs = []string{}
	}
	if r.Contributors == nil {
		r.Contributors = []InternalContributor{}
	}
	if r.Tags == nil {
		r.Tags = []string{}
	}
	type alias PublishDOICollectionRequest
	return json.Marshal(alias(r))
}

type InternalContributor struct {
	//Required Values

	// ID is an internal contributor id, different from a user id.
	ID        int64  `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`

	//Optional Values

	ORCID         string `json:"orcid,omitempty"`
	MiddleInitial string `json:"middleInitial,omitempty"`
	Degree        string `json:"degree,omitempty"`
	UserID        int64  `json:"userId,omitempty"`
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
	// All Values Required

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
