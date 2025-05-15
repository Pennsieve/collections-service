package dto_test

import (
	"encoding/json"
	"fmt"
	"github.com/pennsieve/collections-service/internal/api/datasource"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestGetCollectionResponse_MarshalJSON(t *testing.T) {
	banner := apitest.NewBanner()
	contributor := apitest.NewPublicContributor()
	externalData := `{"name":"external name","externalProp":"external value"}`
	tests := []struct {
		scenario    string
		value       dto.GetCollectionResponse
		contains    []string
		notContains []string
	}{
		{"empty collection has empty arrays not null",
			dto.GetCollectionResponse{
				CollectionSummary: apitest.NewCollectionResponse(0),
			},
			[]string{`"banners":[]`, `"derivedContributors":[]`, `"datasets":[]`},
			[]string{`"banners":null`, `"derivedContributors":null`, `"datasets":null`},
		},
		{"collection contains contributor and dataset",
			dto.GetCollectionResponse{
				CollectionSummary:   apitest.NewCollectionResponse(1, *banner),
				DerivedContributors: []dto.PublicContributor{contributor},
				Datasets:            []dto.Dataset{{Source: datasource.External, Data: []byte(externalData)}},
			},
			[]string{
				fmt.Sprintf(`"banners":[%q]`, *banner),
				`"derivedContributors":[{`,
				fmt.Sprintf(`"source":%q`, datasource.External),
				fmt.Sprintf(`"data":%s`, externalData),
			},
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.scenario, func(t *testing.T) {
			jsonBytes, err := json.Marshal(tt.value)
			require.NoError(t, err)
			jsonString := string(jsonBytes)
			for _, contain := range tt.contains {
				assert.Contains(t, jsonString, contain)
			}
			for _, notContain := range tt.notContains {
				assert.NotContains(t, jsonString, notContain)
			}
		})
	}
}

func TestGetCollectionsResponse_MarshalJSON(t *testing.T) {
	tests := []struct {
		scenario    string
		value       dto.GetCollectionsResponse
		contains    []string
		notContains []string
	}{
		{
			"empty response has empty collections array not null",
			dto.GetCollectionsResponse{
				Limit:      10,
				Offset:     0,
				TotalCount: 0,
			},
			[]string{`"collections":[]`},
			[]string{`"collections":null`},
		},
		{
			"empty collection in response has empty banners array not null",
			dto.GetCollectionsResponse{
				Limit:       10,
				Offset:      0,
				TotalCount:  1,
				Collections: []dto.CollectionSummary{apitest.NewCollectionResponse(0)},
			},
			[]string{`"banners":[]`},
			[]string{`"banners":null`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.scenario, func(t *testing.T) {
			jsonBytes, err := json.Marshal(tt.value)
			require.NoError(t, err)
			jsonString := string(jsonBytes)
			for _, contain := range tt.contains {
				assert.Contains(t, jsonString, contain)
			}
			for _, notContain := range tt.notContains {
				assert.NotContains(t, jsonString, notContain)
			}
		})
	}
}

func TestCreateCollectionResponse_MarshalJSON(t *testing.T) {
	tests := []struct {
		scenario    string
		value       dto.CreateCollectionResponse
		contains    []string
		notContains []string
	}{{
		"empty collection in response has empty banners array not null",
		apitest.NewCreateCollectionResponse(0),
		[]string{`"banners":[]`},
		[]string{`"banners":null`},
	}}

	for _, tt := range tests {
		t.Run(tt.scenario, func(t *testing.T) {
			jsonBytes, err := json.Marshal(tt.value)
			require.NoError(t, err)
			jsonString := string(jsonBytes)
			for _, contain := range tt.contains {
				assert.Contains(t, jsonString, contain)
			}
			for _, notContain := range tt.notContains {
				assert.NotContains(t, jsonString, notContain)
			}
		})
	}
}

func TestPublicDataset_EmbargoReleaseDate(t *testing.T) {
	//embargoReleaseDate has no time component, so requires special marshalling/unmarshalling
	t.Run("has embargo release date", func(t *testing.T) {

		embargoReleaseDate := "2025-05-08"
		expectedEmbargoReleaseTime, err := time.Parse(time.DateOnly, embargoReleaseDate)
		require.NoError(t, err)

		t.Run("unmarshal", func(t *testing.T) {
			embargoedPublicDataset := fmt.Sprintf(`{"embargoReleaseDate": %q}`, embargoReleaseDate)
			var publicDataset dto.PublicDataset
			require.NoError(t, json.Unmarshal([]byte(embargoedPublicDataset), &publicDataset))

			assert.NotNil(t, publicDataset.EmbargoReleaseDate)
			assert.True(t, expectedEmbargoReleaseTime.Equal(time.Time(*publicDataset.EmbargoReleaseDate)))
		})

		t.Run("marshal", func(t *testing.T) {
			asDate := dto.Date(expectedEmbargoReleaseTime)
			embargoedPublicDataset := dto.PublicDataset{EmbargoReleaseDate: &asDate}
			bytes, err := json.Marshal(embargoedPublicDataset)
			require.NoError(t, err)

			assert.Contains(t, string(bytes), fmt.Sprintf(`"embargoReleaseDate":%q`, embargoReleaseDate))
		})

	})

	t.Run("no embargo release date", func(t *testing.T) {
		t.Run("unmarshal", func(t *testing.T) {
			notEmbargoedPublicDataset := `{}`
			var publicDataset dto.PublicDataset
			require.NoError(t, json.Unmarshal([]byte(notEmbargoedPublicDataset), &publicDataset))

			assert.Nil(t, publicDataset.EmbargoReleaseDate)
		})

		t.Run("marshal", func(t *testing.T) {
			notEmbargoedPublicDataset := dto.PublicDataset{}
			bytes, err := json.Marshal(notEmbargoedPublicDataset)
			require.NoError(t, err)
			assert.NotContains(t, string(bytes), `"embargoReleaseDate"`)
		})

	})

}
