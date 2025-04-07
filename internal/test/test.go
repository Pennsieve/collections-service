package test

import (
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/stretchr/testify/require"
	"testing"
)

func Helper(t require.TestingT) {
	if tt, hasHelper := t.(*testing.T); hasHelper {
		tt.Helper()
	}
}

func GroupByDOI(datasets ...dto.PublicDataset) map[string]dto.PublicDataset {
	byDOI := map[string]dto.PublicDataset{}
	for _, dataset := range datasets {
		byDOI[dataset.DOI] = dataset
	}
	return byDOI
}
