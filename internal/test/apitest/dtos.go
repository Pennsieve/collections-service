package apitest

import (
	"github.com/pennsieve/collections-service/internal/api/dto"
)

func NewPublicDataset(doi string, banner *string) dto.PublicDataset {
	// as of now, other values here are not relevant to tests. Maybe add some later
	// Slices are initialized empty so that they match objects that were marshalled and then unmarshalled
	// in tests. (We marshal nil slices into "[]" rather than null.
	return dto.PublicDataset{
		DOI:                  doi,
		Banner:               banner,
		Tags:                 make([]string, 0),
		ModelCount:           make([]dto.ModelCount, 0),
		Collections:          make([]dto.PublicCollection, 0),
		Contributors:         make([]dto.PublicContributor, 0),
		ExternalPublications: make([]dto.PublicExternalPublication, 0),
	}
}

func NewTombstone(doi string, status string) dto.Tombstone {
	// as of now, other values here are not relevant to tests. Maybe add some later
	// Slices are initialized empty so that they match objects that were marshalled and then unmarshalled
	// in tests. (We marshal nil slices into "[]" rather than null.
	return dto.Tombstone{
		Status: status,
		DOI:    doi,
		Tags:   make([]string, 0),
	}
}
