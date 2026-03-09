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

func TestVideoCreatorCreate_UsesSourceTextSidecars(t *testing.T) {
	fs := afero.NewMemMapFs()
	mockText := new(mocks.MockTextProcessor)
	mockTranslation := new(mocks.MockTranslator)
	mockAudio := new(mocks.MockAudioGenerator)
	mockVideo := new(mocks.MockVideoGenerator)
	mockSlide := new(mocks.MockSlideLoader)
	logger := &mockLogger{}

	rootDir := testPath("test")
	slidesDir := testPath("test", "data", "slides")
	outputPath := testPath("test", "data", "out", "output-en.mp4")
	slides := []string{
		testPath("test", "data", "slides", "1.png"),
		testPath("test", "data", "slides", "2.png"),
	}
	audioPaths := []string{
		testPath("test", "data", "cache", "en", "audio", "0.mp3"),
		testPath("test", "data", "cache", "en", "audio", "1.mp3"),
	}

	require.NoError(t, fs.MkdirAll(slidesDir, 0755))
	require.NoError(t, afero.WriteFile(fs, slides[0], []byte("slide1"), 0644))
	require.NoError(t, afero.WriteFile(fs, slides[1], []byte("slide2"), 0644))
	require.NoError(t, afero.WriteFile(fs, testPath("test", "data", "slides", "1.txt"), []byte("Text 1"), 0644))
	require.NoError(t, afero.WriteFile(fs, testPath("test", "data", "slides", "2.txt"), []byte("Text 2"), 0644))

	mockSlide.On("LoadSlides", mock.Anything, slidesDir).Return(slides, nil).Once()
	mockAudio.On("Generate", mock.Anything, "Text 1", audioPaths[0]).Return(nil).Once()
	mockAudio.On("Generate", mock.Anything, "Text 2", audioPaths[1]).Return(nil).Once()
	mockVideo.On("GenerateFromSlides", mock.Anything, slides, audioPaths, outputPath).Return(nil).Once()

	creator := NewVideoCreator(fs, mockText, mockTranslation, mockAudio, mockVideo, mockSlide, logger)
	err := creator.Create(context.Background(), VideoCreatorConfig{
		RootDir:     rootDir,
		InputLang:   "en",
		OutputLangs: []string{"en"},
	})

	require.NoError(t, err)
	mockTranslation.AssertNotCalled(t, "TranslateBatch")
	mockSlide.AssertExpectations(t)
	mockAudio.AssertExpectations(t)
	mockVideo.AssertExpectations(t)
}

func TestVideoCreatorCreate_TranslatesOnlySlidesWithoutLanguageOverride(t *testing.T) {
	fs := afero.NewMemMapFs()
	mockText := new(mocks.MockTextProcessor)
	mockTranslation := new(mocks.MockTranslator)
	mockAudio := new(mocks.MockAudioGenerator)
	mockVideo := new(mocks.MockVideoGenerator)
	mockSlide := new(mocks.MockSlideLoader)
	logger := &mockLogger{}

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

	require.NoError(t, fs.MkdirAll(slidesDir, 0755))
	require.NoError(t, afero.WriteFile(fs, slides[0], []byte("slide1"), 0644))
	require.NoError(t, afero.WriteFile(fs, slides[1], []byte("slide2"), 0644))
	require.NoError(t, afero.WriteFile(fs, testPath("test", "data", "slides", "1.txt"), []byte("Hello"), 0644))
	require.NoError(t, afero.WriteFile(fs, testPath("test", "data", "slides", "2.txt"), []byte("World"), 0644))
	require.NoError(t, afero.WriteFile(fs, testPath("test", "data", "slides", "2.es.txt"), []byte("Mundo directo"), 0644))

	mockSlide.On("LoadSlides", mock.Anything, slidesDir).Return(slides, nil).Once()
	mockTranslation.On("TranslateBatch", mock.Anything, []string{"Hello"}, "es").Return([]string{"Hola"}, nil).Once()
	mockAudio.On("Generate", mock.Anything, "Hola", audioPaths[0]).Return(nil).Once()
	mockAudio.On("Generate", mock.Anything, "Mundo directo", audioPaths[1]).Return(nil).Once()
	mockVideo.On("GenerateFromSlides", mock.Anything, slides, audioPaths, outputPath).Return(nil).Once()

	creator := NewVideoCreator(fs, mockText, mockTranslation, mockAudio, mockVideo, mockSlide, logger)
	err := creator.Create(context.Background(), VideoCreatorConfig{
		RootDir:     rootDir,
		InputLang:   "en",
		OutputLangs: []string{"es"},
	})

	require.NoError(t, err)
	mockSlide.AssertExpectations(t)
	mockTranslation.AssertExpectations(t)
	mockAudio.AssertExpectations(t)
	mockVideo.AssertExpectations(t)
}

func TestVideoCreatorCreate_UsesPrerecordedAudioPerSlide(t *testing.T) {
	fs := afero.NewMemMapFs()
	mockText := new(mocks.MockTextProcessor)
	mockTranslation := new(mocks.MockTranslator)
	mockAudio := new(mocks.MockAudioGenerator)
	mockVideo := new(mocks.MockVideoGenerator)
	mockSlide := new(mocks.MockSlideLoader)
	logger := &mockLogger{}

	rootDir := testPath("test")
	slidesDir := testPath("test", "data", "slides")
	outputPath := testPath("test", "data", "out", "output-es.mp4")
	slides := []string{
		testPath("test", "data", "slides", "1.png"),
		testPath("test", "data", "slides", "2.png"),
	}
	audioPaths := []string{
		testPath("test", "data", "cache", "es", "audio", "0.mp3"),
		testPath("test", "data", "slides", "2.es.wav"),
	}

	require.NoError(t, fs.MkdirAll(slidesDir, 0755))
	require.NoError(t, afero.WriteFile(fs, slides[0], []byte("slide1"), 0644))
	require.NoError(t, afero.WriteFile(fs, slides[1], []byte("slide2"), 0644))
	require.NoError(t, afero.WriteFile(fs, testPath("test", "data", "slides", "1.txt"), []byte("Hello"), 0644))
	require.NoError(t, afero.WriteFile(fs, testPath("test", "data", "slides", "2.es.wav"), []byte("audio"), 0644))

	mockSlide.On("LoadSlides", mock.Anything, slidesDir).Return(slides, nil).Once()
	mockTranslation.On("TranslateBatch", mock.Anything, []string{"Hello"}, "es").Return([]string{"Hola"}, nil).Once()
	mockAudio.On("Generate", mock.Anything, "Hola", audioPaths[0]).Return(nil).Once()
	mockVideo.On("GenerateFromSlides", mock.Anything, slides, audioPaths, outputPath).Return(nil).Once()

	creator := NewVideoCreator(fs, mockText, mockTranslation, mockAudio, mockVideo, mockSlide, logger)
	err := creator.Create(context.Background(), VideoCreatorConfig{
		RootDir:     rootDir,
		InputLang:   "en",
		OutputLangs: []string{"es"},
	})

	require.NoError(t, err)
	mockSlide.AssertExpectations(t)
	mockTranslation.AssertExpectations(t)
	mockAudio.AssertExpectations(t)
	mockVideo.AssertExpectations(t)
}

func TestVideoCreatorCreate_UsesGenericAudioForInputLanguage(t *testing.T) {
	fs := afero.NewMemMapFs()
	mockText := new(mocks.MockTextProcessor)
	mockTranslation := new(mocks.MockTranslator)
	mockAudio := new(mocks.MockAudioGenerator)
	mockVideo := new(mocks.MockVideoGenerator)
	mockSlide := new(mocks.MockSlideLoader)
	logger := &mockLogger{}

	rootDir := testPath("test")
	slidesDir := testPath("test", "data", "slides")
	outputPath := testPath("test", "data", "out", "output-en.mp4")
	slides := []string{
		testPath("test", "data", "slides", "1.png"),
		testPath("test", "data", "slides", "2.png"),
	}
	audioPaths := []string{
		testPath("test", "data", "slides", "1.wav"),
		testPath("test", "data", "cache", "en", "audio", "1.mp3"),
	}

	require.NoError(t, fs.MkdirAll(slidesDir, 0755))
	require.NoError(t, afero.WriteFile(fs, slides[0], []byte("slide1"), 0644))
	require.NoError(t, afero.WriteFile(fs, slides[1], []byte("slide2"), 0644))
	require.NoError(t, afero.WriteFile(fs, testPath("test", "data", "slides", "1.wav"), []byte("audio"), 0644))
	require.NoError(t, afero.WriteFile(fs, testPath("test", "data", "slides", "2.txt"), []byte("Text 2"), 0644))

	mockSlide.On("LoadSlides", mock.Anything, slidesDir).Return(slides, nil).Once()
	mockAudio.On("Generate", mock.Anything, "Text 2", audioPaths[1]).Return(nil).Once()
	mockVideo.On("GenerateFromSlides", mock.Anything, slides, audioPaths, outputPath).Return(nil).Once()

	creator := NewVideoCreator(fs, mockText, mockTranslation, mockAudio, mockVideo, mockSlide, logger)
	err := creator.Create(context.Background(), VideoCreatorConfig{
		RootDir:     rootDir,
		InputLang:   "en",
		OutputLangs: []string{"en"},
	})

	require.NoError(t, err)
	mockTranslation.AssertNotCalled(t, "TranslateBatch")
	mockAudio.AssertExpectations(t)
	mockVideo.AssertExpectations(t)
}

func TestVideoCreatorCreate_FailsWhenNarrationIsMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	mockText := new(mocks.MockTextProcessor)
	mockTranslation := new(mocks.MockTranslator)
	mockAudio := new(mocks.MockAudioGenerator)
	mockVideo := new(mocks.MockVideoGenerator)
	mockSlide := new(mocks.MockSlideLoader)
	logger := &mockLogger{}

	rootDir := testPath("test")
	slidesDir := testPath("test", "data", "slides")
	slides := []string{testPath("test", "data", "slides", "1.png")}

	require.NoError(t, fs.MkdirAll(slidesDir, 0755))
	require.NoError(t, afero.WriteFile(fs, slides[0], []byte("slide1"), 0644))

	mockSlide.On("LoadSlides", mock.Anything, slidesDir).Return(slides, nil).Once()

	creator := NewVideoCreator(fs, mockText, mockTranslation, mockAudio, mockVideo, mockSlide, logger)
	err := creator.Create(context.Background(), VideoCreatorConfig{
		RootDir:     rootDir,
		InputLang:   "en",
		OutputLangs: []string{"en"},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no matching text or audio sidecar")
	mockVideo.AssertNotCalled(t, "GenerateFromSlides")
}

func TestVideoCreatorCreate_FailsWhenSlideLoadingFails(t *testing.T) {
	fs := afero.NewMemMapFs()
	mockText := new(mocks.MockTextProcessor)
	mockTranslation := new(mocks.MockTranslator)
	mockAudio := new(mocks.MockAudioGenerator)
	mockVideo := new(mocks.MockVideoGenerator)
	mockSlide := new(mocks.MockSlideLoader)
	logger := &mockLogger{}

	mockSlide.On("LoadSlides", mock.Anything, testPath("test", "data", "slides")).Return(nil, errors.New("directory not found")).Once()

	creator := NewVideoCreator(fs, mockText, mockTranslation, mockAudio, mockVideo, mockSlide, logger)
	err := creator.Create(context.Background(), VideoCreatorConfig{
		RootDir:     testPath("test"),
		InputLang:   "en",
		OutputLangs: []string{"en"},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load slides")
}
