package validate

import (
	"fmt"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"strings"
)

func CollectionName(name string) *apierrors.Error {
	if nameLen := len(name); nameLen == 0 {
		return apierrors.NewBadRequestError("collection name cannot be empty")
	} else if nameLen > 255 {
		return apierrors.NewBadRequestError("collection name cannot more than 255 characters")
	}
	return nil
}

func CollectionDescription(description string) *apierrors.Error {
	if descriptionLen := len(description); descriptionLen > 255 {
		return apierrors.NewBadRequestError("collection description cannot more than 255 characters")
	}
	return nil
}

func PennsieveDOIPrefix(pennsievePrefix string, dois ...string) *apierrors.Error {
	expectedPrefix := fmt.Sprintf("%s/", pennsievePrefix)
	for _, doi := range dois {
		if !strings.HasPrefix(doi, expectedPrefix) {
			return apierrors.NewBadRequestError(fmt.Sprintf("DOI %s does not have the required Pennsieve prefix %s",
				doi, pennsievePrefix))
		}
	}
	return nil
}
