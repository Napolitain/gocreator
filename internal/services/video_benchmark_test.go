package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/spf13/afero"
)

// Note: These benchmarks test the video service logic but cannot actually
// run FFmpeg in the benchmark environment. They measure the overhead of
// the service methods and parallel coordination.

// Benchmark video generation from slides (tests parallel coordination)
func BenchmarkVideoService_GenerateFromSlides_3Slides(b *testing.B) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewVideoService(fs, logger)

	// Create test slides (text files as placeholders)
	slides := make([]string, 3)
	audioPaths := make([]string, 3)
	for i := 0; i < 3; i++ {
		slidePath := fmt.Sprintf("/slides/slide_%d.txt", i)
		audioPath := fmt.Sprintf("/audio/audio_%d.mp3", i)
		_ = fs.MkdirAll("/slides", 0755)
		_ = fs.MkdirAll("/audio", 0755)
		_ = afero.WriteFile(fs, slidePath, []byte(fmt.Sprintf("Slide %d", i)), 0644)
		_ = afero.WriteFile(fs, audioPath, []byte("audio data"), 0644)
		slides[i] = slidePath
		audioPaths[i] = audioPath
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		outputPath := fmt.Sprintf("/output/video_%d.mp4", i)
		// This will fail due to missing FFmpeg, but measures the service overhead
		_ = service.GenerateFromSlides(ctx, slides, audioPaths, outputPath)
	}
}

// Benchmark video generation with more slides
func BenchmarkVideoService_GenerateFromSlides_10Slides(b *testing.B) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewVideoService(fs, logger)

	// Create test slides
	slides := make([]string, 10)
	audioPaths := make([]string, 10)
	for i := 0; i < 10; i++ {
		slidePath := fmt.Sprintf("/slides/slide_%d.txt", i)
		audioPath := fmt.Sprintf("/audio/audio_%d.mp3", i)
		_ = fs.MkdirAll("/slides", 0755)
		_ = fs.MkdirAll("/audio", 0755)
		_ = afero.WriteFile(fs, slidePath, []byte(fmt.Sprintf("Slide %d", i)), 0644)
		_ = afero.WriteFile(fs, audioPath, []byte("audio data"), 0644)
		slides[i] = slidePath
		audioPaths[i] = audioPath
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		outputPath := fmt.Sprintf("/output/video_%d.mp4", i)
		_ = service.GenerateFromSlides(ctx, slides, audioPaths, outputPath)
	}
}
