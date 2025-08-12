package service

import (
	"encoding/binary"
	"encoding/json"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"hash"
	"hash/fnv"
	"time"
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
	// It gets stored in an integer column in Postgres, so only 32 bits
	ID        int32  `json:"id"`
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

type InternalContributorBuilder struct {
	c    *InternalContributor
	hash hash.Hash32
}

func NewInternalContributorBuilder() *InternalContributorBuilder {
	return &InternalContributorBuilder{
		c:    &InternalContributor{},
		hash: fnv.New32a(),
	}
}

func (b *InternalContributorBuilder) WithFirstName(firstName string) *InternalContributorBuilder {
	b.c.FirstName = firstName
	return b
}

func (b *InternalContributorBuilder) WithLastName(lastName string) *InternalContributorBuilder {
	b.c.LastName = lastName
	return b
}

func (b *InternalContributorBuilder) WithMiddleInitial(middleInitial string) *InternalContributorBuilder {
	b.c.MiddleInitial = middleInitial
	return b
}

func (b *InternalContributorBuilder) WithORCID(orcid string) *InternalContributorBuilder {
	b.c.ORCID = orcid
	return b
}

func (b *InternalContributorBuilder) WithDegree(degree string) *InternalContributorBuilder {
	b.c.Degree = degree
	return b
}

func (b *InternalContributorBuilder) WithUserID(userID int64) *InternalContributorBuilder {
	b.c.UserID = userID
	return b
}

func writeString(h hash.Hash32, label, val string) {
	//Hash Writes never return an error
	_, _ = h.Write([]byte(label))
	_, _ = h.Write([]byte{0}) // separator
	_, _ = h.Write([]byte(val))
	_, _ = h.Write([]byte{0}) // separator
}

func writeInt64(h hash.Hash32, label string, val int64) {
	//Hash Writes never return an error
	_, _ = h.Write([]byte(label))
	_, _ = h.Write([]byte{0}) // separator
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], uint64(val))
	_, _ = h.Write(buf[:])
	_, _ = h.Write([]byte{0})
}

// Build returns an InternalContributor with an ID calculated as the hash of
// the other fields. Trying this since we don't have a contributors table with ids
// like workspaces do.
func (b *InternalContributorBuilder) Build() InternalContributor {
	// Keep the order the same to return consistent IDs.
	writeString(b.hash, "FirstName", b.c.FirstName)
	writeString(b.hash, "LastName", b.c.LastName)

	writeString(b.hash, "ORCID", b.c.ORCID)
	writeString(b.hash, "MiddleInitial", b.c.MiddleInitial)
	writeString(b.hash, "Degree", b.c.Degree)
	writeInt64(b.hash, "UserID", b.c.UserID)

	// masking the high bit to get a positive number.
	// idea is that the number of contributors will always
	// be small enough that this does not increase the collision risk.
	b.c.ID = int32(b.hash.Sum32() & 0x7FFFFFFF)
	return *b.c
}

type DatasetPublishStatusResponse struct {
	/*	required:
		- name
		- sourceOrganizationId
		- sourceDatasetId
		- publishedVersionCount
		- status
		- workflowId
	*/
	Name                  string            `json:"name"`
	SourceOrganizationID  int32             `json:"sourceOrganizationId"`
	SourceDatasetID       int32             `json:"sourceDatasetId"`
	PublishedDatasetID    int32             `json:"publishedDatasetId,omitempty"`
	PublishedVersionCount int32             `json:"publishedVersionCount"`
	Status                dto.PublishStatus `json:"status"`
	LastPublishedDate     *time.Time        `json:"lastPublishedDate,omitempty"`
	/* fields not included since we don't reference them for now:
	sponsorship:
		$ref: "#/components/schemas/SponsorshipRequest"
	workflowId:
		type: integer
		format: int64
	*/
}
