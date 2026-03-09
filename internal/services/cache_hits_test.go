package services

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"

	"gocreator/internal/mocks"

	"github.com/openai/openai-go/v3"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type translationCacheClient struct {
	mu        sync.Mutex
	responses map[string]string
	callCount int
}

func (c *translationCacheClient) ChatCompletion(ctx context.Context, messages []openai.ChatCompletionMessageParamUnion) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var payload string
	for _, message := range messages {
		if message.OfUser != nil && message.OfUser.Content.OfString.Valid() {
			payload = message.OfUser.Content.OfString.Value
			break
		}
	}
	if payload == "" {
		payload = fmt.Sprintf("%v", messages)
	}
	for needle, translated := range c.responses {
		if strings.Contains(payload, needle) {
			c.callCount++
			return translated, nil
		}
	}

	return "", fmt.Errorf("unexpected translation payload: %s", payload)
}

func (c *translationCacheClient) GenerateSpeech(ctx context.Context, text string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("GenerateSpeech should not be called in translation cache tests")
}

// cacheTestReadCloser implements io.ReadCloser for testing
type cacheTestReadCloser struct {
	*strings.Reader
}

func (m *cacheTestReadCloser) Close() error {
	return nil
}

func newCacheTestReadCloser(data string) *cacheTestReadCloser {
	return &cacheTestReadCloser{Reader: strings.NewReader(data)}
}

// TestTranslationCacheHits verifies that translation API calls are properly cached.
func TestTranslationCacheHits(t *testing.T) {
	t.Run("disk cache hit on second run", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		client := &translationCacheClient{responses: map[string]string{
			"Hello": "Hola",
			"World": "Mundo",
		}}
		logger := &mockLogger{}
		cacheDir := testPath("cache", "translations")

		service1 := NewTranslationServiceWithCache(client, logger, fs, cacheDir)
		service2 := NewTranslationServiceWithCache(client, logger, fs, cacheDir)

		texts := []string{"Hello", "World"}
		ctx := context.Background()

		firstRun, err := service1.TranslateBatch(ctx, texts, "es")
		require.NoError(t, err)
		assert.Equal(t, []string{"Hola", "Mundo"}, firstRun)

		secondRun, err := service2.TranslateBatch(ctx, texts, "es")
		require.NoError(t, err)
		assert.Equal(t, []string{"Hola", "Mundo"}, secondRun)

		assert.Equal(t, 2, client.callCount)
	})

	t.Run("partial cache hit only translates changed text", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		client := &translationCacheClient{responses: map[string]string{
			"Hello":    "Hola",
			"World":    "Mundo",
			"Universe": "Universo",
		}}
		logger := &mockLogger{}
		cacheDir := testPath("cache", "translations")

		service1 := NewTranslationServiceWithCache(client, logger, fs, cacheDir)
		service2 := NewTranslationServiceWithCache(client, logger, fs, cacheDir)

		ctx := context.Background()
		_, err := service1.TranslateBatch(ctx, []string{"Hello", "World"}, "es")
		require.NoError(t, err)

		translated, err := service2.TranslateBatch(ctx, []string{"Hello", "Universe"}, "es")
		require.NoError(t, err)
		assert.Equal(t, []string{"Hola", "Universo"}, translated)

		assert.Equal(t, 3, client.callCount)
	})
}

// TestAudioGenerationCacheHits verifies that audio generation API calls are properly cached.
func TestAudioGenerationCacheHits(t *testing.T) {
	t.Run("no cache hit on first audio generation", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		mockClient := new(mocks.MockOpenAIClient)
		logger := &mockLogger{}
		textService := NewTextService(fs, logger)
		service := NewAudioService(fs, mockClient, textService, logger)

		texts := []string{"Hello", "World"}
		outputDir := testPath("output")

		mockClient.On("GenerateSpeech", mock.Anything, "Hello").
			Return(newCacheTestReadCloser("audio1"), nil).Once()
		mockClient.On("GenerateSpeech", mock.Anything, "World").
			Return(newCacheTestReadCloser("audio2"), nil).Once()

		ctx := context.Background()
		paths, err := service.GenerateBatch(ctx, texts, outputDir)

		assert.NoError(t, err)
		assert.Len(t, paths, 2)
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
		outputDir := testPath("output")

		mockClient.On("GenerateSpeech", mock.Anything, "Hello").
			Return(newCacheTestReadCloser("audio1"), nil).Once()
		mockClient.On("GenerateSpeech", mock.Anything, "World").
			Return(newCacheTestReadCloser("audio2"), nil).Once()

		ctx := context.Background()
		paths1, err := service.GenerateBatch(ctx, texts, outputDir)
		require.NoError(t, err)

		paths2, err := service.GenerateBatch(ctx, texts, outputDir)
		assert.NoError(t, err)
		assert.Equal(t, paths1, paths2)

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
		modifiedTexts := []string{"Hello", "Universe"}
		outputDir := testPath("output")

		mockClient.On("GenerateSpeech", mock.Anything, "Hello").
			Return(newCacheTestReadCloser("audio1"), nil).Once()
		mockClient.On("GenerateSpeech", mock.Anything, "World").
			Return(newCacheTestReadCloser("audio2"), nil).Once()

		ctx := context.Background()
		_, err := service.GenerateBatch(ctx, initialTexts, outputDir)
		require.NoError(t, err)

		mockClient.On("GenerateSpeech", mock.Anything, "Universe").
			Return(newCacheTestReadCloser("audio3"), nil).Once()

		paths, err := service.GenerateBatch(ctx, modifiedTexts, outputDir)
		assert.NoError(t, err)
		assert.Len(t, paths, 2)

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
		outputPath := testPath("output", "audio.mp3")

		mockClient.On("GenerateSpeech", mock.Anything, text).
			Return(newCacheTestReadCloser("audio data"), nil).Once()

		ctx := context.Background()
		err := service.Generate(ctx, text, outputPath)
		require.NoError(t, err)

		err = service.Generate(ctx, text, outputPath)
		assert.NoError(t, err)

		mockClient.AssertNumberOfCalls(t, "GenerateSpeech", 1)
		mockClient.AssertExpectations(t)
	})
}

// TestFFmpegVideoCacheHits verifies that video segment output paths persist as expected.
func TestFFmpegVideoCacheHits(t *testing.T) {
	t.Run("video segments directory structure for caching", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		tempDir := testPath("test", "data", "out", ".temp")
		require.NoError(t, fs.MkdirAll(tempDir, 0755))

		segmentPaths := []string{
			testPath("test", "data", "out", ".temp", "video_0.mp4"),
			testPath("test", "data", "out", ".temp", "video_1.mp4"),
			testPath("test", "data", "out", ".temp", "video_2.mp4"),
		}

		for _, path := range segmentPaths {
			err := afero.WriteFile(fs, path, []byte("video data"), 0644)
			assert.NoError(t, err)
		}

		for i, path := range segmentPaths {
			exists, err := afero.Exists(fs, path)
			assert.NoError(t, err)
			assert.True(t, exists, "Video segment %d should exist at %s", i, path)
		}
	})
}
