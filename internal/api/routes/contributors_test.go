package routes

import (
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMergedContributors_Deduplicated_NilUsable(t *testing.T) {
	var merged MergedContributors
	assert.Empty(t, merged.Deduplicated())
}

func TestMergedContributors_Deduplicated(t *testing.T) {
	contrib1 := apitest.NewPublicContributor()
	contrib2 := apitest.NewPublicContributor(apitest.WithMiddleInitial())
	contrib3 := apitest.NewPublicContributor(apitest.WithDegree())
	contrib4 := apitest.NewPublicContributor(apitest.WithOrcid())
	contrib5 := apitest.NewPublicContributor(apitest.WithMiddleInitial(), apitest.WithDegree(), apitest.WithOrcid())
	merged := MergedContributors{}

	merged = merged.Append(contrib1)
	merged = merged.Append(contrib1, contrib2)
	merged = merged.Append(contrib1, contrib2, contrib3)
	merged = merged.Append(contrib1, contrib2, contrib3, contrib4)
	merged = merged.Append(contrib1, contrib2, contrib3, contrib4, contrib5)

	noDups := merged.Deduplicated()
	assert.Equal(t, []dto.PublicContributor{contrib1, contrib2, contrib3, contrib4, contrib5}, noDups)

}
