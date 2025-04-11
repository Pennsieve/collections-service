package dto

import "time"

// CreateCollectionRequest represents the request body of POST /
type CreateCollectionRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	DOIs        []string `json:"dois"`
}

// GetCollectionsResponse represents the response body of GET /
type GetCollectionsResponse struct {
	Limit       int                  `json:"limit"`
	Offset      int                  `json:"offset"`
	TotalCount  int                  `json:"totalCount"`
	Collections []CollectionResponse `json:"collections"`
}

// GetCollectionResponse represents the response body of GET //{id}
type GetCollectionResponse struct {
	CollectionResponse
	Contributors []string        `json:"contributors"`
	Datasets     []PublicDataset `json:"datasets"`
}

// CollectionResponse is a base struct shared by GET / and GET //{id}
type CollectionResponse struct {
	NodeID      string   `json:"nodeId"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Banners     []string `json:"banners"`
	Size        int      `json:"size"`
	UserRole    string   `json:"userRole"`
}

// PublicDataset and it's child DTOs are taken from the Discover service so that
// our responses match those of Discover.
type PublicDataset struct {
	ID                     int64                       `json:"id"`
	SourceDatasetID        *int64                      `json:"sourceDatasetId,omitempty"`
	Name                   string                      `json:"name"`
	Description            string                      `json:"description"`
	OwnerID                *int64                      `json:"ownerId,omitempty"`
	OwnerFirstName         string                      `json:"ownerFirstName"`
	OwnerLastName          string                      `json:"ownerLastName"`
	OwnerOrcid             string                      `json:"ownerOrcid"`
	OrganizationName       string                      `json:"organizationName"`
	OrganizationID         *int64                      `json:"organizationId,omitempty"`
	License                string                      `json:"license"`
	Tags                   []string                    `json:"tags"`
	Version                int                         `json:"version"`
	Revision               *int                        `json:"revision,omitempty"`
	Size                   int64                       `json:"size"`
	ModelCount             []ModelCount                `json:"modelCount"`
	FileCount              int64                       `json:"fileCount"`
	RecordCount            int64                       `json:"recordCount"`
	URI                    string                      `json:"uri"`
	ARN                    string                      `json:"arn"`
	Status                 string                      `json:"status"`
	DOI                    string                      `json:"doi"`
	Banner                 *string                     `json:"banner,omitempty"`
	Readme                 *string                     `json:"readme,omitempty"`
	Changelog              *string                     `json:"changelog,omitempty"`
	Contributors           []PublicContributor         `json:"contributors"`
	Collections            []PublicCollection          `json:"collections,omitempty"`
	ExternalPublications   []PublicExternalPublication `json:"externalPublications,omitempty"`
	Sponsorship            *Sponsorship                `json:"sponsorship,omitempty"`
	PennsieveSchemaVersion *string                     `json:"pennsieveSchemaVersion,omitempty"`
	Embargo                *bool                       `json:"embargo,omitempty"`
	EmbargoReleaseDate     *time.Time                  `json:"embargoReleaseDate,omitempty"`
	EmbargoAccess          *string                     `json:"embargoAccess,omitempty"`
	DatasetType            *string                     `json:"datasetType,omitempty"`
	Release                *ReleaseInfo                `json:"release,omitempty"`
	// CreatedAt is deprecated in favor of FirstPublishedAt or VersionPublishedAt
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt is deprecated in favor of FirstPublishedAt or VersionPublishedAt
	UpdatedAt          time.Time  `json:"updatedAt"`
	FirstPublishedAt   *time.Time `json:"firstPublishedAt,omitempty"`
	VersionPublishedAt *time.Time `json:"versionPublishedAt,omitempty"`
	RevisedAt          *time.Time `json:"revisedAt,omitempty"`
}

type ModelCount struct {
	ModelName string `json:"modelName"`
	Count     int64  `json:"count"`
}

type PublicContributor struct {
	FirstName     string  `json:"firstName"`
	MiddleInitial *string `json:"middleInitial,omitempty"`
	LastName      string  `json:"lastName"`
	Degree        *string `json:"degree,omitempty"`
	Orcid         *string `json:"orcid,omitempty"`
}

type PublicCollection struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type PublicExternalPublication struct {
	DOI              string `json:"doi"`
	RelationshipType string `json:"relationshipType"`
}

type Sponsorship struct {
	Title    string `json:"title"`
	ImageUrl string `json:"imageUrl"`
	Markup   string `json:"markup"`
}

type ReleaseInfo struct {
	Origin        string  `json:"origin"`
	Label         string  `json:"label"`
	Marker        string  `json:"marker"`
	RepoUrl       string  `json:"repoUrl"`
	LabelUrl      *string `json:"labelUrl,omitempty"`
	MarkerUrl     *string `json:"markerUrl,omitempty"`
	ReleaseStatus *string `json:"releaseStatus,omitempty"`
}

type Tombstone struct {
	ID        int64     `json:"id"`
	Version   int       `json:"version"`
	Name      string    `json:"name"`
	Tags      []string  `json:"tags"`
	Status    string    `json:"status"`
	DOI       string    `json:"doi"`
	UpdatedAt time.Time `json:"updatedAt"`
}
