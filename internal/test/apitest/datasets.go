package apitest

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/collections-service/internal/test/mocks"
	"github.com/stretchr/testify/require"
)

// ExpectedPennsieveDatasets are the datasets expected to exist in Discover for a given test
// Add Datasets with NewPublished or NewUnpublished.
// Retrieve expected dataset properties with Expected* methods
// Turn into a mocks.GetDatasetsByDOIFunc with GetDatasetsByDOIFunc
type ExpectedPennsieveDatasets struct {
	DOIToPublicDataset map[string]dto.PublicDataset
	DOIToTombstone     map[string]dto.Tombstone
}

func NewExpectedPennsieveDatasets() *ExpectedPennsieveDatasets {
	return &ExpectedPennsieveDatasets{
		DOIToPublicDataset: make(map[string]dto.PublicDataset),
		DOIToTombstone:     make(map[string]dto.Tombstone),
	}
}

func (e *ExpectedPennsieveDatasets) NewPublished(contributors ...dto.PublicContributor) dto.PublicDataset {
	doi := NewPennsieveDOI()
	banner := NewBanner()
	published := NewPublicDataset(doi, banner, contributors...)
	e.DOIToPublicDataset[doi] = published
	return published
}

func (e *ExpectedPennsieveDatasets) NewUnpublished() dto.Tombstone {
	doi := NewPennsieveDOI()
	status := uuid.NewString()
	tombstone := NewTombstone(doi, status)
	e.DOIToTombstone[doi] = tombstone
	return tombstone
}

func (e *ExpectedPennsieveDatasets) ExpectedBannersForDOIs(t require.TestingT, expectedDOIs []string) []string {
	var expectedBanners []string
	for _, doi := range expectedDOIs {
		expectedBanners = append(expectedBanners, e.ExpectedBannerForDOI(t, doi))
	}
	return expectedBanners
}

func (e *ExpectedPennsieveDatasets) ExpectedBannerForDOI(t require.TestingT, expectedDOI string) string {
	require.Contains(t, e.DOIToPublicDataset, expectedDOI, "no expected published dataset has DOI %s", expectedDOI)
	if bannerOpt := e.DOIToPublicDataset[expectedDOI].Banner; bannerOpt != nil {
		return *bannerOpt
	}
	return ""
}

func (e *ExpectedPennsieveDatasets) ExpectedContributorsForDOI(t require.TestingT, expectedDOI string) []dto.PublicContributor {
	require.Contains(t, e.DOIToPublicDataset, expectedDOI, "no expected published dataset has DOI %s", expectedDOI)
	return e.DOIToPublicDataset[expectedDOI].Contributors
}

// ExpectedContributorsForDOIs does NOT de-deduplicate the contributors!
func (e *ExpectedPennsieveDatasets) ExpectedContributorsForDOIs(t require.TestingT, expectedDOIs []string) []dto.PublicContributor {
	var expectedContributors []dto.PublicContributor
	for _, doi := range expectedDOIs {
		expectedContributors = append(expectedContributors, e.ExpectedContributorsForDOI(t, doi)...)
	}
	return expectedContributors
}

func (e *ExpectedPennsieveDatasets) GetDatasetsByDOIFunc(t require.TestingT) mocks.GetDatasetsByDOIFunc {
	return func(dois []string) (service.DatasetsByDOIResponse, error) {
		response := service.DatasetsByDOIResponse{
			Published:   map[string]dto.PublicDataset{},
			Unpublished: map[string]dto.Tombstone{},
		}
		for _, doi := range dois {
			if publicDataset, published := e.DOIToPublicDataset[doi]; published {
				response.Published[doi] = publicDataset
			} else if tombstone, unpublished := e.DOIToTombstone[doi]; unpublished {
				response.Unpublished[doi] = tombstone
			} else {
				require.FailNow(t, "requested DOI not found", "DOI %s is not expected as Published or Unpublished", doi)
			}
		}
		return response, nil
	}
}

func RequireAsPennsieveDataset(t require.TestingT, actualDataset dto.Dataset, publicDataset *dto.PublicDataset) {
	require.Equal(t, dto.PennsieveSource, actualDataset.Source)
	require.NoError(t, json.Unmarshal(actualDataset.Data, publicDataset))
}
