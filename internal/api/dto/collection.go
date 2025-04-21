package dto

import (
	"encoding/json"
	"fmt"
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
type CreateCollectionResponse CollectionResponse

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
	return CollectionResponse(r).MarshalJSON()
}

// GetCollectionsResponse represents the response body of GET /
type GetCollectionsResponse struct {
	Limit       int                  `json:"limit"`
	Offset      int                  `json:"offset"`
	TotalCount  int                  `json:"totalCount"`
	Collections []CollectionResponse `json:"collections"`
}

func (r GetCollectionsResponse) Marshal() (string, error) {
	return defaultMarshalImpl(r)
}

func (r GetCollectionsResponse) MarshalJSON() ([]byte, error) {
	type GetCollectionsResponseAlias GetCollectionsResponse
	if r.Collections == nil {
		r.Collections = []CollectionResponse{}
	}
	return json.Marshal(GetCollectionsResponseAlias(r))
}

// GetCollectionResponse represents the response body of GET /{nodeId}
type GetCollectionResponse struct {
	CollectionResponse
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
	type Alias CollectionResponse
	return json.Marshal(struct {
		Alias
		DerivedContributors []PublicContributor `json:"derivedContributors"`
		Datasets            []Dataset           `json:"datasets"`
	}{
		Alias(r.CollectionResponse),
		r.DerivedContributors,
		r.Datasets,
	})
}

// CollectionResponse is a base struct shared by POST /,  GET /, and GET /{nodeId}
type CollectionResponse struct {
	NodeID      string   `json:"nodeId"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Banners     []string `json:"banners"`
	Size        int      `json:"size"`
	UserRole    string   `json:"userRole"`
}

func (r CollectionResponse) MarshalJSON() ([]byte, error) {
	// I think this is to avoid infinite recursion
	type CollectionResponseAlias CollectionResponse
	if r.Banners == nil {
		r.Banners = []string{}
	}
	return json.Marshal(CollectionResponseAlias(r))
}

type DOIInformationSource string

const PennsieveSource DOIInformationSource = "Pennsieve"
const ExternalSource DOIInformationSource = "External"

type Dataset struct {
	Source  DOIInformationSource `json:"source"`
	Problem bool                 `json:"problem"`
	// Data is the info we got from looking up the DOI.
	// If Source == PennsieveSource AND Problem == false, then Data is a PublicDataset.
	// If Source == PennsieveSource AND Problem == true, then Data is a Tombstone.
	Data json.RawMessage `json:"data"`
}

func NewPennsieveDataset(publicDataset PublicDataset) (Dataset, error) {
	pennsieveBytes, err := json.Marshal(publicDataset)
	if err != nil {
		return Dataset{}, fmt.Errorf("error marshalling PublicDataset %d version %d: %w",
			publicDataset.ID, publicDataset.Version, err)
	}
	return Dataset{
		Source: PennsieveSource,
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
		Source:  PennsieveSource,
		Problem: true,
		Data:    tombstoneBytes,
	}, nil
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
	Collections            []PublicCollection          `json:"collections"`
	ExternalPublications   []PublicExternalPublication `json:"externalPublications"`
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
