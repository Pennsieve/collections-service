package routes

import (
	"github.com/pennsieve/collections-service/internal/api/config"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/stretchr/testify/assert"
	"testing"
)

func assertExpectedEqualCollectionResponse(t *testing.T, expected *apitest.ExpectedCollection, actual dto.CollectionResponse, expectedDatasets *apitest.ExpectedPennsieveDatasets) {
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
