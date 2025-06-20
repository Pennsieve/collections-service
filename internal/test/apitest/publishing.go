package apitest

import (
	"github.com/pennsieve/collections-service/internal/api/publishing"
	"github.com/pennsieve/collections-service/internal/test/userstest"
	"github.com/stretchr/testify/require"
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
