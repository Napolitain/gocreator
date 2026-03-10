//go:build integration

package services

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gocreator/internal/config"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVideoService_GenerateSingleVideo_ImageEffectsSmoke(t *testing.T) {
	requireFFmpegTools(t)

	tempDir := t.TempDir()
	slidePath := filepath.Join(tempDir, "slide.png")
	audioPath := filepath.Join(tempDir, "audio.wav")
	outputPath := filepath.Join(tempDir, "image-effects.mp4")

	require.NoError(t, writeSolidPNG(slidePath, 320, 240, color.RGBA{R: 20, G: 80, B: 180, A: 255}))
	runExternalCommand(t, "ffmpeg", "-y", "-f", "lavfi", "-i", "anullsrc=r=44100:cl=mono", "-t", "1.5", "-c:a", "pcm_s16le", audioPath)

	service := NewVideoService(afero.NewOsFs(), &mockLogger{})
	err := service.generateSingleVideo(context.Background(), slidePath, audioPath, outputPath, 320, 240, []config.EffectConfig{
		{Type: "ken-burns", Config: config.EffectDetails{ZoomStart: 1.0, ZoomEnd: 1.15, Direction: "center"}},
	})
	require.NoError(t, err)

	info, err := os.Stat(outputPath)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))

	duration, err := service.getVideoDuration(outputPath)
	require.NoError(t, err)
	assert.Greater(t, duration, 1.0)
	assert.Less(t, duration, 2.5)
}

func TestVideoService_GenerateSingleVideo_StabilizeSmoke(t *testing.T) {
	requireFFmpegTools(t)
	if !hasFFmpegFilter(t, "vidstabdetect") || !hasFFmpegFilter(t, "vidstabtransform") {
		t.Skip("ffmpeg does not include vid.stab filters")
	}

	tempDir := t.TempDir()
	videoPath := filepath.Join(tempDir, "input.mp4")
	audioPath := filepath.Join(tempDir, "audio.wav")
	outputPath := filepath.Join(tempDir, "stabilized.mp4")

	runExternalCommand(t, "ffmpeg", "-y", "-f", "lavfi", "-i", "testsrc=size=320x240:rate=30", "-t", "1.5", "-pix_fmt", "yuv420p", videoPath)
	runExternalCommand(t, "ffmpeg", "-y", "-f", "lavfi", "-i", "anullsrc=r=44100:cl=mono", "-t", "1.5", "-c:a", "pcm_s16le", audioPath)

	service := NewVideoService(afero.NewOsFs(), &mockLogger{})
	err := service.generateSingleVideo(context.Background(), videoPath, audioPath, outputPath, 320, 240, []config.EffectConfig{
		{Type: "stabilize", Config: config.EffectDetails{Smoothing: 8}},
	})
	require.NoError(t, err)

	info, err := os.Stat(outputPath)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))

	duration, err := service.getVideoDuration(outputPath)
	require.NoError(t, err)
	assert.Greater(t, duration, 1.0)

	_, err = os.Stat(outputPath + ".transforms.trf")
	assert.True(t, os.IsNotExist(err))
}

func requireFFmpegTools(t *testing.T) {
	t.Helper()
	for _, tool := range []string{"ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s is not installed", tool)
		}
	}
}

func hasFFmpegFilter(t *testing.T, filterName string) bool {
	t.Helper()
	cmd := exec.Command("ffmpeg", "-hide_banner", "-filters")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("failed to inspect ffmpeg filters: %v", err)
	}
	return strings.Contains(string(output), filterName)
}

func runExternalCommand(t *testing.T, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "%s failed: %s", formatCommand(name, args...), string(output))
}

func writeSolidPNG(path string, width, height int, fill color.Color) error {
	imageFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = imageFile.Close() }()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, fill)
		}
	}

	return png.Encode(imageFile, img)
}
