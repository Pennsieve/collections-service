package apitest

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/collections-service/internal/test/mocks"
)

func NewBanner() *string {
	banner := fmt.Sprintf("https://example.com/%s.png", uuid.NewString())
	return &banner
}

type TestBanners map[string]string

func (t TestBanners) WithExpectedPennsieveBanners(expectedDOIs []string) {
	for _, expectedDOI := range expectedDOIs {
		t[expectedDOI] = NewPennsieveDOI()
	}
}

func (t TestBanners) GetExpectedBannersForDOIs(expectedDOIs []string) (expectedBanners []string) {
	for _, expectedDOI := range expectedDOIs {
		expectedBanners = append(expectedBanners, t[expectedDOI])
	}
	return
}

func (t TestBanners) ToDiscoverGetDatasetsByDOIFunc() mocks.GetDatasetsByDOIFunc {
	return func(dois []string) (service.DatasetsByDOIResponse, error) {
		response := service.DatasetsByDOIResponse{
			Published: make(map[string]dto.PublicDataset),
		}
		for _, doi := range dois {
			if banner, found := t[doi]; found {
				response.Published[doi] = NewPublicDataset(doi, &banner)
			} else {
				//not sure what will be best here. Ignore these, send back as unpublished tombstones, or as published with missing banners
				response.Published[doi] = NewPublicDataset(doi, nil)
			}
		}
		return response, nil
	}
}
