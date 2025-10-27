package service

import (
	"context"
	"fmt"
	"github.com/pennsieve/collections-service/internal/shared/logging"
	"github.com/pennsieve/collections-service/internal/shared/util"
	"github.com/stretchr/testify/require"
	"math"
	"net/http"
	"net/url"
	"testing"
	"time"
)

// --- Helper: average and stddev ---
func stats(durations []time.Duration) (avg, stddev time.Duration) {
	if len(durations) == 0 {
		return 0, 0
	}
	var total float64
	for _, d := range durations {
		total += float64(d)
	}
	mean := total / float64(len(durations))

	var sumsq float64
	for _, d := range durations {
		diff := float64(d) - mean
		sumsq += diff * diff
	}
	std := math.Sqrt(sumsq / float64(len(durations)))

	return time.Duration(mean), time.Duration(std)
}

func lookupDOIs(ctx context.Context, t *testing.T, discoverService *HTTPDiscover, limit int) []string {
	doiQueryParams := url.Values{}
	doiQueryParams.Add("limit", fmt.Sprintf("%d", limit))

	requestParams := requestParameters{
		method: http.MethodGet,
		url:    fmt.Sprintf("%s/datasets?%s", discoverService.url, doiQueryParams.Encode()),
	}
	response, err := discoverService.InvokePennsieve(ctx, requestParams)
	require.NoError(t, err)
	defer util.CloseAndWarn(response, discoverService.logger)

	type Dataset struct {
		DOI string `json:"doi"`
	}
	type DTO struct {
		Datasets []Dataset `json:"datasets"`
	}
	var dto DTO
	require.NoError(t, util.UnmarshallResponse(response, &dto))

	var dois []string
	for _, dataset := range dto.Datasets {
		dois = append(dois, dataset.DOI)
	}
	require.Len(t, dois, limit)
	return dois
}

// --- Main benchmark ---
func TestBatchSize(t *testing.T) {
	t.Skip("just meant to be run manually to see how many DOIs we should request in a batch")
	ctx := context.Background()
	discover := NewHTTPDiscover("https://api.pennsieve.io/discover", logging.Default)

	// Try these batch sizes
	batchSizes := []int{1, 5, 10, 25, 50, 60, 70, 80, 90}
	trials := 5

	fmt.Printf("%8s %8s %12s %12s %12s\n",
		"DOIs", "Trials", "Avg(ms)", "StdDev(ms)", "PerDOI(ms)")
	fmt.Println("-------------------------------------------------------------")

	for _, n := range batchSizes {
		durations := make([]time.Duration, 0, trials)

		for trialIdx := 0; trialIdx < trials; trialIdx++ {
			dois := lookupDOIs(ctx, t, discover, n)
			for i := range dois {
				dois[i] = fmt.Sprintf("10.1234/fake%d", i)
			}

			start := time.Now()
			if _, err := discover.GetDatasetsByDOI(ctx, dois); err != nil {
				fmt.Printf("%4d DOIs -> trial %d failed: %v\n", n, trialIdx+1, err)
				continue
			}
			durations = append(durations, time.Since(start))
		}

		if len(durations) == 0 {
			continue
		}

		avg, stddev := stats(durations)
		fmt.Printf("%8d %8d %12.1f %12.1f %12.2f\n",
			n, len(durations),
			float64(avg.Milliseconds()),
			float64(stddev.Milliseconds()),
			float64(avg.Milliseconds())/float64(n))
	}
}
