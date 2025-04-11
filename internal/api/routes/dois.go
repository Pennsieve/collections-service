package routes

import (
	"fmt"
	"strings"
)

// CategorizeDOIs splits the given dois into either Pennsieve or non-Pennsieve, based on the prefix.
// Also de-duplicates the DOIs.
func CategorizeDOIs(pennsieveDOIPrefix string, dois []string) (pennsieveDOIs []string, externalDOIs []string) {
	pennsievePrefixAndSlash := fmt.Sprintf("%s/", pennsieveDOIPrefix)
	seenDOIs := map[string]bool{}
	// Maybe overly complicated, but trying to maintain order of the dois so that
	// if there are dups, we take the first one
	for _, doi := range dois {
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
