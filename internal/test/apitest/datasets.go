package apitest

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/datasource"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/mocks"
	"github.com/stretchr/testify/require"
	"math/rand/v2"
	"time"
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

type PublicDatasetOption func(publicDataset *dto.PublicDataset)

func WithNilBanner() PublicDatasetOption {
	return func(publicDataset *dto.PublicDataset) {
		publicDataset.Banner = nil
	}
}

func WithPublicContributors(contributors ...dto.PublicContributor) PublicDatasetOption {
	return func(publicDataset *dto.PublicDataset) {
		publicDataset.Contributors = append(publicDataset.Contributors, contributors...)
	}
}

func WithDatasetType(datasetType string) PublicDatasetOption {
	return func(publicDataset *dto.PublicDataset) {
		publicDataset.DatasetType = &datasetType
	}
}

func (e *ExpectedPennsieveDatasets) NewPublishedWithOptions(opts ...PublicDatasetOption) dto.PublicDataset {
	doi := NewPennsieveDOI()
	banner := NewBanner()
	publicDataset := NewPublicDataset(doi.Value, banner)
	for _, opt := range opts {
		opt(&publicDataset)
	}
	e.DOIToPublicDataset[doi.Value] = publicDataset
	return publicDataset
}

func (e *ExpectedPennsieveDatasets) NewPublished(contributors ...dto.PublicContributor) dto.PublicDataset {
	return e.NewPublishedWithOptions(WithPublicContributors(contributors...))
}

func (e *ExpectedPennsieveDatasets) NewPublishedWithNilBanner(contributors ...dto.PublicContributor) dto.PublicDataset {
	return e.NewPublishedWithOptions(WithNilBanner(), WithPublicContributors(contributors...))
}

func (e *ExpectedPennsieveDatasets) NewUnpublished() dto.Tombstone {
	doi := NewPennsieveDOI()
	status := uuid.NewString()
	tombstone := NewTombstone(doi.Value, status)
	e.DOIToTombstone[doi.Value] = tombstone
	return tombstone
}

func (e *ExpectedPennsieveDatasets) ExpectedBannersForDOIs(t require.TestingT, expectedDOIs []string) []string {
	test.Helper(t)
	var expectedBanners []string
	for _, doi := range expectedDOIs {
		if _, published := e.DOIToPublicDataset[doi]; published {
			expectedBanners = append(expectedBanners, e.ExpectedBannerForDOI(t, doi))
		}
	}
	return expectedBanners
}

func (e *ExpectedPennsieveDatasets) ExpectedBannerForDOI(t require.TestingT, expectedDOI string) string {
	test.Helper(t)
	require.Contains(t, e.DOIToPublicDataset, expectedDOI, "no expected published dataset has DOI %s", expectedDOI)
	if bannerOpt := e.DOIToPublicDataset[expectedDOI].Banner; bannerOpt != nil {
		return *bannerOpt
	}
	return ""
}

func (e *ExpectedPennsieveDatasets) ExpectedContributorsForDOI(t require.TestingT, expectedDOI string) []dto.PublicContributor {
	test.Helper(t)
	require.Contains(t, e.DOIToPublicDataset, expectedDOI, "no expected published dataset has DOI %s", expectedDOI)
	return e.DOIToPublicDataset[expectedDOI].Contributors
}

// ExpectedContributorsForDOIs does NOT de-deduplicate the contributors!
func (e *ExpectedPennsieveDatasets) ExpectedContributorsForDOIs(t require.TestingT, expectedDOIs []string) []dto.PublicContributor {
	test.Helper(t)
	var expectedContributors []dto.PublicContributor
	for _, doi := range expectedDOIs {
		expectedContributors = append(expectedContributors, e.ExpectedContributorsForDOI(t, doi)...)
	}
	return expectedContributors
}

func (e *ExpectedPennsieveDatasets) GetDatasetsByDOIFunc(t require.TestingT) mocks.GetDatasetsByDOIFunc {
	return func(ctx context.Context, dois []string) (service.DatasetsByDOIResponse, error) {
		test.Helper(t)
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
		sleepMillis := rand.N(10) + 150
		time.Sleep(time.Duration(sleepMillis) * time.Millisecond)
		return response, nil
	}
}

// RequireAsPennsieveDataset will unmarshall actualDataset.Data into publicDataset if it can. If it cannot, it
// will fail the test.
func RequireAsPennsieveDataset(t require.TestingT, actualDataset dto.Dataset, publicDataset *dto.PublicDataset) {
	test.Helper(t)
	require.Equal(t, datasource.Pennsieve, actualDataset.Source)
	require.False(t, actualDataset.Problem)
	require.NoError(t, json.Unmarshal(actualDataset.Data, publicDataset))
}

// RequireAsPennsieveTombstone will unmarshall actualDataset.Data into tombstone if it can. If it cannot, it
// will fail the test.
func RequireAsPennsieveTombstone(t require.TestingT, actualDataset dto.Dataset, tombstone *dto.Tombstone) {
	test.Helper(t)
	require.Equal(t, datasource.Pennsieve, actualDataset.Source)
	require.True(t, actualDataset.Problem)
	require.NoError(t, json.Unmarshal(actualDataset.Data, tombstone))
}
