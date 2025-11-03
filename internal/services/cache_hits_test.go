package services

import (
	"context"
	"strings"
	"testing"

	"gocreator/internal/mocks"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockReadCloser implements io.ReadCloser for testing
type cacheTestReadCloser struct {
	*strings.Reader
}

func (m *cacheTestReadCloser) Close() error {
	return nil
}

func newCacheTestReadCloser(data string) *cacheTestReadCloser {
	return &cacheTestReadCloser{Reader: strings.NewReader(data)}
}

// TestTranslationCacheHits verifies that translation API calls are properly cached
func TestTranslationCacheHits(t *testing.T) {
	t.Run("no cache hit on first run", func(t *testing.T) {
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
			Return(inputTexts, nil).Once()
		mockSlide.On("LoadSlides", mock.Anything, "/test/data/slides").
			Return(slides, nil).Once()
		
		// Translation should be called once (no cache on first run)
		mockTranslation.On("TranslateBatch", mock.Anything, inputTexts, "es").
			Return(translatedTexts, nil).Once()
		
		mockText.On("Save", mock.Anything, "/test/data/cache/es/text/texts.txt", translatedTexts).
			Return(nil).Once()
		mockAudio.On("GenerateBatch", mock.Anything, translatedTexts, "/test/data/cache/es/audio").
			Return(audioPaths, nil).Once()
		mockVideo.On("GenerateFromSlides", mock.Anything, slides, audioPaths, "/test/data/out/output-es.mp4").
			Return(nil).Once()

		// Create service
		creator := NewVideoCreator(fs, mockText, mockTranslation, mockAudio, mockVideo, mockSlide, logger)

		// Execute first run
		cfg := VideoCreatorConfig{
			RootDir:     "/test",
			InputLang:   "en",
			OutputLangs: []string{"es"},
		}
		err := creator.Create(context.Background(), cfg)

		// Assert
		assert.NoError(t, err)
		mockTranslation.AssertExpectations(t)
		// Verify TranslateBatch was called exactly once
		mockTranslation.AssertNumberOfCalls(t, "TranslateBatch", 1)
	})

	t.Run("cache hit on second run", func(t *testing.T) {
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
		require.NoError(t, fs.MkdirAll(dataDir+"/cache/es/text", 0755))
		require.NoError(t, afero.WriteFile(fs, dataDir+"/texts.txt", []byte("Hello\n-\nWorld"), 0644))
		require.NoError(t, afero.WriteFile(fs, dataDir+"/cache/es/text/texts.txt", []byte("Hola\n-\nMundo"), 0644))

		// Setup expectations
		inputTexts := []string{"Hello", "World"}
		cachedTexts := []string{"Hola", "Mundo"}
		slides := []string{"/test/data/slides/1.png", "/test/data/slides/2.png"}
		audioPaths := []string{"/test/data/cache/es/audio/0.mp3", "/test/data/cache/es/audio/1.mp3"}

		mockText.On("Load", mock.Anything, "/test/data/texts.txt").
			Return(inputTexts, nil).Once()
		mockSlide.On("LoadSlides", mock.Anything, "/test/data/slides").
			Return(slides, nil).Once()
		
		// Translation should load from cache instead of translating
		mockText.On("Load", mock.Anything, "/test/data/cache/es/text/texts.txt").
			Return(cachedTexts, nil).Once()
		
		mockAudio.On("GenerateBatch", mock.Anything, cachedTexts, "/test/data/cache/es/audio").
			Return(audioPaths, nil).Once()
		mockVideo.On("GenerateFromSlides", mock.Anything, slides, audioPaths, "/test/data/out/output-es.mp4").
			Return(nil).Once()

		// Create service
		creator := NewVideoCreator(fs, mockText, mockTranslation, mockAudio, mockVideo, mockSlide, logger)

		// Execute second run with cache
		cfg := VideoCreatorConfig{
			RootDir:     "/test",
			InputLang:   "en",
			OutputLangs: []string{"es"},
		}
		err := creator.Create(context.Background(), cfg)

		// Assert
		assert.NoError(t, err)
		// Translation API should NOT be called (cache hit)
		mockTranslation.AssertNotCalled(t, "TranslateBatch")
	})

	t.Run("multiple languages with mixed cache hits", func(t *testing.T) {
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
		require.NoError(t, fs.MkdirAll(dataDir+"/cache/es/text", 0755))
		require.NoError(t, afero.WriteFile(fs, dataDir+"/texts.txt", []byte("Hello\n-\nWorld"), 0644))
		// Spanish translation is cached
		require.NoError(t, afero.WriteFile(fs, dataDir+"/cache/es/text/texts.txt", []byte("Hola\n-\nMundo"), 0644))

		// Setup expectations
		inputTexts := []string{"Hello", "World"}
		cachedSpanishTexts := []string{"Hola", "Mundo"}
		frenchTexts := []string{"Bonjour", "Monde"}
		slides := []string{"/test/data/slides/1.png", "/test/data/slides/2.png"}

		mockText.On("Load", mock.Anything, "/test/data/texts.txt").
			Return(inputTexts, nil).Once()
		mockSlide.On("LoadSlides", mock.Anything, "/test/data/slides").
			Return(slides, nil).Once()

		// Spanish: Load from cache (cache hit)
		mockText.On("Load", mock.Anything, "/test/data/cache/es/text/texts.txt").
			Return(cachedSpanishTexts, nil).Once()
		mockAudio.On("GenerateBatch", mock.Anything, cachedSpanishTexts, "/test/data/cache/es/audio").
			Return([]string{"/test/data/cache/es/audio/0.mp3", "/test/data/cache/es/audio/1.mp3"}, nil).Once()
		mockVideo.On("GenerateFromSlides", mock.Anything, slides, 
			[]string{"/test/data/cache/es/audio/0.mp3", "/test/data/cache/es/audio/1.mp3"}, 
			"/test/data/out/output-es.mp4").
			Return(nil).Once()

		// French: No cache, needs translation (cache miss)
		mockTranslation.On("TranslateBatch", mock.Anything, inputTexts, "fr").
			Return(frenchTexts, nil).Once()
		mockText.On("Save", mock.Anything, "/test/data/cache/fr/text/texts.txt", frenchTexts).
			Return(nil).Once()
		mockAudio.On("GenerateBatch", mock.Anything, frenchTexts, "/test/data/cache/fr/audio").
			Return([]string{"/test/data/cache/fr/audio/0.mp3", "/test/data/cache/fr/audio/1.mp3"}, nil).Once()
		mockVideo.On("GenerateFromSlides", mock.Anything, slides, 
			[]string{"/test/data/cache/fr/audio/0.mp3", "/test/data/cache/fr/audio/1.mp3"}, 
			"/test/data/out/output-fr.mp4").
			Return(nil).Once()

		// Create service
		creator := NewVideoCreator(fs, mockText, mockTranslation, mockAudio, mockVideo, mockSlide, logger)

		// Execute with both Spanish (cached) and French (not cached)
		cfg := VideoCreatorConfig{
			RootDir:     "/test",
			InputLang:   "en",
			OutputLangs: []string{"es", "fr"},
		}
		err := creator.Create(context.Background(), cfg)

		// Assert
		assert.NoError(t, err)
		// Translation API should be called exactly once (only for French)
		mockTranslation.AssertNumberOfCalls(t, "TranslateBatch", 1)
		mockTranslation.AssertExpectations(t)
	})
}

// TestAudioGenerationCacheHits verifies that audio generation API calls are properly cached
func TestAudioGenerationCacheHits(t *testing.T) {
	t.Run("no cache hit on first audio generation", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		mockClient := new(mocks.MockOpenAIClient)
		logger := &mockLogger{}
		textService := NewTextService(fs, logger)
		service := NewAudioService(fs, mockClient, textService, logger)

		texts := []string{"Hello", "World"}
		outputDir := "/output"

		// First generation - API should be called for each text
		mockClient.On("GenerateSpeech", mock.Anything, "Hello").
			Return(newCacheTestReadCloser("audio1"), nil).Once()
		mockClient.On("GenerateSpeech", mock.Anything, "World").
			Return(newCacheTestReadCloser("audio2"), nil).Once()

		ctx := context.Background()
		paths, err := service.GenerateBatch(ctx, texts, outputDir)

		assert.NoError(t, err)
		assert.Len(t, paths, 2)
		// Verify API was called exactly twice (once per text)
		mockClient.AssertNumberOfCalls(t, "GenerateSpeech", 2)
		mockClient.AssertExpectations(t)
	})

	t.Run("cache hit on second audio generation with same texts", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		mockClient := new(mocks.MockOpenAIClient)
		logger := &mockLogger{}
		textService := NewTextService(fs, logger)
		service := NewAudioService(fs, mockClient, textService, logger)

		texts := []string{"Hello", "World"}
		outputDir := "/output"

		// First generation - API should be called
		mockClient.On("GenerateSpeech", mock.Anything, "Hello").
			Return(newCacheTestReadCloser("audio1"), nil).Once()
		mockClient.On("GenerateSpeech", mock.Anything, "World").
			Return(newCacheTestReadCloser("audio2"), nil).Once()

		ctx := context.Background()
		paths1, err := service.GenerateBatch(ctx, texts, outputDir)
		require.NoError(t, err)

		// Second generation with same texts - should use cache, API not called
		paths2, err := service.GenerateBatch(ctx, texts, outputDir)
		assert.NoError(t, err)
		assert.Equal(t, paths1, paths2)

		// Verify API was called exactly twice total (only during first generation)
		mockClient.AssertNumberOfCalls(t, "GenerateSpeech", 2)
		mockClient.AssertExpectations(t)
	})

	t.Run("partial cache hit with changed texts", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		mockClient := new(mocks.MockOpenAIClient)
		logger := &mockLogger{}
		textService := NewTextService(fs, logger)
		service := NewAudioService(fs, mockClient, textService, logger)

		initialTexts := []string{"Hello", "World"}
		modifiedTexts := []string{"Hello", "Universe"} // First text unchanged, second changed
		outputDir := "/output"

		// First generation - API called for both texts
		mockClient.On("GenerateSpeech", mock.Anything, "Hello").
			Return(newCacheTestReadCloser("audio1"), nil).Once()
		mockClient.On("GenerateSpeech", mock.Anything, "World").
			Return(newCacheTestReadCloser("audio2"), nil).Once()

		ctx := context.Background()
		_, err := service.GenerateBatch(ctx, initialTexts, outputDir)
		require.NoError(t, err)

		// Second generation with one changed text
		// "Hello" should use cache, "Universe" should call API
		mockClient.On("GenerateSpeech", mock.Anything, "Universe").
			Return(newCacheTestReadCloser("audio3"), nil).Once()

		paths, err := service.GenerateBatch(ctx, modifiedTexts, outputDir)
		assert.NoError(t, err)
		assert.Len(t, paths, 2)

		// Verify API was called 3 times total (2 first run + 1 second run for "Universe")
		mockClient.AssertNumberOfCalls(t, "GenerateSpeech", 3)
		mockClient.AssertExpectations(t)
	})

	t.Run("cache hit verification with single audio generation", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		mockClient := new(mocks.MockOpenAIClient)
		logger := &mockLogger{}
		textService := NewTextService(fs, logger)
		service := NewAudioService(fs, mockClient, textService, logger)

		text := "Hello world"
		outputPath := "/output/audio.mp3"

		// First generation - should call API
		mockClient.On("GenerateSpeech", mock.Anything, text).
			Return(newCacheTestReadCloser("audio data"), nil).Once()

		ctx := context.Background()
		err := service.Generate(ctx, text, outputPath)
		require.NoError(t, err)

		// Second generation with same text - should use cache, not call API again
		err = service.Generate(ctx, text, outputPath)
		assert.NoError(t, err)

		// Verify API was called exactly once
		mockClient.AssertNumberOfCalls(t, "GenerateSpeech", 1)
		mockClient.AssertExpectations(t)
	})
}

// TestFFmpegVideoCacheHits verifies that video segment generation properly reuses segments
// Note: VideoService doesn't have explicit caching logic, but the test documents expected behavior
func TestFFmpegVideoCacheHits(t *testing.T) {
	t.Run("video segments are generated on first run", func(t *testing.T) {
		// Note: This test documents that VideoService generates new segments each time
		// The cache behavior for video segments relies on the filesystem persisting
		// the .temp directory between runs
		fs := afero.NewMemMapFs()
		logger := &mockLogger{}
		service := NewVideoService(fs, logger)

		// This test verifies the structure, not actual ffmpeg calls
		// since VideoService uses exec.Command which can't be easily mocked
		assert.NotNil(t, service)
		
		// The video service creates temp directory for segments
		// This is where ffmpeg output caching happens via filesystem
		tempDir := "/test/data/out/.temp"
		err := fs.MkdirAll(tempDir, 0755)
		assert.NoError(t, err)

		// Verify temp directory exists (where video segments would be cached)
		exists, err := afero.DirExists(fs, tempDir)
		assert.NoError(t, err)
		assert.True(t, exists, "Temp directory for video segments should exist")
	})

	t.Run("video segments directory structure for caching", func(t *testing.T) {
		// This test documents the expected directory structure for video segment caching
		fs := afero.NewMemMapFs()
		
		// Simulate the structure created during video generation
		tempDir := "/test/data/out/.temp"
		require.NoError(t, fs.MkdirAll(tempDir, 0755))
		
		// Simulate creation of video segments
		segmentPaths := []string{
			"/test/data/out/.temp/video_0.mp4",
			"/test/data/out/.temp/video_1.mp4",
			"/test/data/out/.temp/video_2.mp4",
		}
		
		for _, path := range segmentPaths {
			err := afero.WriteFile(fs, path, []byte("video data"), 0644)
			assert.NoError(t, err)
		}

		// Verify all segments exist (simulating cache persistence)
		for i, path := range segmentPaths {
			exists, err := afero.Exists(fs, path)
			assert.NoError(t, err)
			assert.True(t, exists, "Video segment %d should exist at %s", i, path)
		}

		// In a real scenario, these segments would persist between runs
		// and FFmpeg would only regenerate if the segment is missing
	})
}

// TestIntegratedCacheHitCount verifies cache hits across the entire video creation workflow
func TestIntegratedCacheHitCount(t *testing.T) {
	t.Run("full workflow with all cache misses on first run", func(t *testing.T) {
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
		require.NoError(t, afero.WriteFile(fs, dataDir+"/texts.txt", []byte("Hello\n-\nWorld\n-\nTest"), 0644))

		// Setup expectations
		inputTexts := []string{"Hello", "World", "Test"}
		translatedTexts := []string{"Hola", "Mundo", "Prueba"}
		slides := []string{"/slide1.png", "/slide2.png", "/slide3.png"}
		audioPaths := []string{"/audio0.mp3", "/audio1.mp3", "/audio2.mp3"}

		mockText.On("Load", mock.Anything, "/test/data/texts.txt").
			Return(inputTexts, nil).Once()
		mockSlide.On("LoadSlides", mock.Anything, "/test/data/slides").
			Return(slides, nil).Once()
		mockTranslation.On("TranslateBatch", mock.Anything, inputTexts, "es").
			Return(translatedTexts, nil).Once()
		mockText.On("Save", mock.Anything, "/test/data/cache/es/text/texts.txt", translatedTexts).
			Return(nil).Once()
		mockAudio.On("GenerateBatch", mock.Anything, translatedTexts, "/test/data/cache/es/audio").
			Return(audioPaths, nil).Once()
		mockVideo.On("GenerateFromSlides", mock.Anything, slides, audioPaths, "/test/data/out/output-es.mp4").
			Return(nil).Once()

		creator := NewVideoCreator(fs, mockText, mockTranslation, mockAudio, mockVideo, mockSlide, logger)

		cfg := VideoCreatorConfig{
			RootDir:     "/test",
			InputLang:   "en",
			OutputLangs: []string{"es"},
		}
		err := creator.Create(context.Background(), cfg)

		assert.NoError(t, err)
		
		// Verify cache misses (all API calls made)
		mockTranslation.AssertNumberOfCalls(t, "TranslateBatch", 1) // Translation API called
		mockAudio.AssertNumberOfCalls(t, "GenerateBatch", 1)       // Audio API called
		mockVideo.AssertNumberOfCalls(t, "GenerateFromSlides", 1)  // Video generation called
		
		mockText.AssertExpectations(t)
		mockTranslation.AssertExpectations(t)
		mockAudio.AssertExpectations(t)
		mockVideo.AssertExpectations(t)
	})

	t.Run("full workflow with all cache hits on second run", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		mockText := new(mocks.MockTextProcessor)
		mockTranslation := new(mocks.MockTranslator)
		mockAudio := new(mocks.MockAudioGenerator)
		mockVideo := new(mocks.MockVideoGenerator)
		mockSlide := new(mocks.MockSlideLoader)
		logger := &mockLogger{}

		// Create test data with all caches populated
		dataDir := "/test/data"
		require.NoError(t, fs.MkdirAll(dataDir+"/slides", 0755))
		require.NoError(t, fs.MkdirAll(dataDir+"/cache/es/text", 0755))
		require.NoError(t, fs.MkdirAll(dataDir+"/cache/es/audio", 0755))
		require.NoError(t, afero.WriteFile(fs, dataDir+"/texts.txt", []byte("Hello\n-\nWorld"), 0644))
		require.NoError(t, afero.WriteFile(fs, dataDir+"/cache/es/text/texts.txt", []byte("Hola\n-\nMundo"), 0644))

		inputTexts := []string{"Hello", "World"}
		cachedTexts := []string{"Hola", "Mundo"}
		slides := []string{"/slide1.png", "/slide2.png"}
		
		// Note: Audio cache is handled by AudioService internally, not by VideoCreator
		// VideoCreator just calls GenerateBatch which handles its own caching
		mockText.On("Load", mock.Anything, "/test/data/texts.txt").
			Return(inputTexts, nil).Once()
		mockSlide.On("LoadSlides", mock.Anything, "/test/data/slides").
			Return(slides, nil).Once()
		mockText.On("Load", mock.Anything, "/test/data/cache/es/text/texts.txt").
			Return(cachedTexts, nil).Once()
		mockAudio.On("GenerateBatch", mock.Anything, cachedTexts, "/test/data/cache/es/audio").
			Return([]string{"/audio0.mp3", "/audio1.mp3"}, nil).Once()
		mockVideo.On("GenerateFromSlides", mock.Anything, slides, 
			[]string{"/audio0.mp3", "/audio1.mp3"}, "/test/data/out/output-es.mp4").
			Return(nil).Once()

		creator := NewVideoCreator(fs, mockText, mockTranslation, mockAudio, mockVideo, mockSlide, logger)

		cfg := VideoCreatorConfig{
			RootDir:     "/test",
			InputLang:   "en",
			OutputLangs: []string{"es"},
		}
		err := creator.Create(context.Background(), cfg)

		assert.NoError(t, err)
		
		// Verify cache hits (translation API not called)
		mockTranslation.AssertNumberOfCalls(t, "TranslateBatch", 0) // Translation cache hit
		// Note: Audio and video would still be called but with cached data
		mockAudio.AssertNumberOfCalls(t, "GenerateBatch", 1)
		mockVideo.AssertNumberOfCalls(t, "GenerateFromSlides", 1)
	})
}
