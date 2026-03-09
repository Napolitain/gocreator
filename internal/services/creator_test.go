package services

import (
	"context"
	"errors"
	"testing"

	"gocreator/internal/mocks"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestVideoCreator_Create(t *testing.T) {
	t.Run("successful video creation", func(t *testing.T) {
		// Setup mocks
		fs := afero.NewMemMapFs()
		mockText := new(mocks.MockTextProcessor)
		mockTranslation := new(mocks.MockTranslator)
		mockAudio := new(mocks.MockAudioGenerator)
		mockVideo := new(mocks.MockVideoGenerator)
		mockSlide := new(mocks.MockSlideLoader)
		logger := &mockLogger{}

		// Create test data directory structure
		rootDir := testPath("test")
		slidesDir := testPath("test", "data", "slides")
		textsPath := testPath("test", "data", "texts.txt")
		audioDir := testPath("test", "data", "cache", "en", "audio")
		outputPath := testPath("test", "data", "out", "output-en.mp4")
		require.NoError(t, fs.MkdirAll(slidesDir, 0755))
		require.NoError(t, afero.WriteFile(fs, textsPath, []byte("Text 1\n-\nText 2"), 0644))

		// Setup expectations
		inputTexts := []string{"Text 1", "Text 2"}
		slides := []string{testPath("test", "data", "slides", "1.png"), testPath("test", "data", "slides", "2.png")}
		audioPaths := []string{testPath("test", "data", "cache", "en", "audio", "0.mp3"), testPath("test", "data", "cache", "en", "audio", "1.mp3")}

		mockText.On("Load", mock.Anything, textsPath).
			Return(inputTexts, nil)
		mockSlide.On("LoadSlides", mock.Anything, slidesDir).
			Return(slides, nil)
		mockAudio.On("GenerateBatch", mock.Anything, inputTexts, audioDir).
			Return(audioPaths, nil)
		mockVideo.On("GenerateFromSlides", mock.Anything, slides, audioPaths, outputPath).
			Return(nil)

		// Create service
		creator := NewVideoCreator(fs, mockText, mockTranslation, mockAudio, mockVideo, mockSlide, logger)

		// Execute
		cfg := VideoCreatorConfig{
			RootDir:     rootDir,
			InputLang:   "en",
			OutputLangs: []string{"en"},
		}
		err := creator.Create(context.Background(), cfg)

		// Assert
		assert.NoError(t, err)
		mockText.AssertExpectations(t)
		mockSlide.AssertExpectations(t)
		mockAudio.AssertExpectations(t)
		mockVideo.AssertExpectations(t)
	})

	t.Run("video creation with translation", func(t *testing.T) {
		// Setup mocks
		fs := afero.NewMemMapFs()
		mockText := new(mocks.MockTextProcessor)
		mockTranslation := new(mocks.MockTranslator)
		mockAudio := new(mocks.MockAudioGenerator)
		mockVideo := new(mocks.MockVideoGenerator)
		mockSlide := new(mocks.MockSlideLoader)
		logger := &mockLogger{}

		// Create test data directory structure
		rootDir := testPath("test")
		slidesDir := testPath("test", "data", "slides")
		textsPath := testPath("test", "data", "texts.txt")
		translationTextPath := testPath("test", "data", "cache", "es", "text", "texts.txt")
		audioDir := testPath("test", "data", "cache", "es", "audio")
		outputPath := testPath("test", "data", "out", "output-es.mp4")
		require.NoError(t, fs.MkdirAll(slidesDir, 0755))
		require.NoError(t, afero.WriteFile(fs, textsPath, []byte("Hello\n-\nWorld"), 0644))

		// Setup expectations
		inputTexts := []string{"Hello", "World"}
		translatedTexts := []string{"Hola", "Mundo"}
		slides := []string{testPath("test", "data", "slides", "1.png"), testPath("test", "data", "slides", "2.png")}
		audioPaths := []string{testPath("test", "data", "cache", "es", "audio", "0.mp3"), testPath("test", "data", "cache", "es", "audio", "1.mp3")}

		mockText.On("Load", mock.Anything, textsPath).
			Return(inputTexts, nil)
		mockSlide.On("LoadSlides", mock.Anything, slidesDir).
			Return(slides, nil)
		mockTranslation.On("TranslateBatch", mock.Anything, inputTexts, "es").
			Return(translatedTexts, nil)
		mockText.On("Save", mock.Anything, translationTextPath, translatedTexts).
			Return(nil)
		mockAudio.On("GenerateBatch", mock.Anything, translatedTexts, audioDir).
			Return(audioPaths, nil)
		mockVideo.On("GenerateFromSlides", mock.Anything, slides, audioPaths, outputPath).
			Return(nil)

		// Create service
		creator := NewVideoCreator(fs, mockText, mockTranslation, mockAudio, mockVideo, mockSlide, logger)

		// Execute
		cfg := VideoCreatorConfig{
			RootDir:     rootDir,
			InputLang:   "en",
			OutputLangs: []string{"es"},
		}
		err := creator.Create(context.Background(), cfg)

		// Assert
		assert.NoError(t, err)
		mockText.AssertExpectations(t)
		mockTranslation.AssertExpectations(t)
		mockSlide.AssertExpectations(t)
		mockAudio.AssertExpectations(t)
		mockVideo.AssertExpectations(t)
	})

	t.Run("video creation with cached translation", func(t *testing.T) {
		// Setup mocks
		fs := afero.NewMemMapFs()
		mockText := new(mocks.MockTextProcessor)
		mockTranslation := new(mocks.MockTranslator)
		mockAudio := new(mocks.MockAudioGenerator)
		mockVideo := new(mocks.MockVideoGenerator)
		mockSlide := new(mocks.MockSlideLoader)
		logger := &mockLogger{}

		// Create test data directory structure with cached translation
		rootDir := testPath("test")
		slidesDir := testPath("test", "data", "slides")
		textsPath := testPath("test", "data", "texts.txt")
		translationTextPath := testPath("test", "data", "cache", "fr", "text", "texts.txt")
		audioDir := testPath("test", "data", "cache", "fr", "audio")
		outputPath := testPath("test", "data", "out", "output-fr.mp4")
		require.NoError(t, fs.MkdirAll(slidesDir, 0755))
		require.NoError(t, fs.MkdirAll(testPath("test", "data", "cache", "fr", "text"), 0755))
		require.NoError(t, afero.WriteFile(fs, textsPath, []byte("Hello\n-\nWorld"), 0644))
		require.NoError(t, afero.WriteFile(fs, translationTextPath, []byte("Bonjour\n-\nMonde"), 0644))

		// Setup expectations
		inputTexts := []string{"Hello", "World"}
		cachedTexts := []string{"Bonjour", "Monde"}
		slides := []string{testPath("test", "data", "slides", "1.png"), testPath("test", "data", "slides", "2.png")}
		audioPaths := []string{testPath("test", "data", "cache", "fr", "audio", "0.mp3"), testPath("test", "data", "cache", "fr", "audio", "1.mp3")}

		mockText.On("Load", mock.Anything, textsPath).
			Return(inputTexts, nil)
		mockSlide.On("LoadSlides", mock.Anything, slidesDir).
			Return(slides, nil)
		// Translation should load from cache, not translate
		mockText.On("Load", mock.Anything, translationTextPath).
			Return(cachedTexts, nil)
		mockAudio.On("GenerateBatch", mock.Anything, cachedTexts, audioDir).
			Return(audioPaths, nil)
		mockVideo.On("GenerateFromSlides", mock.Anything, slides, audioPaths, outputPath).
			Return(nil)

		// Create service
		creator := NewVideoCreator(fs, mockText, mockTranslation, mockAudio, mockVideo, mockSlide, logger)

		// Execute
		cfg := VideoCreatorConfig{
			RootDir:     rootDir,
			InputLang:   "en",
			OutputLangs: []string{"fr"},
		}
		err := creator.Create(context.Background(), cfg)

		// Assert
		assert.NoError(t, err)
		mockText.AssertExpectations(t)
		mockSlide.AssertExpectations(t)
		mockAudio.AssertExpectations(t)
		mockVideo.AssertExpectations(t)
		// Translation should NOT be called since we used cache
		mockTranslation.AssertNotCalled(t, "TranslateBatch")
	})

	t.Run("error loading texts", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		mockText := new(mocks.MockTextProcessor)
		mockTranslation := new(mocks.MockTranslator)
		mockAudio := new(mocks.MockAudioGenerator)
		mockVideo := new(mocks.MockVideoGenerator)
		mockSlide := new(mocks.MockSlideLoader)
		logger := &mockLogger{}

		mockText.On("Load", mock.Anything, testPath("test", "data", "texts.txt")).
			Return(nil, errors.New("file not found"))

		creator := NewVideoCreator(fs, mockText, mockTranslation, mockAudio, mockVideo, mockSlide, logger)

		cfg := VideoCreatorConfig{
			RootDir:     testPath("test"),
			InputLang:   "en",
			OutputLangs: []string{"en"},
		}
		err := creator.Create(context.Background(), cfg)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load input texts")
	})

	t.Run("error loading slides", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		mockText := new(mocks.MockTextProcessor)
		mockTranslation := new(mocks.MockTranslator)
		mockAudio := new(mocks.MockAudioGenerator)
		mockVideo := new(mocks.MockVideoGenerator)
		mockSlide := new(mocks.MockSlideLoader)
		logger := &mockLogger{}

		mockText.On("Load", mock.Anything, testPath("test", "data", "texts.txt")).
			Return([]string{"Text 1"}, nil)
		mockSlide.On("LoadSlides", mock.Anything, testPath("test", "data", "slides")).
			Return(nil, errors.New("directory not found"))

		creator := NewVideoCreator(fs, mockText, mockTranslation, mockAudio, mockVideo, mockSlide, logger)

		cfg := VideoCreatorConfig{
			RootDir:     testPath("test"),
			InputLang:   "en",
			OutputLangs: []string{"en"},
		}
		err := creator.Create(context.Background(), cfg)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load slides")
	})

	t.Run("error slide and text count mismatch", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		mockText := new(mocks.MockTextProcessor)
		mockTranslation := new(mocks.MockTranslator)
		mockAudio := new(mocks.MockAudioGenerator)
		mockVideo := new(mocks.MockVideoGenerator)
		mockSlide := new(mocks.MockSlideLoader)
		logger := &mockLogger{}

		mockText.On("Load", mock.Anything, testPath("test", "data", "texts.txt")).
			Return([]string{"Text 1", "Text 2"}, nil)
		mockSlide.On("LoadSlides", mock.Anything, testPath("test", "data", "slides")).
			Return([]string{testPath("slide1.png")}, nil) // Only 1 slide but 2 texts

		creator := NewVideoCreator(fs, mockText, mockTranslation, mockAudio, mockVideo, mockSlide, logger)

		cfg := VideoCreatorConfig{
			RootDir:     testPath("test"),
			InputLang:   "en",
			OutputLangs: []string{"en"},
		}
		err := creator.Create(context.Background(), cfg)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "slide and text count mismatch")
	})
}
