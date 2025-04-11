package apitest

import (
	"github.com/pennsieve/collections-service/internal/api/dto"
)

func NewPublicDataset(doi string, banner *string) dto.PublicDataset {
	// as of now, other values here are not relevant to tests. Maybe add some later
	return dto.PublicDataset{
		DOI:    doi,
		Banner: banner,
	}
}

func NewTombstone(doi string, status string) dto.Tombstone {
	// as of now, other values here are not relevant to tests. Maybe add some later
	return dto.Tombstone{
		Status: status,
		DOI:    doi,
	}
}
