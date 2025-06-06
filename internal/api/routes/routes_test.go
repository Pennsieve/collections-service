package routes

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/config"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"net/http"
	"strings"
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

func TestHandleError(t *testing.T) {
	t.Run("apierror", func(t *testing.T) {
		var logBuffer bytes.Buffer
		h := slog.NewJSONHandler(&logBuffer, nil)
		logger := slog.New(h)

		nodeID := uuid.NewString()
		notFound := apierrors.NewCollectionNotFoundError(nodeID)

		resp, err := handleError(notFound, logger)
		require.NoError(t, err)

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		assert.Equal(t, DefaultErrorResponseHeaders(), resp.Headers)
		assert.Contains(t, resp.Body, nodeID)
		assert.Contains(t, resp.Body, fmt.Sprintf(`"errorId": %q`, notFound.ID))

		// Check that there is exactly one log entry for the error.
		// logger appends newline to each log entry
		log := logBuffer.String()
		errorLog, emptyString, found := strings.Cut(log, "\n")
		assert.True(t, found)
		assert.NotEmpty(t, errorLog)
		assert.Empty(t, emptyString)

		assert.Contains(t, errorLog, fmt.Sprintf(`"msg":"returning API error to caller"`))
		assert.Contains(t, errorLog, fmt.Sprintf(`"id":%q`, notFound.ID))
		assert.Contains(t, errorLog, fmt.Sprintf(`"userMessage":%q`, notFound.UserMessage))
		assert.Contains(t, errorLog, `"cause":"none"`)

	})

	t.Run("not an apierror", func(t *testing.T) {
		var logBuffer bytes.Buffer
		h := slog.NewJSONHandler(&logBuffer, nil)
		logger := slog.New(h)

		nonAPIError := errors.New("unexpected non-apierror error")

		resp, err := handleError(nonAPIError, logger)
		require.NoError(t, err)

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		assert.Equal(t, DefaultErrorResponseHeaders(), resp.Headers)
		assert.Contains(t, resp.Body, `"errorId"`)

		// Check that there is exactly one log entry for the error.
		// logger appends newline to each log entry
		log := logBuffer.String()
		errorLog, emptyString, found := strings.Cut(log, "\n")
		assert.True(t, found)
		assert.NotEmpty(t, errorLog)
		assert.Empty(t, emptyString)

		assert.Contains(t, errorLog, fmt.Sprintf(`"msg":"returning API error to caller"`))
		assert.Contains(t, errorLog, `"id"`)
		assert.Contains(t, errorLog, `"userMessage":"server error"`)
		assert.Contains(t, errorLog, fmt.Sprintf(`"cause":%q`, nonAPIError.Error()))

	})

}
