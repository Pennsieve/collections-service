package service

import "fmt"

// CollectionNeverPublishedError is returned by InternalDiscover.UnpublishCollection if Discover reports
// that the collection has never been published. Discover returns a 204 No Content response in this case,
// but we are turning it into this error.
type CollectionNeverPublishedError struct {
	ID     int64
	NodeID string
}

func (e CollectionNeverPublishedError) Error() string {
	return fmt.Sprintf("collection %s (%d) has not been published", e.NodeID, e.ID)
}

// LatestDOINotFoundError is returned by DOI.GetLatestDOI if the DOI service returns a 404.
type LatestDOINotFoundError struct {
	ID     int64
	NodeID string
}

func (e LatestDOINotFoundError) Error() string {
	return fmt.Sprintf("most recent DOI for %s (%d) not found", e.NodeID, e.ID)
}
