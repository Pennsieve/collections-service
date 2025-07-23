package validate

import (
	"fmt"
	"github.com/pennsieve/collections-service/internal/api/apierrors"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"slices"
	"strings"
)

func CollectionName(value string) error {
	if valueLen := len(value); valueLen == 0 {
		return apierrors.NewBadRequestError("collection name cannot be empty")
	} else if valueLen > 255 {
		return apierrors.NewBadRequestError("collection name cannot have more than 255 characters")
	}
	return nil
}

func CollectionDescription(value string) error {
	if valueLen := len(value); valueLen > 255 {
		return apierrors.NewBadRequestError("collection description cannot have more than 255 characters")
	}
	return nil
}

func IntQueryParamValue(key string, value int, requiredMin int) error {
	if value < requiredMin {
		return apierrors.NewBadRequestError(fmt.Sprintf("query param %s cannot be less than %d: %d", key, requiredMin, value))
	}
	return nil
}

func License(value *string, required bool) error {
	if value == nil || len(*value) == 0 {
		if required {
			return apierrors.NewBadRequestError("missing required license")
		}
		return nil
	}
	idx := slices.Index(dto.ValidLicenses, *value)
	if idx == -1 {
		return apierrors.NewBadRequestError(fmt.Sprintf("invalid license: %q", *value))
	}
	return nil
}

func Tags(value []string, required bool) error {
	//Discover DB defines tags as an array of text, so no max value on length of individual tag.

	if value == nil || len(value) == 0 {
		if required {
			return apierrors.NewBadRequestError("tags array cannot be empty")
		}
		return nil
	}

	for _, tag := range value {
		if len(strings.TrimSpace(tag)) == 0 {
			return apierrors.NewBadRequestError("tags array cannot contain empty values")
		}
	}
	return nil
}
