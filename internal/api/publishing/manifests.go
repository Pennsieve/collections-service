package publishing

import "github.com/pennsieve/collections-service/internal/api/apijson"

type FileManifest struct {
	Name            string `json:"name,omitempty"`
	Path            string `json:"path"`
	Size            int64  `json:"size"`
	FileType        string `json:"fileType"`
	SourcePackageId string `json:"sourcePackageId,omitempty"`
	S3VersionId     string `json:"s3VersionId,omitempty"`
	SHA256          string `json:"sha256,omitempty"`
}

type ManifestV5 struct {
	PennsieveDatasetId int64                  `json:"pennsieveDatasetId"`
	Version            int                    `json:"version"`
	Revision           int                    `json:"revision,omitempty"`
	Name               string                 `json:"name,omitempty"`
	Description        string                 `json:"description"`
	Creator            PublishedContributor   `json:"creator"`
	Contributors       []PublishedContributor `json:"contributors"`
	SourceOrganization string                 `json:"sourceOrganization"`
	Keywords           []string               `json:"keywords"`
	DatePublished      apijson.Date           `json:"datePublished"`
	License            string                 `json:"license,omitempty"`
	// ID is the DOI
	ID string `json:"@id"`
	// Publisher is "The University of Pennsylvania"
	Publisher string `json:"publisher"`
	// Context is "http://schema.org/"
	Context string `json:"@context"`
	// Type is "Collection" TODO (or "collection" ?)
	Type string `json:"@type"`
	// SchemaVersion is "http://schema.org/version/3.7/"
	SchemaVersion       string                         `json:"schemaVersion"`
	Collections         []PublishedCollection          `json:"collections,omitempty"`
	RelatedPublications []PublishedExternalPublication `json:"relatedPublications,omitempty"`
	Files               []FileManifest                 `json:"files"`
	References          []string                       `json:"references"`
	// PennsieveSchemaVersion is "5.0"
	PennsieveSchemaVersion string `json:"pennsieveSchemaVersion"`
}

type PublishedContributor struct {
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	Orcid         string `json:"orcid,omitempty"`
	MiddleInitial string `json:"middle_initial,omitempty"`
	Degree        string `json:"degree,omitempty"`
}

type PublishedCollection struct {
	Name string `json:"name"`
}

type PublishedExternalPublication struct {
	DOI              string `json:"doi"`
	RelationshipType string `json:"relationshipType,omitempty"`
}
