package routes

import (
	"github.com/pennsieve/collections-service/internal/api/config"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/stretchr/testify/assert"
	"testing"
)

func assertExpectedEqualCollectionSummary(t *testing.T, expected *apitest.ExpectedCollection, actual dto.CollectionSummary, expectedDatasets *apitest.ExpectedPennsieveDatasets) {
	t.Helper()
	assert.Equal(t, *expected.NodeID, actual.NodeID)
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.Description, actual.Description)
	assert.Equal(t, expected.Users[0].PermissionBit.ToRole().String(), actual.UserRole)
	assert.Len(t, expected.DOIs, actual.Size)
	bannerLen := min(config.MaxBannersPerCollection, len(expected.DOIs))
	expectedBanners := expectedDatasets.ExpectedBannersForDOIs(t, expected.DOIs.Strings()[:bannerLen])
	assert.Equal(t, expectedBanners, actual.Banners)
}

// assertEqualExpectedGetCollectionResponse makes a number of simplifying assumptions:
// that all the datasets are of type dto.PublicDataset, and so contain no dto.Tombstone
// that all contributors are unique
func assertEqualExpectedGetCollectionResponse(t *testing.T, expected *apitest.ExpectedCollection, actual dto.GetCollectionResponse, expectedDatasets *apitest.ExpectedPennsieveDatasets) {
	t.Helper()
	assertExpectedEqualCollectionSummary(t, expected, actual.CollectionSummary, expectedDatasets)

	if assert.Len(t, actual.Datasets, len(expected.DOIs)) {
		for i := 0; i < len(expected.DOIs); i++ {
			actualDataset := actual.Datasets[i]
			expectedDOI := expected.DOIs[i].DOI
			var actualPublicDataset dto.PublicDataset
			apitest.RequireAsPennsieveDataset(t, actualDataset, &actualPublicDataset)
			assert.Equal(t, expectedDOI, actualPublicDataset.DOI)
			assert.Equal(t, expectedDatasets.DOIToPublicDataset[expectedDOI], actualPublicDataset)
		}
	}
	// there should be no duplicates in the contributors since they contain UUIDs for any strings
	// So it's ok to use results straight from ExpectedContributorsForDOIs
	assert.Equal(t, expectedDatasets.ExpectedContributorsForDOIs(t, expected.DOIs.Strings()), actual.DerivedContributors)

}
