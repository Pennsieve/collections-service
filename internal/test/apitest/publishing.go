package apitest

import (
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/publishing"
	"github.com/pennsieve/collections-service/internal/test/userstest"
	"github.com/stretchr/testify/require"
	"math/rand/v2"
	"slices"
)

func ToPublishedContributor(testUser userstest.User) publishing.PublishedContributor {
	orcid := ""
	if orcidAuth := testUser.GetORCIDAuthorization(); orcidAuth != nil {
		orcid = orcidAuth.ORCID
	}
	return publishing.PublishedContributor{
		FirstName:     testUser.GetFirstName(),
		LastName:      testUser.GetLastName(),
		Orcid:         orcid,
		MiddleInitial: testUser.GetMiddleInitial(),
		Degree:        testUser.GetDegree(),
	}
}

func FindManifestEntry(t require.TestingT, manifest publishing.ManifestV5) publishing.FileManifest {
	manifestEntryIdx := slices.IndexFunc(manifest.Files, func(fileManifest publishing.FileManifest) bool {
		return fileManifest.Name == publishing.ManifestFileName &&
			fileManifest.Path == publishing.ManifestFileName &&
			fileManifest.FileType == publishing.ManifestFileType
	})
	require.True(t, manifestEntryIdx >= 0, "no FileManifest found with name and path equal to %s and type equal to %s", publishing.ManifestFileName, publishing.ManifestFileType)
	return manifest.Files[manifestEntryIdx]
}

type ManifestOption func(builder *publishing.ManifestBuilder) *publishing.ManifestBuilder

func WithManifestPennsieveDatasetID(id int64) ManifestOption {
	return func(builder *publishing.ManifestBuilder) *publishing.ManifestBuilder {
		return builder.WithPennsieveDatasetID(id)
	}
}

func WithManifestVersion(datasetVersion int64) ManifestOption {
	return func(builder *publishing.ManifestBuilder) *publishing.ManifestBuilder {
		return builder.WithVersion(datasetVersion)
	}
}

func WithManifestDescription(description string) ManifestOption {
	return func(builder *publishing.ManifestBuilder) *publishing.ManifestBuilder {
		return builder.WithDescription(description)
	}
}

func NewExpectedManifest(t require.TestingT, opts ...ManifestOption) publishing.ManifestV5 {
	builder := publishing.NewManifestBuilder().
		WithPennsieveDatasetID(rand.Int64N(5000) + 1).
		WithVersion(rand.Int64N(20) + 1).
		WithID(NewPennsieveDOI().Value).
		WithName(uuid.NewString()).
		WithDescription(uuid.NewString()).
		WithReferences([]string{
			NewPennsieveDOI().Value,
			NewPennsieveDOI().Value,
			NewPennsieveDOI().Value,
		}).
		WithKeywords([]string{
			uuid.NewString(),
			uuid.NewString(),
		}).
		WithLicense(uuid.NewString()).
		WithCreator(ToPublishedContributor(
			userstest.NewTestUser(
				userstest.WithFirstName(uuid.NewString()),
				userstest.WithLastName(uuid.NewString()),
				userstest.WithDegree(uuid.NewString()[:3]),
				userstest.WithMiddleInitial(uuid.NewString()[:1]),
				userstest.WithORCID(uuid.NewString()),
			)))
	for _, opt := range opts {
		builder = opt(builder)
	}
	manifest, err := builder.Build()
	require.NoError(t, err, "error building expected manifest")
	return manifest
}

func RequireManifestsEqual(t require.TestingT, expected, actual publishing.ManifestV5) {
	// Need this function because apijson.Dates will not work with standard require.Equal(t, expected, actual)
	expectedPublishDate, actualPublishDate := expected.DatePublished, actual.DatePublished
	require.True(t, expectedPublishDate.Equal(actualPublishDate))

	require.Equal(t, expected.Collections, actual.Collections)
	require.Equal(t, expected.Context, actual.Context)
	require.Equal(t, expected.Contributors, actual.Contributors)
	require.Equal(t, expected.Creator, actual.Creator)
	require.Equal(t, expected.Description, actual.Description)
	require.Equal(t, expected.Files, actual.Files)
	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.Keywords, actual.Keywords)
	require.Equal(t, expected.License, actual.License)
	require.Equal(t, expected.Name, actual.Name)
	require.Equal(t, expected.PennsieveDatasetId, actual.PennsieveDatasetId)
	require.Equal(t, expected.PennsieveSchemaVersion, actual.PennsieveSchemaVersion)
	require.Equal(t, expected.Publisher, actual.Publisher)
	require.Equal(t, expected.References, actual.References)
	require.Equal(t, expected.RelatedPublications, actual.RelatedPublications)
	require.Equal(t, expected.Revision, actual.Revision)
	require.Equal(t, expected.SchemaVersion, actual.SchemaVersion)
	require.Equal(t, expected.SourceOrganization, actual.SourceOrganization)
	require.Equal(t, expected.Type, actual.Type)
	require.Equal(t, expected.Version, actual.Version)
}
