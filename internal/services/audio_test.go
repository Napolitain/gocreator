package services

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"gocreator/internal/mocks"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockReadCloser implements io.ReadCloser for testing
type mockReadCloser struct {
	*strings.Reader
}

func (m *mockReadCloser) Close() error {
	return nil
}

func newMockReadCloser(data string) io.ReadCloser {
	return &mockReadCloser{Reader: strings.NewReader(data)}
}

func TestAudioService_Generate(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		outputPath  string
		mockData    string
		mockError   error
		expectError bool
	}{
		{
			name:        "successful generation",
			text:        "Hello world",
			outputPath:  "/output/audio.mp3",
			mockData:    "mock audio data",
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "api error",
			text:        "Test",
			outputPath:  "/output/audio.mp3",
			mockData:    "",
			mockError:   errors.New("API error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			mockClient := new(mocks.MockOpenAIClient)
			logger := &mockLogger{}
			textService := NewTextService(fs, logger)
			service := NewAudioService(fs, mockClient, textService, logger)

			// Setup mock expectations
			if tt.mockError == nil {
				mockClient.On("GenerateSpeech", mock.Anything, tt.text).
					Return(newMockReadCloser(tt.mockData), tt.mockError)
			} else {
				mockClient.On("GenerateSpeech", mock.Anything, tt.text).
					Return(nil, tt.mockError)
			}

			ctx := context.Background()
			err := service.Generate(ctx, tt.text, tt.outputPath)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Verify file was created
				exists, _ := afero.Exists(fs, tt.outputPath)
				assert.True(t, exists)

				// Verify content
				content, _ := afero.ReadFile(fs, tt.outputPath)
				assert.Equal(t, tt.mockData, string(content))
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestAudioService_Generate_WithCache(t *testing.T) {
	fs := afero.NewMemMapFs()
	mockClient := new(mocks.MockOpenAIClient)
	logger := &mockLogger{}
	textService := NewTextService(fs, logger)
	service := NewAudioService(fs, mockClient, textService, logger)

	text := "Hello world"
	outputPath := "/output/audio.mp3"

	// First generation - should call API
	mockClient.On("GenerateSpeech", mock.Anything, text).
		Return(newMockReadCloser("audio data"), nil).Once()

	ctx := context.Background()
	err := service.Generate(ctx, text, outputPath)
	require.NoError(t, err)

	// Second generation with same text - should use cache, not call API again
	// We don't set up another mock expectation, so if it calls the API, the test will fail
	err = service.Generate(ctx, text, outputPath)
	assert.NoError(t, err)

	mockClient.AssertExpectations(t)
}

func TestAudioService_GenerateBatch(t *testing.T) {
	tests := []struct {
		name          string
		texts         []string
		outputDir     string
		mockData      []string
		expectedPaths []string
		expectError   bool
	}{
		{
			name:      "successful batch generation",
			texts:     []string{"Hello", "World", "Test"},
			outputDir: "/output",
			mockData:  []string{"audio1", "audio2", "audio3"},
			expectedPaths: []string{
				"/output/0.mp3",
				"/output/1.mp3",
				"/output/2.mp3",
			},
			expectError: false,
		},
		{
			name:          "empty input",
			texts:         []string{},
			outputDir:     "/output",
			mockData:      []string{},
			expectedPaths: []string{},
			expectError:   false,
		},
		{
			name:      "single item",
			texts:     []string{"Test"},
			outputDir: "/output",
			mockData:  []string{"audio1"},
			expectedPaths: []string{
				"/output/0.mp3",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			mockClient := new(mocks.MockOpenAIClient)
			logger := &mockLogger{}
			textService := NewTextService(fs, logger)
			service := NewAudioService(fs, mockClient, textService, logger)

			// Setup mock expectations for each text
			for i, text := range tt.texts {
				mockClient.On("GenerateSpeech", mock.Anything, text).
					Return(newMockReadCloser(tt.mockData[i]), nil)
			}

			ctx := context.Background()
			paths, err := service.GenerateBatch(ctx, tt.texts, tt.outputDir)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedPaths, paths)

				// Verify all files were created
				for i, path := range paths {
					exists, _ := afero.Exists(fs, path)
					assert.True(t, exists, "File should exist: %s", path)

					content, _ := afero.ReadFile(fs, path)
					assert.Equal(t, tt.mockData[i], string(content))
				}
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestAudioService_GenerateBatch_WithCache(t *testing.T) {
	fs := afero.NewMemMapFs()
	mockClient := new(mocks.MockOpenAIClient)
	logger := &mockLogger{}
	textService := NewTextService(fs, logger)
	service := NewAudioService(fs, mockClient, textService, logger)

	texts := []string{"Hello", "World"}
	outputDir := "/output"

	// First batch generation
	mockClient.On("GenerateSpeech", mock.Anything, "Hello").
		Return(newMockReadCloser("audio1"), nil).Once()
	mockClient.On("GenerateSpeech", mock.Anything, "World").
		Return(newMockReadCloser("audio2"), nil).Once()

	ctx := context.Background()
	paths1, err := service.GenerateBatch(ctx, texts, outputDir)
	require.NoError(t, err)
	assert.Len(t, paths1, 2)

	// Second batch generation with same texts - should use cache
	paths2, err := service.GenerateBatch(ctx, texts, outputDir)
	require.NoError(t, err)
	assert.Equal(t, paths1, paths2)

	// Verify API was only called once per text
	mockClient.AssertExpectations(t)
}
