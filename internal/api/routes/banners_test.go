package routes

import (
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_collectBanners(t *testing.T) {
	type args struct {
		requestedDOIs []string
		datasetsByDOI map[string]dto.PublicDataset
	}

	dataset1 := dto.PublicDataset{DOI: test.NewDOI(), Banner: test.NewBanner()}
	dataset2 := dto.PublicDataset{DOI: test.NewDOI(), Banner: test.NewBanner()}
	dataset3 := dto.PublicDataset{DOI: test.NewDOI(), Banner: test.NewBanner()}
	dataset4 := dto.PublicDataset{DOI: test.NewDOI(), Banner: test.NewBanner()}
	dataset5 := dto.PublicDataset{DOI: test.NewDOI(), Banner: test.NewBanner()}
	datasetWithoutBanner := dto.PublicDataset{DOI: test.NewDOI()}

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
			args{[]string{test.NewDOI(), test.NewDOI(), test.NewDOI()}, nil},
			nil,
		},
		{
			"less than 4",
			args{[]string{test.NewDOI(), dataset1.DOI, test.NewDOI()},
				test.GroupByDOI(dataset1)},
			[]string{*dataset1.Banner},
		},
		{
			"four",
			args{[]string{dataset1.DOI, dataset2.DOI, dataset3.DOI, dataset4.DOI},
				test.GroupByDOI(dataset1, dataset2, dataset3, dataset4)},
			[]string{*dataset1.Banner, *dataset2.Banner, *dataset3.Banner, *dataset4.Banner},
		},
		{
			"more than four",
			args{[]string{dataset5.DOI, dataset4.DOI, dataset3.DOI, dataset2.DOI, dataset1.DOI},
				test.GroupByDOI(dataset1, dataset2, dataset3, dataset4, dataset5)},
			[]string{*dataset5.Banner, *dataset4.Banner, *dataset3.Banner, *dataset2.Banner},
		},
		{
			"a lot of external DOIs",
			args{[]string{test.NewDOI(), test.NewDOI(), test.NewDOI(), test.NewDOI(), test.NewDOI(), dataset1.DOI},
				test.GroupByDOI(dataset1)},
			[]string{*dataset1.Banner},
		},
		{
			"a good DOI missing a banner",
			args{[]string{dataset5.DOI, datasetWithoutBanner.DOI, dataset1.DOI},
				test.GroupByDOI(dataset1, datasetWithoutBanner, dataset5)},
			[]string{*dataset5.Banner, "", *dataset1.Banner},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.expected, collectBanners(tt.args.requestedDOIs, tt.args.datasetsByDOI), "collectBanners(%v, %v)", tt.args.requestedDOIs, tt.args.datasetsByDOI)
		})
	}
}
