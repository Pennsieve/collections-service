package routes

import "github.com/pennsieve/collections-service/internal/api/dto"

type MergedContributors []dto.PublicContributor

func (m MergedContributors) Append(contributors ...dto.PublicContributor) MergedContributors {
	return append(m, contributors...)
}

// Deduplicated maintains the order that the contributors are added to this MergedContributors
func (m MergedContributors) Deduplicated() []dto.PublicContributor {
	var deduplicated []dto.PublicContributor
	seenContribs := make(map[dto.PublicContributor]bool)
	for _, c := range m {
		if seen := seenContribs[c]; !seen {
			seenContribs[c] = true
			deduplicated = append(deduplicated, c)
		}
	}
	return deduplicated
}
