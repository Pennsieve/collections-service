package routes

import (
	"fmt"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/datasource"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/collections-service/internal/api/store"
	"strings"
)

// CategorizeDOIs splits the given dois into either Pennsieve or non-Pennsieve, based on the prefix.
// Also de-duplicates the DOIs and trims any leading or trailing whitespace.
func CategorizeDOIs(pennsieveDOIPrefix string, dois []string) (pennsieveDOIs []string, externalDOIs []string) {
	pennsievePrefixAndSlash := fmt.Sprintf("%s/", pennsieveDOIPrefix)
	seenDOIs := map[string]bool{}
	// Maybe overly complicated, but trying to maintain order of the dois so that
	// if there are dups, we take the first one
	for _, doi := range dois {
		doi = strings.TrimSpace(doi)
		if _, seen := seenDOIs[doi]; !seen {
			seenDOIs[doi] = true
			if strings.HasPrefix(doi, pennsievePrefixAndSlash) {
				pennsieveDOIs = append(pennsieveDOIs, doi)
			} else {
				externalDOIs = append(externalDOIs, doi)
			}
		}
	}
	return
}

func GroupByDatasource(dois []store.DOI) (pennsieveDOIs []string, externalDOIs []string) {
	for _, doi := range dois {
		if doi.Datasource == datasource.Pennsieve {
			pennsieveDOIs = append(pennsieveDOIs, doi.Value)
		} else {
			externalDOIs = append(externalDOIs, doi.Value)
		}
	}
	return
}

// ValidateDiscoverResponse returns a Bad Request *apierrors.Error if datasetResults
// contains unpublished datasets or published collection datasets (a collection cannot contain a collection).
func ValidateDiscoverResponse(datasetResults service.DatasetsByDOIResponse) error {
	if len(datasetResults.Unpublished) > 0 {
		var details []string
		for _, unpublished := range datasetResults.Unpublished {
			details = append(details, fmt.Sprintf("%s status is %s", unpublished.DOI, unpublished.Status))
		}
		return apierrors.NewBadRequestError(fmt.Sprintf("request contains unpublished DOIs: %s", strings.Join(details, ", ")))
	}

	var collectionDetails []string
	for publishedDOI, published := range datasetResults.Published {
		if published.DatasetType != nil && *published.DatasetType == dto.CollectionDatasetType {
			collectionDetails = append(collectionDetails, publishedDOI)
		}
	}
	if len(collectionDetails) > 0 {
		return apierrors.NewBadRequestError(fmt.Sprintf("request contains collection DOIs: %s", strings.Join(collectionDetails, ", ")))
	}
	return nil
}
