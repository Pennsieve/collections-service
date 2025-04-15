package dto_test

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetCollectionResponse_MarshalJSON(t *testing.T) {
	banner := apitest.NewBanner()
	contributor := uuid.NewString()
	externalData := `{"name":"external name","externalProp":"external value"}`
	tests := []struct {
		scenario    string
		value       dto.GetCollectionResponse
		contains    []string
		notContains []string
	}{
		{"empty collection has empty arrays not null",
			dto.GetCollectionResponse{
				CollectionResponse: apitest.NewCollectionResponse(0),
			},
			[]string{`"banners":[]`, `"contributors":[]`, `"datasets":[]`},
			[]string{`"banners":null`, `"contributors":null`, `"datasets":null`},
		},
		{"collection contains contributor and dataset",
			dto.GetCollectionResponse{
				CollectionResponse: apitest.NewCollectionResponse(1, *banner),
				Contributors:       []string{contributor},
				Datasets:           []dto.Dataset{{Source: dto.ExternalSource, Data: []byte(externalData)}},
			},
			[]string{
				fmt.Sprintf(`"banners":[%q]`, *banner),
				fmt.Sprintf(`"contributors":[%q]`, contributor),
				fmt.Sprintf(`"source":%q`, dto.ExternalSource),
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
				Collections: []dto.CollectionResponse{apitest.NewCollectionResponse(0)},
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
