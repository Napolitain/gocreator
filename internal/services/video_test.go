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
		args, err := service.buildSingleVideoArgs(videoRenderInput{
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
		require.NoError(t, err)

		require.Contains(t, args, "-filter_complex")
		assert.Contains(t, strings.Join(args, " "), "amix=inputs=2:duration=first:dropout_transition=0[a]")
		assert.Contains(t, args, "[a]")
		assert.Contains(t, args, "0:v:0")
		assert.Equal(t, "out.mp4", args[len(args)-1])
	})

	t.Run("video slides without embedded audio keep narration track only", func(t *testing.T) {
		args, err := service.buildSingleVideoArgs(videoRenderInput{
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
		require.NoError(t, err)

		assert.NotContains(t, strings.Join(args, " "), "amix=inputs=2")
		assert.Contains(t, args, "1:a:0")
	})

	t.Run("slide alignment loops or trims video to narration duration", func(t *testing.T) {
		args, err := service.buildSingleVideoArgs(videoRenderInput{
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
		require.NoError(t, err)

		assert.Equal(t, []string{"-y", "-stream_loop", "-1", "-i", "clip.mp4", "-i", "voice.mp3"}, args[:7])
		assert.Contains(t, strings.Join(args, " "), "-t 18.00")
	})

	t.Run("image slides keep still-image rendering flags", func(t *testing.T) {
		args, err := service.buildSingleVideoArgs(videoRenderInput{
			slidePath:    "slide.png",
			audioPath:    "voice.mp3",
			outputPath:   "out.mp4",
			targetWidth:  1920,
			targetHeight: 1080,
			inputWidth:   1024,
			inputHeight:  768,
		})
		require.NoError(t, err)

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

func TestVideoServiceBuildSingleVideoArgs_WithEffects(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewVideoService(fs, logger)

	t.Run("image slides add ken burns and overlay filters", func(t *testing.T) {
		args, err := service.buildSingleVideoArgs(videoRenderInput{
			slidePath:     "slide.png",
			audioPath:     "voice.mp3",
			outputPath:    "out.mp4",
			targetWidth:   1920,
			targetHeight:  1080,
			inputWidth:    1024,
			inputHeight:   768,
			audioDuration: 6,
			effects: []config.EffectConfig{
				{Type: "ken-burns", Config: config.EffectDetails{ZoomStart: 1.0, ZoomEnd: 1.2, Direction: "left"}},
				{Type: "text-overlay", Config: config.EffectDetails{Text: "Hello", Position: "bottom-right", FontSize: 24}},
			},
		})
		require.NoError(t, err)

		require.Contains(t, args, "-filter_complex")
		joined := strings.Join(args, " ")
		assert.Contains(t, joined, "zoompan=")
		assert.Contains(t, joined, "drawtext=text='Hello'")
		assert.NotContains(t, joined, " -vf ")
	})

	t.Run("video slides apply post-processing effects after scaling", func(t *testing.T) {
		args, err := service.buildSingleVideoArgs(videoRenderInput{
			slidePath:        "clip.mp4",
			audioPath:        "voice.mp3",
			outputPath:       "out.mp4",
			targetWidth:      1280,
			targetHeight:     720,
			inputWidth:       640,
			inputHeight:      480,
			videoDuration:    10,
			audioDuration:    10,
			isVideo:          true,
			hasEmbeddedAudio: true,
			effects: []config.EffectConfig{
				{Type: "color-grade", Config: config.EffectDetails{Brightness: 0.1, Contrast: 1.1}},
				{Type: "vignette", Config: config.EffectDetails{Intensity: 0.3}},
				{Type: "film-grain", Config: config.EffectDetails{Intensity: 0.2}},
			},
		})
		require.NoError(t, err)

		joined := strings.Join(args, " ")
		assert.Contains(t, joined, "scale=1280:720:force_original_aspect_ratio=decrease")
		assert.Contains(t, joined, "eq=brightness=0.10:contrast=1.10:saturation=1.00")
		assert.Contains(t, joined, "vignette=")
		assert.Contains(t, joined, "noise=alls=")
		assert.Contains(t, joined, "amix=inputs=2:duration=first:dropout_transition=0[a]")
	})

	t.Run("video slides reject ken burns", func(t *testing.T) {
		err := service.validateEffectsForSlide("clip.mp4", true, []config.EffectConfig{{Type: "ken-burns"}})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ken-burns")
	})
}

func TestVideoServiceResolveEffectsForSlide_DeterministicRandom(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewVideoService(fs, logger)

	effects := []config.EffectConfig{
		{Type: "ken-burns", Config: config.EffectDetails{Direction: "random"}},
	}

	first := service.resolveEffectsForSlide("slides/01.png", effects)
	second := service.resolveEffectsForSlide("slides/01.png", effects)
	third := service.resolveEffectsForSlide("slides/02.png", effects)

	require.Len(t, first, 1)
	require.Len(t, second, 1)
	require.Len(t, third, 1)
	assert.Equal(t, first[0].Config.Direction, second[0].Config.Direction)
	assert.NotEqual(t, "random", first[0].Config.Direction)
	assert.Contains(t, []string{"left", "right", "up", "down", "center"}, third[0].Config.Direction)
}
