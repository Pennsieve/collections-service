package dto

import (
	"encoding/json"
	"fmt"
	"github.com/pennsieve/collections-service/internal/api/apijson"
	"github.com/pennsieve/collections-service/internal/api/datasource"
	"github.com/pennsieve/collections-service/internal/api/publishing"
	"time"
)

// Response types that contain slices implement json.Marshaler only
// so that nil slices are serialized as '"[]"' rather than 'null'.
// Thought this would be easier than always ensuring that slices were
// initialized to non-nil values.

// CreateCollectionRequest represents the request body of POST /
type CreateCollectionRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	DOIs        []string `json:"dois"`
}

// CreateCollectionResponse represents the response body of POST /
type CreateCollectionResponse CollectionSummary

// PatchCollectionRequest represents the request body of PATCH /collections/{nodeId}
type PatchCollectionRequest struct {
	Name        *string    `json:"name,omitempty"`
	Description *string    `json:"description,omitempty"`
	DOIs        *PatchDOIs `json:"dois,omitempty"`
}

type PatchDOIs struct {
	Remove []string `json:"remove,omitempty"`
	Add    []string `json:"add,omitempty"`
}

func (r CreateCollectionResponse) Marshal() (string, error) {
	return defaultMarshalImpl(r)
}

func (r CreateCollectionResponse) MarshalJSON() ([]byte, error) {
	return CollectionSummary(r).MarshalJSON()
}

// GetCollectionsResponse represents the response body of GET /
type GetCollectionsResponse struct {
	Limit       int                 `json:"limit"`
	Offset      int                 `json:"offset"`
	TotalCount  int                 `json:"totalCount"`
	Collections []CollectionSummary `json:"collections"`
}

func (r GetCollectionsResponse) Marshal() (string, error) {
	return defaultMarshalImpl(r)
}

func (r GetCollectionsResponse) MarshalJSON() ([]byte, error) {
	type GetCollectionsResponseAlias GetCollectionsResponse
	if r.Collections == nil {
		r.Collections = []CollectionSummary{}
	}
	return json.Marshal(GetCollectionsResponseAlias(r))
}

// GetCollectionResponse represents the response body of GET /{nodeId}
type GetCollectionResponse struct {
	CollectionSummary
	DerivedContributors []PublicContributor `json:"derivedContributors"`
	Datasets            []Dataset           `json:"datasets"`
}

func (r GetCollectionResponse) Marshal() (string, error) {
	return defaultMarshalImpl(r)
}

// MarshalJSON is implemented so that nil slices get marshalled as [] instead of null.
// The subtleties of embedded structs with added fields and JSON marshalling has complicated
// the implementation
func (r GetCollectionResponse) MarshalJSON() ([]byte, error) {
	if r.Banners == nil {
		r.Banners = []string{}
	}
	if r.DerivedContributors == nil {
		r.DerivedContributors = []PublicContributor{}
	}
	if r.Datasets == nil {
		r.Datasets = []Dataset{}
	}
	type Alias CollectionSummary
	return json.Marshal(struct {
		Alias
		DerivedContributors []PublicContributor `json:"derivedContributors"`
		Datasets            []Dataset           `json:"datasets"`
	}{
		Alias(r.CollectionSummary),
		r.DerivedContributors,
		r.Datasets,
	})
}

// PublishedDataset contains publishing info obtained from Discover.
// Only included in responses to requests that use the includePublishedDataset query param
type PublishedDataset struct {
	ID                int32      `json:"id,omitempty"`
	Version           int32      `json:"version"`
	LastPublishedDate *time.Time `json:"lastPublishedDate,omitempty"`
}

type Publication struct {
	Status           publishing.Status `json:"status"`
	Type             publishing.Type   `json:"type,omitempty"`
	PublishedDataset *PublishedDataset `json:"publishedDataset,omitempty"`
}

// CollectionSummary is a base struct shared by POST /,  GET /,  GET /{nodeId}, and PATCH /{nodeId} responses.
// Publication is never returned by POST /
type CollectionSummary struct {
	NodeID      string       `json:"nodeId"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Banners     []string     `json:"banners"`
	Size        int          `json:"size"`
	UserRole    string       `json:"userRole"`
	Publication *Publication `json:"publication,omitempty"`
}

func (r CollectionSummary) MarshalJSON() ([]byte, error) {
	// I think this is to avoid infinite recursion
	type CollectionSummaryAlias CollectionSummary
	if r.Banners == nil {
		r.Banners = []string{}
	}
	return json.Marshal(CollectionSummaryAlias(r))
}

type Dataset struct {
	Source  datasource.DOIDatasource `json:"source"`
	Problem bool                     `json:"problem"`
	// Data is the info we got from looking up the DOI.
	// If Source == Pennsieve AND Problem == false, then Data is a PublicDataset.
	// If Source == Pennsieve AND Problem == true, then Data is a Tombstone.
	Data json.RawMessage `json:"data"`
}

func NewPennsieveDataset(publicDataset PublicDataset) (Dataset, error) {
	pennsieveBytes, err := json.Marshal(publicDataset)
	if err != nil {
		return Dataset{}, fmt.Errorf("error marshalling PublicDataset %d version %d: %w",
			publicDataset.ID, publicDataset.Version, err)
	}
	return Dataset{
		Source: datasource.Pennsieve,
		Data:   pennsieveBytes,
	}, nil
}

func NewTombstoneDataset(tombstone Tombstone) (Dataset, error) {
	tombstoneBytes, err := json.Marshal(tombstone)
	if err != nil {
		return Dataset{}, fmt.Errorf("error marshalling Tombstone: %d version %d: %w",
			tombstone.ID, tombstone.Version, err)
	}
	return Dataset{
		Source:  datasource.Pennsieve,
		Problem: true,
		Data:    tombstoneBytes,
	}, nil
}

const CollectionDatasetType = "collection"

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
	Collections            []PublicCollection          `json:"collections"`
	ExternalPublications   []PublicExternalPublication `json:"externalPublications"`
	Sponsorship            *Sponsorship                `json:"sponsorship,omitempty"`
	PennsieveSchemaVersion *string                     `json:"pennsieveSchemaVersion,omitempty"`
	Embargo                *bool                       `json:"embargo,omitempty"`
	EmbargoReleaseDate     *apijson.Date               `json:"embargoReleaseDate,omitempty"`
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

func (p PublicDataset) MarshalJSON() ([]byte, error) {
	type PublicDatasetAlias PublicDataset
	if p.Tags == nil {
		p.Tags = []string{}
	}
	if p.ModelCount == nil {
		p.ModelCount = []ModelCount{}
	}
	if p.Contributors == nil {
		p.Contributors = []PublicContributor{}
	}
	if p.Collections == nil {
		p.Collections = []PublicCollection{}
	}
	if p.ExternalPublications == nil {
		p.ExternalPublications = []PublicExternalPublication{}
	}
	return json.Marshal(PublicDatasetAlias(p))
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

func (t Tombstone) MarshalJSON() ([]byte, error) {
	type TombstoneAlias Tombstone
	if t.Tags == nil {
		t.Tags = []string{}
	}
	return json.Marshal(TombstoneAlias(t))
}

type PublishCollectionRequest struct {
	License string   `json:"license"`
	Tags    []string `json:"tags"`
}

// PublishStatus values here are copied from com.pennsieve.models.PublishStatus in pennsieve-api core-models
// A lot of these don't apply to collections.
type PublishStatus string

const NotPublished PublishStatus = "NOT_PUBLISHED"

const PublishInProgress PublishStatus = "PUBLISH_IN_PROGRESS"
const PublishSucceeded PublishStatus = "PUBLISH_SUCCEEDED"
const PublishFailed PublishStatus = "PUBLISH_FAILED"

const EmbargoInProgress PublishStatus = "EMBARGO_IN_PROGRESS"
const EmbargoSucceeded PublishStatus = "EMBARGO_SUCCEEDED"
const EmbargoFailed PublishStatus = "EMBARGO_FAILED"

const ReleaseInProgress PublishStatus = "RELEASE_IN_PROGRESS"
const ReleaseFailed PublishStatus = "RELEASE_FAILED"

const Unpublished PublishStatus = "UNPUBLISHED"

func (ps PublishStatus) ToPublishingStatus() publishing.Status {
	switch ps {
	case PublishSucceeded, Unpublished:
		return publishing.CompletedStatus
	case PublishFailed:
		return publishing.FailedStatus
		// Don't think we should see any other PublishStatus than those listed in the first two cases.
		// Fail if we get an unexpected PublishStatus
	default:
		return publishing.FailedStatus
	}

}

type PublishCollectionResponse struct {
	PublishedDatasetID int64         `json:"publishedDatasetId"`
	PublishedVersion   int64         `json:"publishedVersion"`
	Status             PublishStatus `json:"status"`
}

func (r PublishCollectionResponse) Marshal() (string, error) {
	return defaultMarshalImpl(r)
}
