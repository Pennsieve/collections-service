package service

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatchDOIsByQueryLength(t *testing.T) {
	// queryLen returns the length of the URL-encoded "doi=...&doi=..." string
	// that batchDOIsByQueryLength is trying to bound. Used to verify that no
	// produced batch exceeds the limit.
	queryLen := func(dois []string) int {
		params := url.Values{}
		for _, doi := range dois {
			params.Add("doi", doi)
		}
		return len(params.Encode())
	}

	t.Run("nil input produces no batches", func(t *testing.T) {
		batches := batchDOIsByQueryLength(nil, 1800)
		assert.Empty(t, batches)
	})

	t.Run("empty input produces no batches", func(t *testing.T) {
		batches := batchDOIsByQueryLength([]string{}, 1800)
		assert.Empty(t, batches)
	})

	t.Run("single DOI produces a single batch", func(t *testing.T) {
		dois := []string{"10.21397/ukuq-zazo"}
		batches := batchDOIsByQueryLength(dois, 1800)
		require.Len(t, batches, 1)
		assert.Equal(t, dois, batches[0])
	})

	t.Run("small DOI set fits in one batch", func(t *testing.T) {
		dois := []string{
			"10.21397/ukuq-zazo",
			"10.21397/haxt-dknu",
			"10.21397/qzoa-7ovf",
			"10.21397/cbop-ckwn",
		}
		batches := batchDOIsByQueryLength(dois, 1800)
		require.Len(t, batches, 1, "4 short DOIs should fit comfortably in one batch")
		assert.Equal(t, dois, batches[0])
	})

	t.Run("splits when query length would exceed limit", func(t *testing.T) {
		// 170 Pennsieve-style DOIs — the real-world worst case that triggered
		// the 414 originally.
		var dois []string
		for i := range 170 {
			dois = append(dois, fmt.Sprintf("10.21397/%04d-zazo", i))
		}
		batches := batchDOIsByQueryLength(dois, 1800)
		require.Greater(t, len(batches), 1, "170 DOIs should not fit in a single 1800-char batch")

		// All input DOIs preserved, in order, across the batches.
		var flattened []string
		for _, batch := range batches {
			flattened = append(flattened, batch...)
		}
		assert.Equal(t, dois, flattened)

		// No batch exceeds the limit.
		for i, batch := range batches {
			assert.LessOrEqualf(t, queryLen(batch), 1800, "batch %d exceeded limit", i)
		}
	})

	t.Run("DOI longer than limit gets its own batch and is not dropped", func(t *testing.T) {
		// Pathological input: a single DOI whose encoded "doi=...&" already
		// exceeds the cap. We expect the function to emit it as a single-element
		// batch rather than loop forever or drop it silently.
		hugeDOI := "10.21397/" + strings.Repeat("x", 5000)
		dois := []string{"10.21397/ukuq-zazo", hugeDOI, "10.21397/haxt-dknu"}

		batches := batchDOIsByQueryLength(dois, 1800)

		var flattened []string
		for _, batch := range batches {
			flattened = append(flattened, batch...)
		}
		assert.Equal(t, dois, flattened, "no DOI should be dropped, even oversized ones")

		// Find the batch containing the huge DOI; it should be alone.
		var hugeBatchFound bool
		for _, batch := range batches {
			for _, doi := range batch {
				if doi == hugeDOI {
					assert.Len(t, batch, 1, "oversized DOI should be alone in its batch")
					hugeBatchFound = true
				}
			}
		}
		assert.True(t, hugeBatchFound)
	})

	t.Run("very small limit forces one DOI per batch", func(t *testing.T) {
		dois := []string{"a", "b", "c"}
		// Encoded "doi=a" is 5 chars; "doi=a&doi=b" is 11. A limit of 5 forces
		// each into its own batch (the second one would push past 5).
		batches := batchDOIsByQueryLength(dois, 5)
		require.Len(t, batches, 3)
		for i, batch := range batches {
			assert.Equal(t, []string{dois[i]}, batch)
		}
	})
}
