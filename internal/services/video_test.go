package services

import (
	"strings"
	"testing"

	"gocreator/internal/config"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewVideoService(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}

	service := NewVideoService(fs, logger)

	assert.NotNil(t, service)
	assert.Equal(t, fs, service.fs)
	assert.Equal(t, logger, service.logger)
	assert.Equal(t, config.MediaAlignmentVideo, service.mediaAlignment)
}

// Note: isVideoFile, getMediaDimensions, concatenateVideos, and generateSingleVideo
// all depend on ffmpeg/ffprobe being installed and available.
// These functions are tested in integration tests but cannot be easily unit tested
// without mocking the exec.Command functionality or having ffmpeg installed.

func TestVideoServiceBuildSingleVideoArgs(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewVideoService(fs, logger)

	t.Run("video slides mix embedded audio with narration", func(t *testing.T) {
		args := service.buildSingleVideoArgs(videoRenderInput{
			slidePath:        "clip.mp4",
			audioPath:        "voice.mp3",
			outputPath:       "out.mp4",
			targetWidth:      1280,
			targetHeight:     720,
			inputWidth:       1280,
			inputHeight:      720,
			videoDuration:    12.5,
			isVideo:          true,
			hasEmbeddedAudio: true,
		})

		require.Contains(t, args, "-filter_complex")
		assert.Contains(t, strings.Join(args, " "), "amix=inputs=2:duration=first:dropout_transition=0[a]")
		assert.Contains(t, args, "[a]")
		assert.Contains(t, args, "0:v:0")
		assert.Equal(t, "out.mp4", args[len(args)-1])
	})

	t.Run("video slides without embedded audio keep narration track only", func(t *testing.T) {
		args := service.buildSingleVideoArgs(videoRenderInput{
			slidePath:     "clip.mp4",
			audioPath:     "voice.mp3",
			outputPath:    "out.mp4",
			targetWidth:   1280,
			targetHeight:  720,
			inputWidth:    1280,
			inputHeight:   720,
			videoDuration: 8,
			isVideo:       true,
		})

		assert.NotContains(t, strings.Join(args, " "), "amix=inputs=2")
		assert.Contains(t, args, "1:a:0")
	})

	t.Run("slide alignment loops or trims video to narration duration", func(t *testing.T) {
		args := service.buildSingleVideoArgs(videoRenderInput{
			slidePath:     "clip.mp4",
			audioPath:     "voice.mp3",
			outputPath:    "out.mp4",
			targetWidth:   1280,
			targetHeight:  720,
			inputWidth:    1280,
			inputHeight:   720,
			videoDuration: 12.5,
			audioDuration: 18,
			isVideo:       true,
			alignToSlide:  true,
		})

		assert.Equal(t, []string{"-y", "-stream_loop", "-1", "-i", "clip.mp4", "-i", "voice.mp3"}, args[:7])
		assert.Contains(t, strings.Join(args, " "), "-t 18.00")
	})

	t.Run("image slides keep still-image rendering flags", func(t *testing.T) {
		args := service.buildSingleVideoArgs(videoRenderInput{
			slidePath:    "slide.png",
			audioPath:    "voice.mp3",
			outputPath:   "out.mp4",
			targetWidth:  1920,
			targetHeight: 1080,
			inputWidth:   1024,
			inputHeight:  768,
		})

		assert.Equal(t, []string{"-y", "-loop", "1", "-i", "slide.png", "-i", "voice.mp3"}, args[:7])
		assert.Contains(t, args, "-vf")
		assert.Contains(t, strings.Join(args, " "), "scale=1920:1080:force_original_aspect_ratio=decrease")
		assert.Contains(t, args, "-shortest")
	})
}

func TestVideoServiceSetMediaAlignment(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewVideoService(fs, logger)

	require.NoError(t, service.SetMediaAlignment(config.MediaAlignmentSlide))
	assert.Equal(t, config.MediaAlignmentSlide, service.mediaAlignment)

	err := service.SetMediaAlignment("invalid")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported media alignment")
}
