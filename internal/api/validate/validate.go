package validate

import (
	"fmt"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
)

func CollectionName(name string) *apierrors.Error {
	if nameLen := len(name); nameLen == 0 {
		return apierrors.NewBadRequestError("collection name cannot be empty")
	} else if nameLen > 255 {
		return apierrors.NewBadRequestError("collection name cannot have more than 255 characters")
	}
	return nil
}

func CollectionDescription(description string) *apierrors.Error {
	if descriptionLen := len(description); descriptionLen > 255 {
		return apierrors.NewBadRequestError("collection description cannot have more than 255 characters")
	}
	return nil
}

func IntQueryParamValue(key string, value int, requiredMin int) *apierrors.Error {
	if value < requiredMin {
		return apierrors.NewBadRequestError(fmt.Sprintf("query param %s cannot be less than %d: %d", key, requiredMin, value))
	}
	return nil
}
