package collections

import "errors"

var ErrCollectionNotFound = errors.New("collection not found")

var ErrPublishInProgress = errors.New("publish already in progress")
