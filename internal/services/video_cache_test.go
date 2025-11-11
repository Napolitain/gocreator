package services

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVideoService_computeSegmentHash(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewVideoService(fs, logger)

	// Create test files
	slidePath := "/test/slide.png"
	audioPath := "/test/audio.mp3"
	require.NoError(t, afero.WriteFile(fs, slidePath, []byte("slide data"), 0644))
	require.NoError(t, afero.WriteFile(fs, audioPath, []byte("audio data"), 0644))

	// Compute hash
	hash1, err := service.computeSegmentHash(slidePath, audioPath, 1920, 1080)
	require.NoError(t, err)
	assert.NotEmpty(t, hash1)

	// Same inputs should produce same hash
	hash2, err := service.computeSegmentHash(slidePath, audioPath, 1920, 1080)
	require.NoError(t, err)
	assert.Equal(t, hash1, hash2)

	// Different dimensions should produce different hash
	hash3, err := service.computeSegmentHash(slidePath, audioPath, 1280, 720)
	require.NoError(t, err)
	assert.NotEqual(t, hash1, hash3)

	// Different slide content should produce different hash
	require.NoError(t, afero.WriteFile(fs, slidePath, []byte("different slide"), 0644))
	hash4, err := service.computeSegmentHash(slidePath, audioPath, 1920, 1080)
	require.NoError(t, err)
	assert.NotEqual(t, hash1, hash4)
}

func TestVideoService_checkSegmentCache(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewVideoService(fs, logger)

	slidePath := "/test/slide.png"
	audioPath := "/test/audio.mp3"
	outputPath := "/test/output.mp4"

	// Create test files
	require.NoError(t, afero.WriteFile(fs, slidePath, []byte("slide data"), 0644))
	require.NoError(t, afero.WriteFile(fs, audioPath, []byte("audio data"), 0644))

	t.Run("cache miss when output doesn't exist", func(t *testing.T) {
		cached, err := service.checkSegmentCache(slidePath, audioPath, outputPath, 1920, 1080)
		require.NoError(t, err)
		assert.False(t, cached)
	})

	t.Run("cache miss when hash file doesn't exist", func(t *testing.T) {
		require.NoError(t, afero.WriteFile(fs, outputPath, []byte("video data"), 0644))
		cached, err := service.checkSegmentCache(slidePath, audioPath, outputPath, 1920, 1080)
		require.NoError(t, err)
		assert.False(t, cached)
	})

	t.Run("cache miss when hash doesn't match", func(t *testing.T) {
		require.NoError(t, afero.WriteFile(fs, outputPath+".hash", []byte("wrong hash"), 0644))
		cached, err := service.checkSegmentCache(slidePath, audioPath, outputPath, 1920, 1080)
		require.NoError(t, err)
		assert.False(t, cached)
	})

	t.Run("cache hit when hash matches", func(t *testing.T) {
		// Save correct hash
		require.NoError(t, service.saveSegmentHash(slidePath, audioPath, outputPath, 1920, 1080))
		cached, err := service.checkSegmentCache(slidePath, audioPath, outputPath, 1920, 1080)
		require.NoError(t, err)
		assert.True(t, cached)
	})

	t.Run("cache miss when input changes", func(t *testing.T) {
		// Modify slide
		require.NoError(t, afero.WriteFile(fs, slidePath, []byte("modified slide"), 0644))
		cached, err := service.checkSegmentCache(slidePath, audioPath, outputPath, 1920, 1080)
		require.NoError(t, err)
		assert.False(t, cached)
	})
}

func TestVideoService_saveSegmentHash(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewVideoService(fs, logger)

	slidePath := "/test/slide.png"
	audioPath := "/test/audio.mp3"
	outputPath := "/test/output.mp4"

	// Create test files
	require.NoError(t, afero.WriteFile(fs, slidePath, []byte("slide data"), 0644))
	require.NoError(t, afero.WriteFile(fs, audioPath, []byte("audio data"), 0644))

	// Save hash
	err := service.saveSegmentHash(slidePath, audioPath, outputPath, 1920, 1080)
	require.NoError(t, err)

	// Verify hash file was created
	exists, err := afero.Exists(fs, outputPath+".hash")
	require.NoError(t, err)
	assert.True(t, exists)

	// Verify hash content
	hashData, err := afero.ReadFile(fs, outputPath+".hash")
	require.NoError(t, err)
	assert.NotEmpty(t, hashData)
}

func TestVideoService_computeFinalVideoHash(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewVideoService(fs, logger)

	// Create test video files
	video1 := "/test/video1.mp4"
	video2 := "/test/video2.mp4"
	require.NoError(t, afero.WriteFile(fs, video1, []byte("video1 data"), 0644))
	require.NoError(t, afero.WriteFile(fs, video2, []byte("video2 data"), 0644))

	videoFiles := []string{video1, video2}

	t.Run("same inputs produce same hash", func(t *testing.T) {
		hash1, err := service.computeFinalVideoHash(videoFiles)
		require.NoError(t, err)
		assert.NotEmpty(t, hash1)

		hash2, err := service.computeFinalVideoHash(videoFiles)
		require.NoError(t, err)
		assert.Equal(t, hash1, hash2)
	})

	t.Run("different transition config produces different hash", func(t *testing.T) {
		service.SetTransition(TransitionConfig{Type: TransitionNone})
		hash1, err := service.computeFinalVideoHash(videoFiles)
		require.NoError(t, err)

		service.SetTransition(TransitionConfig{Type: TransitionFade, Duration: 0.5})
		hash2, err := service.computeFinalVideoHash(videoFiles)
		require.NoError(t, err)

		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("different video content produces different hash", func(t *testing.T) {
		hash1, err := service.computeFinalVideoHash(videoFiles)
		require.NoError(t, err)

		// Modify video content
		require.NoError(t, afero.WriteFile(fs, video1, []byte("modified video1"), 0644))
		hash2, err := service.computeFinalVideoHash(videoFiles)
		require.NoError(t, err)

		assert.NotEqual(t, hash1, hash2)
	})
}

func TestVideoService_checkFinalVideoCache(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewVideoService(fs, logger)

	video1 := "/test/video1.mp4"
	video2 := "/test/video2.mp4"
	outputPath := "/test/final.mp4"

	// Create test video files
	require.NoError(t, afero.WriteFile(fs, video1, []byte("video1 data"), 0644))
	require.NoError(t, afero.WriteFile(fs, video2, []byte("video2 data"), 0644))

	videoFiles := []string{video1, video2}

	t.Run("cache miss when output doesn't exist", func(t *testing.T) {
		cached, err := service.checkFinalVideoCache(videoFiles, outputPath)
		require.NoError(t, err)
		assert.False(t, cached)
	})

	t.Run("cache miss when hash file doesn't exist", func(t *testing.T) {
		require.NoError(t, afero.WriteFile(fs, outputPath, []byte("final video"), 0644))
		cached, err := service.checkFinalVideoCache(videoFiles, outputPath)
		require.NoError(t, err)
		assert.False(t, cached)
	})

	t.Run("cache miss when hash doesn't match", func(t *testing.T) {
		require.NoError(t, afero.WriteFile(fs, outputPath+".hash", []byte("wrong hash"), 0644))
		cached, err := service.checkFinalVideoCache(videoFiles, outputPath)
		require.NoError(t, err)
		assert.False(t, cached)
	})

	t.Run("cache hit when hash matches", func(t *testing.T) {
		// Save correct hash
		require.NoError(t, service.saveFinalVideoHash(videoFiles, outputPath))
		cached, err := service.checkFinalVideoCache(videoFiles, outputPath)
		require.NoError(t, err)
		assert.True(t, cached)
	})

	t.Run("cache miss when segment changes", func(t *testing.T) {
		// Modify segment
		require.NoError(t, afero.WriteFile(fs, video1, []byte("modified video1"), 0644))
		cached, err := service.checkFinalVideoCache(videoFiles, outputPath)
		require.NoError(t, err)
		assert.False(t, cached)
	})

	t.Run("cache miss when transition config changes", func(t *testing.T) {
		// Reset video content and save hash
		require.NoError(t, afero.WriteFile(fs, video1, []byte("video1 data"), 0644))
		service.SetTransition(TransitionConfig{Type: TransitionNone})
		require.NoError(t, service.saveFinalVideoHash(videoFiles, outputPath))

		// Cache hit with same config
		cached, err := service.checkFinalVideoCache(videoFiles, outputPath)
		require.NoError(t, err)
		assert.True(t, cached)

		// Change transition config - cache miss
		service.SetTransition(TransitionConfig{Type: TransitionFade, Duration: 0.5})
		cached, err = service.checkFinalVideoCache(videoFiles, outputPath)
		require.NoError(t, err)
		assert.False(t, cached)
	})
}

func TestVideoService_saveFinalVideoHash(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewVideoService(fs, logger)

	video1 := "/test/video1.mp4"
	video2 := "/test/video2.mp4"
	outputPath := "/test/final.mp4"

	// Create test video files
	require.NoError(t, afero.WriteFile(fs, video1, []byte("video1 data"), 0644))
	require.NoError(t, afero.WriteFile(fs, video2, []byte("video2 data"), 0644))

	videoFiles := []string{video1, video2}

	// Save hash
	err := service.saveFinalVideoHash(videoFiles, outputPath)
	require.NoError(t, err)

	// Verify hash file was created
	exists, err := afero.Exists(fs, outputPath+".hash")
	require.NoError(t, err)
	assert.True(t, exists)

	// Verify hash content
	hashData, err := afero.ReadFile(fs, outputPath+".hash")
	require.NoError(t, err)
	assert.NotEmpty(t, hashData)
}

func TestVideoService_concatenateVideosWithTransitions_GuardSingleVideo(t *testing.T) {
fs := afero.NewMemMapFs()
logger := &mockLogger{}
service := NewVideoService(fs, logger)
service.SetTransition(TransitionConfig{Type: TransitionFade, Duration: 0.5})

video1 := "/test/video1.mp4"
outputPath := "/test/final.mp4"

// Create test video file
require.NoError(t, afero.WriteFile(fs, video1, []byte("video1 data"), 0644))

videoFiles := []string{video1}

// Should return error when called with single video
err := service.concatenateVideosWithTransitions(videoFiles, outputPath)
require.Error(t, err)
assert.Contains(t, err.Error(), "requires at least 2 videos")
}
