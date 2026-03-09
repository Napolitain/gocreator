package services

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlideNarrationBaseCandidates(t *testing.T) {
	assert.Equal(t, []string{"01-cover"}, slideNarrationBaseCandidates("01-cover.png"))
	assert.Equal(t, []string{"02-handout-page-0007", "02-handout-p007", "02-handout-p7"}, slideNarrationBaseCandidates("02-handout-page-0007.png"))
}

func TestVideoCreatorLookupSourceText_UsesPDFAlias(t *testing.T) {
	fs := afero.NewMemMapFs()
	creator := &VideoCreator{fs: fs}
	slidesDir := testPath("test", "data", "slides")

	require.NoError(t, fs.MkdirAll(slidesDir, 0755))
	require.NoError(t, afero.WriteFile(fs, testPath("test", "data", "slides", "02-handout-p001.txt"), []byte("Page one"), 0644))

	text, found, err := creator.lookupSourceText(slidesDir, testPath("test", "data", "cache", "pdf", "02-handout-page-0001.png"), "en")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "Page one", text)
}

func TestVideoCreatorLookupSourceText_FailsOnAmbiguousPDFAliases(t *testing.T) {
	fs := afero.NewMemMapFs()
	creator := &VideoCreator{fs: fs}
	slidesDir := testPath("test", "data", "slides")

	require.NoError(t, fs.MkdirAll(slidesDir, 0755))
	require.NoError(t, afero.WriteFile(fs, testPath("test", "data", "slides", "02-handout-page-0001.txt"), []byte("Page one"), 0644))
	require.NoError(t, afero.WriteFile(fs, testPath("test", "data", "slides", "02-handout-p001.txt"), []byte("Alias page one"), 0644))

	_, _, err := creator.lookupSourceText(slidesDir, testPath("test", "data", "cache", "pdf", "02-handout-page-0001.png"), "en")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple matching text sidecars")
}
