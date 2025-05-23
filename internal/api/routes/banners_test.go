package routes

import (
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_collectBanners(t *testing.T) {
	type args struct {
		requestedDOIs []string
		datasetsByDOI map[string]dto.PublicDataset
	}

	dataset1 := dto.PublicDataset{DOI: apitest.NewPennsieveDOI().Value, Banner: apitest.NewBanner()}
	dataset2 := dto.PublicDataset{DOI: apitest.NewPennsieveDOI().Value, Banner: apitest.NewBanner()}
	dataset3 := dto.PublicDataset{DOI: apitest.NewPennsieveDOI().Value, Banner: apitest.NewBanner()}
	dataset4 := dto.PublicDataset{DOI: apitest.NewPennsieveDOI().Value, Banner: apitest.NewBanner()}
	dataset5 := dto.PublicDataset{DOI: apitest.NewPennsieveDOI().Value, Banner: apitest.NewBanner()}
	datasetWithoutBanner := dto.PublicDataset{DOI: apitest.NewPennsieveDOI().Value}

	tests := []struct {
		name     string
		args     args
		expected []string
	}{
		{
			"everything empty",
			args{nil, nil},
			nil,
		},
		{
			"map empty",
			args{[]string{apitest.NewExternalDOI().Value, apitest.NewExternalDOI().Value, apitest.NewExternalDOI().Value}, nil},
			nil,
		},
		{
			"less than 4",
			args{[]string{apitest.NewExternalDOI().Value, dataset1.DOI, apitest.NewExternalDOI().Value},
				groupByDOI(dataset1)},
			[]string{*dataset1.Banner},
		},
		{
			"four",
			args{[]string{dataset1.DOI, dataset2.DOI, dataset3.DOI, dataset4.DOI},
				groupByDOI(dataset1, dataset2, dataset3, dataset4)},
			[]string{*dataset1.Banner, *dataset2.Banner, *dataset3.Banner, *dataset4.Banner},
		},
		{
			"more than four",
			args{[]string{dataset5.DOI, dataset4.DOI, dataset3.DOI, dataset2.DOI, dataset1.DOI},
				groupByDOI(dataset1, dataset2, dataset3, dataset4, dataset5)},
			[]string{*dataset5.Banner, *dataset4.Banner, *dataset3.Banner, *dataset2.Banner},
		},
		{
			"a lot of external DOIs",
			args{[]string{apitest.NewExternalDOI().Value, apitest.NewExternalDOI().Value, apitest.NewExternalDOI().Value, apitest.NewExternalDOI().Value, apitest.NewExternalDOI().Value, dataset1.DOI},
				groupByDOI(dataset1)},
			[]string{*dataset1.Banner},
		},
		{
			"a good DOI missing a banner",
			args{[]string{dataset5.DOI, datasetWithoutBanner.DOI, dataset1.DOI},
				groupByDOI(dataset1, datasetWithoutBanner, dataset5)},
			[]string{*dataset5.Banner, "", *dataset1.Banner},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.expected, collectBanners(tt.args.requestedDOIs, tt.args.datasetsByDOI), "collectBanners(%v, %v)", tt.args.requestedDOIs, tt.args.datasetsByDOI)
		})
	}
}

func groupByDOI(datasets ...dto.PublicDataset) map[string]dto.PublicDataset {
	byDOI := map[string]dto.PublicDataset{}
	for _, dataset := range datasets {
		byDOI[dataset.DOI] = dataset
	}
	return byDOI
}
