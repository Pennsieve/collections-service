package validate

import (
	"github.com/pennsieve/collections-service/internal/api/apierrors"
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
