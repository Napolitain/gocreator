//go:build integration

package services

import (
	"context"
	"image/color"
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

func TestPostProcessService_Run_Smoke(t *testing.T) {
	requireFFmpegTools(t)

	tempDir := t.TempDir()
	outputDir := filepath.Join(tempDir, "out")
	masterVideo := filepath.Join(tempDir, "master.mp4")
	slide1 := filepath.Join(tempDir, "slide-1.png")
	slide2 := filepath.Join(tempDir, "slide-2.png")
	audio1 := filepath.Join(tempDir, "audio-1.wav")
	audio2 := filepath.Join(tempDir, "audio-2.wav")

	require.NoError(t, writeSolidPNG(slide1, 320, 240, imageColor(30, 90, 180)))
	require.NoError(t, writeSolidPNG(slide2, 320, 240, imageColor(180, 90, 30)))
	runExternalCommand(t, "ffmpeg", "-y", "-f", "lavfi", "-i", "anullsrc=r=44100:cl=mono", "-t", "1.0", "-c:a", "pcm_s16le", audio1)
	runExternalCommand(t, "ffmpeg", "-y", "-f", "lavfi", "-i", "anullsrc=r=44100:cl=mono", "-t", "1.2", "-c:a", "pcm_s16le", audio2)
	runExternalCommand(t, "ffmpeg",
		"-y",
		"-f", "lavfi", "-i", "color=c=navy:s=320x240:d=2.2",
		"-f", "lavfi", "-i", "anullsrc=r=44100:cl=stereo",
		"-shortest",
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-c:a", "aac",
		masterVideo,
	)

	service := NewPostProcessService(afero.NewOsFs(), &mockLogger{})
	result, err := service.Run(context.Background(), PostProcessRequest{
		RootDir:        tempDir,
		OutputDir:      outputDir,
		BaseName:       "smoke",
		Lang:           "en",
		MasterVideo:    masterVideo,
		Slides:         []string{slide1, slide2},
		Texts:          []string{"Smoke subtitle one", "Smoke subtitle two"},
		AudioPaths:     []string{audio1, audio2},
		MediaAlignment: config.MediaAlignmentSlide,
		Output: config.OutputConfig{
			Format:  "mp4",
			Quality: "medium",
			Formats: []config.FormatConfig{
				{Type: "webm", Resolution: "320x240"},
			},
		},
		Subtitles: config.SubtitlesConfig{
			Enabled:   true,
			BurnIn:    true,
			Languages: "all",
			Style:     config.DefaultSubtitlesConfig().Style,
			Timing:    config.DefaultSubtitlesConfig().Timing,
		},
		Metadata: config.MetadataConfig{
			Title: "Smoke Title",
			Thumbnail: config.ThumbnailConfig{
				Enabled:   true,
				Source:    "frame",
				FrameTime: 0.5,
			},
		},
		Chapters: config.ChaptersConfig{
			Enabled: true,
			Markers: []config.ChapterMarker{
				{Slide: 0, Title: "Intro"},
				{Slide: 1, Title: "Outro"},
			},
		},
	})
	require.NoError(t, err)

	for _, path := range append(append([]string{}, result.ExportedPaths...), result.SubtitlePaths...) {
		info, statErr := os.Stat(path)
		require.NoError(t, statErr)
		assert.Greater(t, info.Size(), int64(0))
	}
	info, statErr := os.Stat(result.ThumbnailPath)
	require.NoError(t, statErr)
	assert.Greater(t, info.Size(), int64(0))

	srtData, err := os.ReadFile(filepath.Join(outputDir, "smoke.srt"))
	require.NoError(t, err)
	assert.Contains(t, string(srtData), "Smoke subtitle one")

	titleOut, err := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format_tags=title",
		"-of", "default=noprint_wrappers=1:nokey=1",
		result.PrimaryOutputPath,
	).CombinedOutput()
	require.NoError(t, err)
	assert.Contains(t, strings.TrimSpace(string(titleOut)), "Smoke Title")
}

func imageColor(r, g, b uint8) color.RGBA {
	return color.RGBA{R: r, G: g, B: b, A: 255}
}
