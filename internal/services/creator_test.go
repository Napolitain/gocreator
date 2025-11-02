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
		dataDir := "/test/data"
		require.NoError(t, fs.MkdirAll(dataDir+"/slides", 0755))
		require.NoError(t, afero.WriteFile(fs, dataDir+"/texts.txt", []byte("Text 1\n-\nText 2"), 0644))

		// Setup expectations
		inputTexts := []string{"Text 1", "Text 2"}
		slides := []string{"/test/data/slides/1.png", "/test/data/slides/2.png"}
		audioPaths := []string{"/test/data/cache/en/audio/0.mp3", "/test/data/cache/en/audio/1.mp3"}

		mockText.On("Load", mock.Anything, "/test/data/texts.txt").
			Return(inputTexts, nil)
		mockSlide.On("LoadSlides", mock.Anything, "/test/data/slides").
			Return(slides, nil)
		mockAudio.On("GenerateBatch", mock.Anything, inputTexts, "/test/data/cache/en/audio").
			Return(audioPaths, nil)
		mockVideo.On("GenerateFromSlides", mock.Anything, slides, audioPaths, "/test/data/out/output-en.mp4").
			Return(nil)

		// Create service
		creator := NewVideoCreator(fs, mockText, mockTranslation, mockAudio, mockVideo, mockSlide, logger)

		// Execute
		cfg := VideoCreatorConfig{
			RootDir:     "/test",
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
		dataDir := "/test/data"
		require.NoError(t, fs.MkdirAll(dataDir+"/slides", 0755))
		require.NoError(t, afero.WriteFile(fs, dataDir+"/texts.txt", []byte("Hello\n-\nWorld"), 0644))

		// Setup expectations
		inputTexts := []string{"Hello", "World"}
		translatedTexts := []string{"Hola", "Mundo"}
		slides := []string{"/test/data/slides/1.png", "/test/data/slides/2.png"}
		audioPaths := []string{"/test/data/cache/es/audio/0.mp3", "/test/data/cache/es/audio/1.mp3"}

		mockText.On("Load", mock.Anything, "/test/data/texts.txt").
			Return(inputTexts, nil)
		mockSlide.On("LoadSlides", mock.Anything, "/test/data/slides").
			Return(slides, nil)
		mockTranslation.On("TranslateBatch", mock.Anything, inputTexts, "es").
			Return(translatedTexts, nil)
		mockText.On("Save", mock.Anything, "/test/data/cache/es/text/texts.txt", translatedTexts).
			Return(nil)
		mockAudio.On("GenerateBatch", mock.Anything, translatedTexts, "/test/data/cache/es/audio").
			Return(audioPaths, nil)
		mockVideo.On("GenerateFromSlides", mock.Anything, slides, audioPaths, "/test/data/out/output-es.mp4").
			Return(nil)

		// Create service
		creator := NewVideoCreator(fs, mockText, mockTranslation, mockAudio, mockVideo, mockSlide, logger)

		// Execute
		cfg := VideoCreatorConfig{
			RootDir:     "/test",
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
		dataDir := "/test/data"
		require.NoError(t, fs.MkdirAll(dataDir+"/slides", 0755))
		require.NoError(t, fs.MkdirAll(dataDir+"/cache/fr/text", 0755))
		require.NoError(t, afero.WriteFile(fs, dataDir+"/texts.txt", []byte("Hello\n-\nWorld"), 0644))
		require.NoError(t, afero.WriteFile(fs, dataDir+"/cache/fr/text/texts.txt", []byte("Bonjour\n-\nMonde"), 0644))

		// Setup expectations
		inputTexts := []string{"Hello", "World"}
		cachedTexts := []string{"Bonjour", "Monde"}
		slides := []string{"/test/data/slides/1.png", "/test/data/slides/2.png"}
		audioPaths := []string{"/test/data/cache/fr/audio/0.mp3", "/test/data/cache/fr/audio/1.mp3"}

		mockText.On("Load", mock.Anything, "/test/data/texts.txt").
			Return(inputTexts, nil)
		mockSlide.On("LoadSlides", mock.Anything, "/test/data/slides").
			Return(slides, nil)
		// Translation should load from cache, not translate
		mockText.On("Load", mock.Anything, "/test/data/cache/fr/text/texts.txt").
			Return(cachedTexts, nil)
		mockAudio.On("GenerateBatch", mock.Anything, cachedTexts, "/test/data/cache/fr/audio").
			Return(audioPaths, nil)
		mockVideo.On("GenerateFromSlides", mock.Anything, slides, audioPaths, "/test/data/out/output-fr.mp4").
			Return(nil)

		// Create service
		creator := NewVideoCreator(fs, mockText, mockTranslation, mockAudio, mockVideo, mockSlide, logger)

		// Execute
		cfg := VideoCreatorConfig{
			RootDir:     "/test",
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

		mockText.On("Load", mock.Anything, "/test/data/texts.txt").
			Return(nil, errors.New("file not found"))

		creator := NewVideoCreator(fs, mockText, mockTranslation, mockAudio, mockVideo, mockSlide, logger)

		cfg := VideoCreatorConfig{
			RootDir:     "/test",
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

		mockText.On("Load", mock.Anything, "/test/data/texts.txt").
			Return([]string{"Text 1"}, nil)
		mockSlide.On("LoadSlides", mock.Anything, "/test/data/slides").
			Return(nil, errors.New("directory not found"))

		creator := NewVideoCreator(fs, mockText, mockTranslation, mockAudio, mockVideo, mockSlide, logger)

		cfg := VideoCreatorConfig{
			RootDir:     "/test",
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

		mockText.On("Load", mock.Anything, "/test/data/texts.txt").
			Return([]string{"Text 1", "Text 2"}, nil)
		mockSlide.On("LoadSlides", mock.Anything, "/test/data/slides").
			Return([]string{"/slide1.png"}, nil) // Only 1 slide but 2 texts

		creator := NewVideoCreator(fs, mockText, mockTranslation, mockAudio, mockVideo, mockSlide, logger)

		cfg := VideoCreatorConfig{
			RootDir:     "/test",
			InputLang:   "en",
			OutputLangs: []string{"en"},
		}
		err := creator.Create(context.Background(), cfg)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "slide and text count mismatch")
	})
}

func TestVideoCreator_Create_WithGoogleSlides(t *testing.T) {
	t.Run("successful video creation with Google Slides", func(t *testing.T) {
		// Setup mocks
		fs := afero.NewMemMapFs()
		mockText := new(mocks.MockTextProcessor)
		mockTranslation := new(mocks.MockTranslator)
		mockAudio := new(mocks.MockAudioGenerator)
		mockVideo := new(mocks.MockVideoGenerator)
		mockSlide := new(mocks.MockSlideLoader)
		logger := &mockLogger{}

		// Create test data directory structure
		dataDir := "/test/data"
		require.NoError(t, fs.MkdirAll(dataDir, 0755))

		// Setup expectations for Google Slides
		slides := []string{"/test/data/slides/slide_0.png", "/test/data/slides/slide_1.png"}
		notes := []string{"Note 1", "Note 2"}
		audioPaths := []string{"/test/data/cache/en/audio/0.mp3", "/test/data/cache/en/audio/1.mp3"}

		mockSlide.On("LoadFromGoogleSlides", mock.Anything, "test-presentation-id", "/test/data/slides").
			Return(slides, notes, nil)
		mockText.On("Save", mock.Anything, "/test/data/texts.txt", notes).
			Return(nil)
		mockAudio.On("GenerateBatch", mock.Anything, notes, "/test/data/cache/en/audio").
			Return(audioPaths, nil)
		mockVideo.On("GenerateFromSlides", mock.Anything, slides, audioPaths, "/test/data/out/output-en.mp4").
			Return(nil)

		// Create service
		creator := NewVideoCreator(fs, mockText, mockTranslation, mockAudio, mockVideo, mockSlide, logger)

		// Execute
		cfg := VideoCreatorConfig{
			RootDir:        "/test",
			InputLang:      "en",
			OutputLangs:    []string{"en"},
			GoogleSlidesID: "test-presentation-id",
		}
		err := creator.Create(context.Background(), cfg)

		// Assert
		assert.NoError(t, err)
		mockSlide.AssertExpectations(t)
		mockText.AssertExpectations(t)
		mockAudio.AssertExpectations(t)
		mockVideo.AssertExpectations(t)
	})

	t.Run("Google Slides API error", func(t *testing.T) {
		// Setup mocks
		fs := afero.NewMemMapFs()
		mockText := new(mocks.MockTextProcessor)
		mockTranslation := new(mocks.MockTranslator)
		mockAudio := new(mocks.MockAudioGenerator)
		mockVideo := new(mocks.MockVideoGenerator)
		mockSlide := new(mocks.MockSlideLoader)
		logger := &mockLogger{}

		// Setup expectations for Google Slides error
		mockSlide.On("LoadFromGoogleSlides", mock.Anything, "invalid-id", "/test/data/slides").
			Return(nil, nil, errors.New("API error"))

		// Create service
		creator := NewVideoCreator(fs, mockText, mockTranslation, mockAudio, mockVideo, mockSlide, logger)

		// Execute
		cfg := VideoCreatorConfig{
			RootDir:        "/test",
			InputLang:      "en",
			OutputLangs:    []string{"en"},
			GoogleSlidesID: "invalid-id",
		}
		err := creator.Create(context.Background(), cfg)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load Google Slides")
		mockSlide.AssertExpectations(t)
	})
}
