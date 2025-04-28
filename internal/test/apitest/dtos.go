package apitest

import (
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
	"math/rand"
	"strings"
)

func NewPublicDataset(doi string, banner *string, contributors ...dto.PublicContributor) dto.PublicDataset {
	// as of now, other values here are not relevant to tests. Maybe add some later
	// Slices are initialized empty so that they match objects that were marshalled and then unmarshalled
	// in tests. (We marshal nil slices into "[]" rather than null.)
	dataset := dto.PublicDataset{
		DOI:                  doi,
		Banner:               banner,
		Tags:                 make([]string, 0),
		ModelCount:           make([]dto.ModelCount, 0),
		Collections:          make([]dto.PublicCollection, 0),
		Contributors:         make([]dto.PublicContributor, 0),
		ExternalPublications: make([]dto.PublicExternalPublication, 0),
	}
	for _, c := range contributors {
		dataset.Contributors = append(dataset.Contributors, c)
	}
	return dataset
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

func NewCreateCollectionResponse(size int, banners ...string) dto.CreateCollectionResponse {
	return dto.CreateCollectionResponse(NewCollectionResponse(size, banners...))
}

func NewCollectionResponse(size int, banners ...string) dto.CollectionSummary {
	return dto.CollectionSummary{
		NodeID:      uuid.NewString(),
		Name:        uuid.NewString(),
		Description: uuid.NewString(),
		Banners:     banners,
		Size:        size,
		UserRole:    role.Owner.String(),
	}
}

type PublicContributorOptionFunc func(contributor *dto.PublicContributor)

func NewPublicContributor(options ...PublicContributorOptionFunc) dto.PublicContributor {
	contributor := &dto.PublicContributor{
		FirstName: uuid.NewString(),
		LastName:  uuid.NewString(),
	}
	for _, option := range options {
		option(contributor)
	}
	return *contributor
}

func WithMiddleInitial() PublicContributorOptionFunc {
	return func(contributor *dto.PublicContributor) {
		middleInitial := randomLetter()
		contributor.MiddleInitial = &middleInitial
	}
}

func WithDegree() PublicContributorOptionFunc {
	return func(contributor *dto.PublicContributor) {
		var sb strings.Builder
		for i := 0; i < 3; i++ {
			sb.WriteString(randomLetter())
		}
		degree := sb.String()
		contributor.Degree = &degree
	}
}

func WithOrcid() PublicContributorOptionFunc {
	return func(contributor *dto.PublicContributor) {
		orcid := uuid.NewString()
		contributor.Orcid = &orcid
	}
}

func randomLetter() string {
	return string("ABCDEFGHIJKLMNOPQURTUVWXYZ"[rand.Intn(26)])
}
