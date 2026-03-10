package services

import (
	"context"
	"testing"

	"gocreator/internal/config"
	"gocreator/internal/interfaces"
	"gocreator/internal/mocks"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestVideoCreatorCreate_UsesVoiceOverridesForConcreteAudioService(t *testing.T) {
	fs := afero.NewMemMapFs()
	mockTranslation := new(mocks.MockTranslator)
	mockOpenAI := new(mocks.MockOpenAIClient)
	mockVideo := new(mocks.MockVideoGenerator)
	mockSlide := new(mocks.MockSlideLoader)
	logger := &mockLogger{}
	textService := NewTextService(fs, logger)
	audioService := NewAudioService(fs, mockOpenAI, textService, logger)

	rootDir := testPath("test")
	slidesDir := testPath("test", "data", "slides")
	outputPath := testPath("test", "data", "out", "output-es.mp4")
	slides := []string{
		testPath("test", "data", "slides", "1.png"),
		testPath("test", "data", "slides", "2.png"),
	}
	audioPaths := []string{
		testPath("test", "data", "cache", "es", "audio", "0.mp3"),
		testPath("test", "data", "cache", "es", "audio", "1.mp3"),
	}

	require.NoError(t, fs.MkdirAll(slidesDir, 0o755))
	require.NoError(t, afero.WriteFile(fs, slides[0], []byte("slide1"), 0o644))
	require.NoError(t, afero.WriteFile(fs, slides[1], []byte("slide2"), 0o644))
	require.NoError(t, afero.WriteFile(fs, testPath("test", "data", "slides", "1.txt"), []byte("Hello"), 0o644))
	require.NoError(t, afero.WriteFile(fs, testPath("test", "data", "slides", "2.txt"), []byte("World"), 0o644))

	expectedOptions := interfaces.SpeechOptions{
		Model: "tts-1",
		Voice: "nova",
		Speed: 1.25,
	}

	mockSlide.On("LoadSlides", mock.Anything, slidesDir).Return(slides, nil).Once()
	mockTranslation.On("TranslateBatch", mock.Anything, []string{"Hello", "World"}, "es").Return([]string{"Hola", "Mundo"}, nil).Once()
	mockOpenAI.On("GenerateSpeechWithOptions", mock.Anything, "Hola", expectedOptions).Return(newMockReadCloser("audio-1"), nil).Once()
	mockOpenAI.On("GenerateSpeechWithOptions", mock.Anything, "Mundo", expectedOptions).Return(newMockReadCloser("audio-2"), nil).Once()
	mockVideo.On("GenerateFromSlides", mock.Anything, slides, audioPaths, outputPath).Return(nil).Once()

	creator := NewVideoCreator(fs, textService, mockTranslation, audioService, mockVideo, mockSlide, logger)
	err := creator.Create(context.Background(), VideoCreatorConfig{
		RootDir:     rootDir,
		InputLang:   "en",
		OutputLangs: []string{"es"},
		Voice: config.VoiceConfig{
			Model: "tts-1",
			Voice: "alloy",
			Speed: 1.0,
			PerLanguage: map[string]config.VoiceSetup{
				"es": {
					Voice: "nova",
					Speed: 1.25,
				},
			},
		},
	})

	require.NoError(t, err)
	mockSlide.AssertExpectations(t)
	mockTranslation.AssertExpectations(t)
	mockOpenAI.AssertExpectations(t)
	mockOpenAI.AssertNotCalled(t, "GenerateSpeech", mock.Anything, mock.Anything)
	mockVideo.AssertExpectations(t)
}

func TestNeedsPostProcess(t *testing.T) {
	t.Run("defaults stay on fast path", func(t *testing.T) {
		assert.False(t, needsPostProcess(VideoCreatorConfig{}, "en"))
	})

	t.Run("subtitle language mismatch keeps fast path", func(t *testing.T) {
		cfg := VideoCreatorConfig{
			Subtitles: config.SubtitlesConfig{
				Enabled:   true,
				Languages: []string{"es"},
			},
		}
		assert.False(t, needsPostProcess(cfg, "en"))
		assert.True(t, needsPostProcess(cfg, "es"))
	})

	t.Run("non default output or audio features enable post processing", func(t *testing.T) {
		assert.True(t, needsPostProcess(VideoCreatorConfig{
			Output: config.OutputConfig{Format: "webm"},
		}, "en"))
		assert.True(t, needsPostProcess(VideoCreatorConfig{
			Audio: config.AudioConfig{
				BackgroundMusic: config.BackgroundMusicConfig{
					Enabled: true,
					File:    "music.mp3",
				},
			},
		}, "en"))
	})
}
