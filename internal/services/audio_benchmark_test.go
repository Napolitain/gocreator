package services

import (
	"context"
	"io"
	"strconv"
	"strings"
	"testing"

	"gocreator/internal/mocks"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/mock"
)

// benchmarkReadCloser implements io.ReadCloser for benchmark testing
type benchmarkReadCloser struct {
	*strings.Reader
}

func (m *benchmarkReadCloser) Close() error {
	return nil
}

func newBenchmarkReadCloser(data string) io.ReadCloser {
	return &benchmarkReadCloser{Reader: strings.NewReader(data)}
}

// Benchmark audio generation without cache
func BenchmarkAudioService_Generate_NoCache(b *testing.B) {
	fs := afero.NewMemMapFs()
	mockClient := new(mocks.MockOpenAIClient)
	logger := &mockLogger{}
	textService := NewTextService(fs, logger)
	service := NewAudioService(fs, mockClient, textService, logger)

	text := "This is a test text for audio generation benchmark"
	ctx := context.Background()

	// Mock API response
	mockClient.On("GenerateSpeech", mock.Anything, text).
		Return(newBenchmarkReadCloser("audio data"), nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		outputPath := "/output/audio_" + strconv.Itoa(i) + ".mp3"
		_ = service.Generate(ctx, text, outputPath)
	}
}

// Benchmark audio generation with cache hit
func BenchmarkAudioService_Generate_WithCache(b *testing.B) {
	fs := afero.NewMemMapFs()
	mockClient := new(mocks.MockOpenAIClient)
	logger := &mockLogger{}
	textService := NewTextService(fs, logger)
	service := NewAudioService(fs, mockClient, textService, logger)

	text := "This is a test text for audio generation benchmark"
	outputPath := "/output/audio.mp3"
	ctx := context.Background()

	// Generate once to populate cache
	mockClient.On("GenerateSpeech", mock.Anything, text).
		Return(newBenchmarkReadCloser("audio data"), nil).Once()
	_ = service.Generate(ctx, text, outputPath)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.Generate(ctx, text, outputPath)
	}
}

// Benchmark batch audio generation without cache
func BenchmarkAudioService_GenerateBatch_NoCache(b *testing.B) {
	fs := afero.NewMemMapFs()
	mockClient := new(mocks.MockOpenAIClient)
	logger := &mockLogger{}
	textService := NewTextService(fs, logger)
	service := NewAudioService(fs, mockClient, textService, logger)

	texts := []string{
		"First text for batch generation",
		"Second text for batch generation",
		"Third text for batch generation",
		"Fourth text for batch generation",
		"Fifth text for batch generation",
	}
	ctx := context.Background()

	// Mock API responses for all texts
	for _, text := range texts {
		mockClient.On("GenerateSpeech", mock.Anything, text).
			Return(newBenchmarkReadCloser("audio data"), nil)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		outputDir := "/output/batch_" + strconv.Itoa(i)
		_, _ = service.GenerateBatch(ctx, texts, outputDir)
	}
}

// Benchmark batch audio generation with cache hit
func BenchmarkAudioService_GenerateBatch_WithCache(b *testing.B) {
	fs := afero.NewMemMapFs()
	mockClient := new(mocks.MockOpenAIClient)
	logger := &mockLogger{}
	textService := NewTextService(fs, logger)
	service := NewAudioService(fs, mockClient, textService, logger)

	texts := []string{
		"First text for batch generation",
		"Second text for batch generation",
		"Third text for batch generation",
		"Fourth text for batch generation",
		"Fifth text for batch generation",
	}
	outputDir := "/output/batch"
	ctx := context.Background()

	// Mock API responses for initial generation
	for _, text := range texts {
		mockClient.On("GenerateSpeech", mock.Anything, text).
			Return(newBenchmarkReadCloser("audio data"), nil).Once()
	}

	// Generate once to populate cache
	_, _ = service.GenerateBatch(ctx, texts, outputDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GenerateBatch(ctx, texts, outputDir)
	}
}
