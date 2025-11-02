package services

import (
	"context"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTextService_Load(t *testing.T) {
	tests := []struct {
		name        string
		fileContent string
		expected    []string
		expectError bool
	}{
		{
			name:        "single text",
			fileContent: "Hello world",
			expected:    []string{"Hello world"},
			expectError: false,
		},
		{
			name:        "multiple texts with delimiter",
			fileContent: "First text\n-\nSecond text\n-\nThird text",
			expected:    []string{"First text", "Second text", "Third text"},
			expectError: false,
		},
		{
			name:        "multiline texts",
			fileContent: "First line\nSecond line\n-\nAnother text",
			expected:    []string{"First line\nSecond line", "Another text"},
			expectError: false,
		},
		{
			name:        "empty file",
			fileContent: "",
			expected:    []string{},
			expectError: false,
		},
		{
			name:        "only delimiters",
			fileContent: "-\n-\n-",
			expected:    []string{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create in-memory filesystem
			fs := afero.NewMemMapFs()
			logger := &mockLogger{}
			service := NewTextService(fs, logger)

			// Create test file
			testPath := "/test.txt"
			err := afero.WriteFile(fs, testPath, []byte(tt.fileContent), 0644)
			require.NoError(t, err)

			// Test Load
			ctx := context.Background()
			result, err := service.Load(ctx, testPath)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestTextService_Load_FileNotFound(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewTextService(fs, logger)

	ctx := context.Background()
	_, err := service.Load(ctx, "/nonexistent.txt")
	assert.Error(t, err)
}

func TestTextService_Save(t *testing.T) {
	tests := []struct {
		name     string
		texts    []string
		expected string
	}{
		{
			name:     "single text",
			texts:    []string{"Hello world"},
			expected: "Hello world",
		},
		{
			name:     "multiple texts",
			texts:    []string{"First", "Second", "Third"},
			expected: "First\n-\nSecond\n-\nThird",
		},
		{
			name:     "empty slice",
			texts:    []string{},
			expected: "",
		},
		{
			name:     "multiline texts",
			texts:    []string{"Line 1\nLine 2", "Another text"},
			expected: "Line 1\nLine 2\n-\nAnother text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			logger := &mockLogger{}
			service := NewTextService(fs, logger)

			testPath := "/test.txt"
			ctx := context.Background()

			// Test Save
			err := service.Save(ctx, testPath, tt.texts)
			assert.NoError(t, err)

			// Verify file content
			content, err := afero.ReadFile(fs, testPath)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(content))
		})
	}
}

func TestTextService_Hash(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewTextService(fs, logger)

	tests := []struct {
		name  string
		text  string
		text2 string
		same  bool
	}{
		{
			name:  "identical texts produce same hash",
			text:  "Hello world",
			text2: "Hello world",
			same:  true,
		},
		{
			name:  "different texts produce different hashes",
			text:  "Hello world",
			text2: "Hello World",
			same:  false,
		},
		{
			name:  "empty string",
			text:  "",
			text2: "",
			same:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := service.Hash(tt.text)
			hash2 := service.Hash(tt.text2)

			assert.NotEmpty(t, hash1)
			assert.NotEmpty(t, hash2)

			if tt.same {
				assert.Equal(t, hash1, hash2)
			} else {
				assert.NotEqual(t, hash1, hash2)
			}
		})
	}
}

func TestTextService_SaveAndLoadHashes(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewTextService(fs, logger)

	ctx := context.Background()
	testPath := "/hashes.txt"

	// Test saving hashes
	hashes := []string{
		"abc123",
		"def456",
		"ghi789",
	}

	err := service.SaveHashes(ctx, testPath, hashes)
	assert.NoError(t, err)

	// Test loading hashes
	loadedHashes, err := service.LoadHashes(ctx, testPath)
	assert.NoError(t, err)
	assert.Equal(t, hashes, loadedHashes)
}

func TestTextService_LoadHashes_NonExistent(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewTextService(fs, logger)

	ctx := context.Background()
	hashes, err := service.LoadHashes(ctx, "/nonexistent.txt")
	
	assert.NoError(t, err)
	assert.Empty(t, hashes)
}
