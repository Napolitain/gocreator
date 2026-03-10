package services

import (
	"context"
	"path/filepath"
	"testing"

	"gocreator/internal/config"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostProcessServiceRun_GeneratesArtifactsAndExports(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	rootDir := testPath("test")
	outputDir := testPath("test", "data", "out")
	masterVideo := testPath("test", "data", "out", "output-en.master.mp4")
	slides := []string{
		testPath("test", "data", "slides", "1.png"),
		testPath("test", "data", "slides", "2.png"),
	}
	audioPaths := []string{
		testPath("test", "data", "cache", "en", "audio", "0.mp3"),
		testPath("test", "data", "cache", "en", "audio", "1.mp3"),
	}

	require.NoError(t, writeTestFile(fs, masterVideo, "master"))
	require.NoError(t, writeTestFile(fs, slides[0], "slide-1"))
	require.NoError(t, writeTestFile(fs, slides[1], "slide-2"))
	require.NoError(t, writeTestFile(fs, audioPaths[0], "audio-1"))
	require.NoError(t, writeTestFile(fs, audioPaths[1], "audio-2"))

	executor := newFakeCommandExecutor(
		expectedCommand{Name: "ffprobe", Result: newCommandResult("1.5\n", "")},
		expectedCommand{Name: "ffprobe", Result: newCommandResult("codec_type=video\nduration=0.000000\n", "")},
		expectedCommand{Name: "ffprobe", Result: newCommandResult("2.0\n", "")},
		expectedCommand{Name: "ffprobe", Result: newCommandResult("codec_type=video\nduration=0.000000\n", "")},
		expectedCommand{
			Name:     "ffmpeg",
			Contains: []string{"subtitles=", ".burned.mp4"},
			Run: func(_ string, args []string) {
				require.NoError(t, writeTestFile(fs, args[len(args)-1], "burned"))
			},
		},
		expectedCommand{
			Name:     "ffmpeg",
			Contains: []string{"-map_metadata", "-map_chapters"},
			Run: func(_ string, args []string) {
				metadataPath := args[4]
				data, err := afero.ReadFile(fs, metadataPath)
				require.NoError(t, err)
				assert.Contains(t, string(data), "title=Demo Title")
				assert.Contains(t, string(data), "[CHAPTER]")
				assert.Contains(t, string(data), "title=Introduction")
				require.NoError(t, writeTestFile(fs, args[len(args)-1], "primary"))
			},
		},
		expectedCommand{
			Name:     "ffmpeg",
			Contains: []string{"libvpx-vp9"},
			Run: func(_ string, args []string) {
				require.NoError(t, writeTestFile(fs, args[len(args)-1], "webm"))
			},
		},
		expectedCommand{
			Name:     "ffmpeg",
			Contains: []string{"-map_metadata", "-map_chapters"},
			Run: func(_ string, args []string) {
				require.NoError(t, writeTestFile(fs, args[len(args)-1], "webm-meta"))
			},
		},
		expectedCommand{
			Name:     "ffmpeg",
			Contains: []string{"-vframes", "1"},
			Run: func(_ string, args []string) {
				require.NoError(t, writeTestFile(fs, args[len(args)-1], "thumbnail"))
			},
		},
	)
	service := NewPostProcessServiceWithExecutor(fs, logger, executor)

	result, err := service.Run(context.Background(), PostProcessRequest{
		RootDir:        rootDir,
		OutputDir:      outputDir,
		BaseName:       "output-en",
		Lang:           "en",
		MasterVideo:    masterVideo,
		Slides:         slides,
		Texts:          []string{"Welcome to the demo", "Thanks for watching"},
		AudioPaths:     audioPaths,
		MediaAlignment: config.MediaAlignmentSlide,
		Output: config.OutputConfig{
			Format:  "mp4",
			Quality: "medium",
			Formats: []config.FormatConfig{
				{Type: "webm", Resolution: "1280x720", Quality: "high"},
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
			Title: "Demo Title",
			Thumbnail: config.ThumbnailConfig{
				Enabled:   true,
				Source:    "frame",
				FrameTime: 0.5,
			},
		},
		Chapters: config.ChaptersConfig{
			Enabled: true,
			Markers: []config.ChapterMarker{
				{Slide: 0, Title: "Introduction"},
				{Slide: 1, Title: "Outro"},
			},
		},
	})

	require.NoError(t, err)
	executor.AssertDone(t)

	assert.Equal(t, testPath("test", "data", "out", "output-en.mp4"), result.PrimaryOutputPath)
	assert.Equal(t, []string{
		testPath("test", "data", "out", "output-en.mp4"),
		testPath("test", "data", "out", "output-en-webm-1280x720-high.webm"),
	}, result.ExportedPaths)
	assert.Equal(t, []string{
		testPath("test", "data", "out", "output-en.srt"),
		testPath("test", "data", "out", "output-en.vtt"),
	}, result.SubtitlePaths)
	assert.Equal(t, testPath("test", "data", "out", "output-en-thumbnail.jpg"), result.ThumbnailPath)

	for _, path := range append(result.ExportedPaths, result.SubtitlePaths...) {
		exists, err := afero.Exists(fs, path)
		require.NoError(t, err)
		assert.True(t, exists, path)
	}
	exists, err := afero.Exists(fs, result.ThumbnailPath)
	require.NoError(t, err)
	assert.True(t, exists, result.ThumbnailPath)
}

func TestPostProcessServiceApplyAudioPostProcessing_UsesDuckingAndSoundEffects(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	rootDir := testPath("test")
	workingVideo := testPath("test", "data", "out", "output-en.master.mp4")
	tempDir := testPath("test", "data", "out", ".temp")
	musicPath := testPath("test", "assets", "music.mp3")
	sfxPath := testPath("test", "assets", "stinger.wav")

	require.NoError(t, writeTestFile(fs, workingVideo, "video"))
	require.NoError(t, writeTestFile(fs, musicPath, "music"))
	require.NoError(t, writeTestFile(fs, sfxPath, "sfx"))

	executor := newFakeCommandExecutor(
		expectedCommand{Name: "ffprobe", Result: newCommandResult("10.0\n", "")},
		expectedCommand{
			Name:     "ffmpeg",
			Contains: []string{"sidechaincompress", ".music.mp4"},
			Run: func(_ string, args []string) {
				require.NoError(t, writeTestFile(fs, args[len(args)-1], "ducked"))
			},
		},
		expectedCommand{
			Name:     "ffmpeg",
			Contains: []string{"adelay=3000|3000", ".sfx-0.mp4"},
			Run: func(_ string, args []string) {
				require.NoError(t, writeTestFile(fs, args[len(args)-1], "sfx"))
			},
		},
	)
	service := NewPostProcessServiceWithExecutor(fs, logger, executor)
	tempFiles := []string{}

	outputPath, err := service.applyAudioPostProcessing(context.Background(), PostProcessRequest{
		RootDir:  rootDir,
		BaseName: "output-en",
		Audio: config.AudioConfig{
			BackgroundMusic: config.BackgroundMusicConfig{
				Enabled: true,
				File:    testPath("assets", "music.mp3"),
				Volume:  0.2,
			},
			Ducking: config.DuckingConfig{
				Enabled: true,
			},
			SoundEffects: []config.SoundEffectConfig{
				{
					Slide: 1,
					File:  testPath("assets", "stinger.wav"),
				},
			},
		},
	}, tempDir, workingVideo, []float64{0, 2}, 1, &tempFiles)

	require.NoError(t, err)
	executor.AssertDone(t)
	assert.Equal(t, testPath("test", "data", "out", ".temp", "output-en.sfx-0.mp4"), outputPath)
	assert.Contains(t, tempFiles, testPath("test", "data", "out", ".temp", "output-en.music.mp4"))
	assert.Contains(t, tempFiles, testPath("test", "data", "out", ".temp", "output-en.sfx-0.mp4"))

	exists, err := afero.Exists(fs, outputPath)
	require.NoError(t, err)
	assert.True(t, exists)
}

func writeTestFile(fs afero.Fs, path string, content string) error {
	if err := fs.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return afero.WriteFile(fs, path, []byte(content), 0o644)
}
