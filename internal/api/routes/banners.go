package routes

import "github.com/pennsieve/collections-service/internal/api/dto"

const maxBanners = 4

// collectBanners returns a []string of length at most maxBanners containing banner URLs.
// The banner URLS will in the same order as requestedDOIs with any DOIs not found in datasetsByDOI skipped.
func collectBanners(requestedDOIs []string, datasetsByDOI map[string]dto.PublicDataset) []string {
	var banners []string
	for i, foundDOIs := 0, 0; i < len(requestedDOIs) && foundDOIs < maxBanners; i++ {
		requestedDOI := requestedDOIs[i]
		if dataset, found := datasetsByDOI[requestedDOI]; found {
			foundDOIs++
			if bannerOpt := dataset.Banner; bannerOpt != nil {
				banners = append(banners, *bannerOpt)
			} else {
				// Add a place-holder for some default banner the FE adds in?
				banners = append(banners, "")
			}
		}
	}
	return banners
}
