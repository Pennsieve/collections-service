package publishing

import (
	"encoding/json"
	"fmt"
	"github.com/pennsieve/collections-service/internal/api/apijson"
	"slices"
	"strconv"
	"time"
)

type FileManifest struct {
	Name            string `json:"name,omitempty"`
	Path            string `json:"path"`
	Size            int64  `json:"size"`
	FileType        string `json:"fileType"`
	SourcePackageId string `json:"sourcePackageId,omitempty"`
	S3VersionId     string `json:"s3VersionId,omitempty"`
	SHA256          string `json:"sha256,omitempty"`
}

const ManifestFileName = "manifest.json"
const ManifestFileType = "Json"

const ManifestPublisher = "The University of Pennsylvania"
const ManifestContext = "http://schema.org/"
const ManifestType = "Collection"
const ManifestSchemaVersion = "http://schema.org/version/3.7/"
const ManifestPennsieveSchemaVersion = "5.0"

type ManifestV5 struct {
	PennsieveDatasetId int64                  `json:"pennsieveDatasetId"`
	Version            int64                  `json:"version"`
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
	// Type is "Collection"
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

func (m ManifestV5) Marshal() ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}

func (m ManifestV5) S3Key() string {
	return S3Key(m.PennsieveDatasetId)
}

func S3Key(publishedDatasetID int64) string {
	return fmt.Sprintf("%d/%s", publishedDatasetID, ManifestFileName)
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

type ManifestBuilder struct {
	m *ManifestV5
}

func NewManifestBuilder() *ManifestBuilder {
	return &ManifestBuilder{m: &ManifestV5{
		DatePublished:          apijson.Date(time.Now().UTC()),
		Publisher:              ManifestPublisher,
		Context:                ManifestContext,
		Type:                   ManifestType,
		SchemaVersion:          ManifestSchemaVersion,
		PennsieveSchemaVersion: ManifestPennsieveSchemaVersion,
	}}
}

func (b *ManifestBuilder) WithPennsieveDatasetID(id int64) *ManifestBuilder {
	b.m.PennsieveDatasetId = id
	return b
}

func (b *ManifestBuilder) WithVersion(version int64) *ManifestBuilder {
	b.m.Version = version
	return b
}

func (b *ManifestBuilder) WithName(name string) *ManifestBuilder {
	b.m.Name = name
	return b
}

func (b *ManifestBuilder) WithDescription(description string) *ManifestBuilder {
	b.m.Description = description
	return b
}

func (b *ManifestBuilder) WithCreator(publicContributor PublishedContributor) *ManifestBuilder {
	b.m.Creator = publicContributor
	b.m.Contributors = append(b.m.Contributors, publicContributor)
	return b
}

func (b *ManifestBuilder) WithKeywords(keywords []string) *ManifestBuilder {
	b.m.Keywords = append(b.m.Keywords, keywords...)
	return b
}

func (b *ManifestBuilder) WithLicense(license string) *ManifestBuilder {
	b.m.License = license
	return b
}

func (b *ManifestBuilder) WithID(doi string) *ManifestBuilder {
	b.m.ID = doi
	return b
}

func (b *ManifestBuilder) WithReferences(dois []string) *ManifestBuilder {
	b.m.References = append(b.m.References, dois...)
	return b
}

func (b *ManifestBuilder) Build() (ManifestV5, error) {
	manifestEntry := FileManifest{
		Name:     ManifestFileName,
		Path:     ManifestFileName,
		Size:     0,
		FileType: ManifestFileType,
	}
	b.m.Files = append(b.m.Files, manifestEntry)

	if b.m.References == nil {
		b.m.References = make([]string, 0)
	}
	if b.m.Keywords == nil {
		b.m.Keywords = make([]string, 0)
	}
	if b.m.Contributors == nil {
		b.m.Contributors = make([]PublishedContributor, 0)
	}

	jsonBytesWithZeroSize, err := b.m.Marshal()
	if err != nil {
		return ManifestV5{}, fmt.Errorf("error marshalling manifest to calculate size: %w", err)
	}
	// subtract one byte for length of 0 character
	lenWithoutSize := int64(len(jsonBytesWithZeroSize) - 1)
	lenOfSize := numberOfDigits(lenWithoutSize)
	size := lenWithoutSize + lenOfSize
	// needs adjustment if we are at a power of 10
	if numberOfDigits(size) > lenOfSize {
		size += 1
	}

	// This is overkill at the moment when the manifest is the only element in slice, but
	// the assumption is that we will eventually add more files
	manifestIndex := slices.Index(b.m.Files, manifestEntry)
	b.m.Files[manifestIndex].Size = size

	return *b.m, nil
}

func numberOfDigits(size int64) int64 {
	return int64(len(strconv.FormatInt(size, 10)))
}
