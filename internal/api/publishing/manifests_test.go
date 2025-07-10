package publishing_test

import (
	"encoding/json"
	"github.com/pennsieve/collections-service/internal/api/publishing"
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestManifestBuilder_Build(t *testing.T) {
	tests := []struct {
		scenario    string
		description string
	}{
		{"all one-byte characters", "This description only has one-byte utf-8 characters"},
		{"two-byte character", `This description has a two-byte utf-8 character: ©`},
		{"three-byte character", `This description has a three-byte utf-8 character: ࠉ`},
		{"four-byte character", `This description has a four-byte utf-8 character: 𒀻`},
	}

	for _, tt := range tests {
		t.Run(tt.scenario, func(t *testing.T) {
			expectedManifest := apitest.NewExpectedManifest(t, apitest.WithManifestDescription(tt.description))

			require.Len(t, expectedManifest.Files, 1)
			manifestEntry := apitest.FindManifestEntry(t, expectedManifest)
			assert.Equal(t, publishing.ManifestFileName, manifestEntry.Name)
			assert.Equal(t, publishing.ManifestFileName, manifestEntry.Path)
			assert.Equal(t, publishing.ManifestFileType, manifestEntry.FileType)
			assert.NotZero(t, manifestEntry.Size)

			asBytes, err := expectedManifest.Marshal()
			require.NoError(t, err)
			assert.Equal(t, manifestEntry.Size, int64(len(asBytes)))

			var decodedManifest publishing.ManifestV5
			require.NoError(t, json.Unmarshal(asBytes, &decodedManifest))

			apitest.RequireManifestsEqual(t, expectedManifest, decodedManifest)
		})
	}

}

func TestManifestBuilder_Build_EdgeCase_OrderOfMagnitudeChange(t *testing.T) {
	// Get the size a base manifest
	tempManifest, err := publishing.NewManifestBuilder().Build()
	require.NoError(t, err)
	tempManifestBytes, err := tempManifest.Marshal()
	tempSize := len(tempManifestBytes)

	// Find the lowest power of 10 that is bigger than tempSize
	powerOfTen := 10
	searching := true
	for searching {
		if powerOfTen/10 <= tempSize && tempSize < powerOfTen {
			searching = false
		} else {
			powerOfTen = powerOfTen * 10
		}
	}

	// Create the test manifest with a power of 10 size.
	manifest, err := publishing.NewManifestBuilder().WithDescription(strings.Repeat("a", powerOfTen-tempSize)).Build()
	require.NoError(t, err)
	manifestBytes, err := manifest.Marshal()
	require.NoError(t, err)
	manifestSize := int64(len(manifestBytes))

	sizeInManifest := apitest.FindManifestEntry(t, manifest).Size
	assert.Equal(t, manifestSize, sizeInManifest)

}

func TestManifestV5_TotalSize(t *testing.T) {
	// this test really depends on the fact that apitest.NewExpectedManifest uses
	// the Builder which sets the manifestEntry for us.
	manifest := apitest.NewExpectedManifest(t)

	// right now there is only one file, the manifest itself,
	// so TotalSize() should equal size of manifest.
	require.Len(t, manifest.Files, 1)

	manifestBytes, err := manifest.Marshal()
	require.NoError(t, err)
	assert.Equal(t, int64(len(manifestBytes)), manifest.TotalSize())

}
