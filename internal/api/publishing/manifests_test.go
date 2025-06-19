package publishing

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"slices"
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
			builder := NewManifestBuilder().WithDescription(tt.description)
			manifest, err := builder.Build()
			require.NoError(t, err)

			require.Len(t, manifest.Files, 1)
			manifestEntry := manifest.Files[0]
			assert.Equal(t, ManifestFileName, manifestEntry.Name)
			assert.Equal(t, ManifestFileName, manifestEntry.Path)
			assert.Equal(t, ManifestFileType, manifestEntry.FileType)
			assert.NotZero(t, manifestEntry.Size)

			asBytes, err := manifest.Marshal()
			require.NoError(t, err)
			assert.Equal(t, manifestEntry.Size, int64(len(asBytes)))
		})
	}

}

func TestManifestBuilder_Build_EdgeCase_OrderOfMagnitudeChange(t *testing.T) {
	// Get the size a base manifest
	tempManifest, err := NewManifestBuilder().Build()
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
	manifest, err := NewManifestBuilder().WithDescription(strings.Repeat("a", powerOfTen-tempSize)).Build()
	require.NoError(t, err)
	manifestBytes, err := manifest.Marshal()
	require.NoError(t, err)
	manifestSize := int64(len(manifestBytes))

	manifestEntryIdx := slices.IndexFunc(manifest.Files, func(fileManifest FileManifest) bool {
		return fileManifest.Path == ManifestFileName
	})
	require.True(t, manifestEntryIdx >= 0)
	sizeInManifest := manifest.Files[manifestEntryIdx].Size
	assert.Equal(t, manifestSize, sizeInManifest)

}
