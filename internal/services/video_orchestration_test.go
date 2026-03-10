package services

import (
	"context"
	"strings"
	"testing"

	"gocreator/internal/config"
	"gocreator/internal/mocks"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestVideoService_GenerateSingleVideo_UsesFastCacheWithoutCommands(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	executor := newFakeCommandExecutor()
	service := NewVideoServiceWithExecutor(fs, logger, executor)

	slidePath := testPath("test", "slide.png")
	audioPath := testPath("test", "audio.wav")
	outputPath := testPath("test", "out.mp4")

	require.NoError(t, afero.WriteFile(fs, slidePath, []byte("slide"), 0644))
	require.NoError(t, afero.WriteFile(fs, audioPath, []byte("audio"), 0644))
	require.NoError(t, afero.WriteFile(fs, outputPath, []byte("cached video"), 0644))
	require.NoError(t, service.saveSegmentHash(slidePath, audioPath, outputPath, 1920, 1080, config.MediaAlignmentVideo, nil))

	err := service.generateSingleVideo(context.Background(), slidePath, audioPath, outputPath, 1920, 1080, nil)
	require.NoError(t, err)
	assert.Empty(t, executor.Calls())
}

func TestVideoService_GenerateSingleVideo_WithStabilizeUsesInjectedExecutor(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	outputPath := testPath("test", "out.mp4")
	transformPath := outputPath + ".transforms.trf"
	executor := newFakeCommandExecutor(
		expectedCommand{
			Name:   "ffprobe",
			Result: newCommandResult("codec_type=video\nduration=3.5\n", ""),
		},
		expectedCommand{
			Name:   "ffmpeg",
			Result: newCommandResult("", "Stream #0:0: Video: h264, yuv420p, 640x360 [SAR 1:1 DAR 16:9]\n"),
		},
		expectedCommand{
			Name:   "ffprobe",
			Result: newCommandResult("3.5\n", ""),
		},
		expectedCommand{
			Name:   "ffprobe",
			Result: newCommandResult("2.0\n", ""),
		},
		expectedCommand{
			Name:   "ffprobe",
			Result: newCommandResult("video\naudio\n", ""),
		},
		expectedCommand{
			Name: "ffmpeg",
			Contains: []string{
				"vidstabdetect=shakiness=5:accuracy=15:result='/test/out.mp4.transforms.trf'",
				"-f null -",
			},
			Run: func(_ string, _ []string) {
				_ = afero.WriteFile(fs, transformPath, []byte("transforms"), 0644)
			},
		},
		expectedCommand{
			Name: "ffmpeg",
			Contains: []string{
				"-filter_complex",
				"vidstabtransform=input='/test/out.mp4.transforms.trf':smoothing=12",
				"drawtext=text='Stable'",
				"amix=inputs=2:duration=first:dropout_transition=0[a]",
				outputPath,
			},
			Run: func(_ string, _ []string) {
				_ = afero.WriteFile(fs, outputPath, []byte("rendered video"), 0644)
			},
		},
	)
	service := NewVideoServiceWithExecutor(fs, logger, executor)

	slidePath := testPath("test", "clip.mp4")
	audioPath := testPath("test", "audio.wav")
	require.NoError(t, afero.WriteFile(fs, slidePath, []byte("video"), 0644))
	require.NoError(t, afero.WriteFile(fs, audioPath, []byte("audio"), 0644))

	err := service.generateSingleVideo(context.Background(), slidePath, audioPath, outputPath, 1280, 720, []config.EffectConfig{
		{Type: "stabilize", Config: config.EffectDetails{Smoothing: 12}},
		{Type: "text-overlay", Config: config.EffectDetails{Text: "Stable", Position: "center"}},
	})
	require.NoError(t, err)
	executor.AssertDone(t)

	hashExists, err := afero.Exists(fs, outputPath+".hash")
	require.NoError(t, err)
	assert.True(t, hashExists)

	transformExists, err := afero.Exists(fs, transformPath)
	require.NoError(t, err)
	assert.False(t, transformExists)
}

func TestVideoCreatorCreate_PassesEffectsToConcreteVideoService(t *testing.T) {
	fs := afero.NewMemMapFs()
	mockText := new(mocks.MockTextProcessor)
	mockTranslation := new(mocks.MockTranslator)
	mockAudio := new(mocks.MockAudioGenerator)
	mockSlide := new(mocks.MockSlideLoader)
	logger := &mockLogger{}

	rootDir := testPath("test")
	slidesDir := testPath("test", "data", "slides")
	slidePath := testPath("test", "data", "slides", "1.png")
	audioPath := testPath("test", "data", "slides", "1.wav")
	outputPath := testPath("test", "data", "out", "output-en.mp4")

	require.NoError(t, fs.MkdirAll(slidesDir, 0755))
	require.NoError(t, afero.WriteFile(fs, slidePath, []byte("slide"), 0644))
	require.NoError(t, afero.WriteFile(fs, audioPath, []byte("audio"), 0644))

	executor := newFakeCommandExecutor(
		expectedCommand{
			Name:   "ffmpeg",
			Result: newCommandResult("", "Stream #0:0: Video: png, rgb24, 320x240\n"),
		},
		expectedCommand{
			Name:   "ffprobe",
			Result: newCommandResult("codec_type=video\nduration=0.0\n", ""),
		},
		expectedCommand{
			Name:   "ffmpeg",
			Result: newCommandResult("", "Stream #0:0: Video: png, rgb24, 320x240\n"),
		},
		expectedCommand{
			Name:   "ffprobe",
			Result: newCommandResult("1.2\n", ""),
		},
		expectedCommand{
			Name: "ffmpeg",
			Contains: []string{
				"-filter_complex",
				"drawtext=text='Caption'",
			},
			Run: func(_ string, args []string) {
				_ = afero.WriteFile(fs, args[len(args)-1], []byte("segment"), 0644)
			},
		},
		expectedCommand{
			Name: "ffmpeg",
			Contains: []string{
				"concat=n=1:v=1:a=1[outv][outa]",
				outputPath,
			},
			Run: func(_ string, args []string) {
				_ = afero.WriteFile(fs, args[len(args)-1], []byte("final"), 0644)
			},
		},
	)
	videoService := NewVideoServiceWithExecutor(fs, logger, executor)
	mockSlide.On("LoadSlides", mock.Anything, slidesDir).Return([]string{slidePath}, nil).Once()

	creator := NewVideoCreator(fs, mockText, mockTranslation, mockAudio, videoService, mockSlide, logger)
	err := creator.Create(context.Background(), VideoCreatorConfig{
		RootDir:     rootDir,
		InputLang:   "en",
		OutputLangs: []string{"en"},
		Effects: []config.EffectConfig{
			{Type: "text-overlay", Slides: []int{0}, Config: config.EffectDetails{Text: "Caption", Position: "bottom-right"}},
		},
	})
	require.NoError(t, err)
	executor.AssertDone(t)
	mockSlide.AssertExpectations(t)
	mockTranslation.AssertNotCalled(t, "TranslateBatch")
	mockAudio.AssertNotCalled(t, "Generate")

	exists, err := afero.Exists(fs, outputPath)
	require.NoError(t, err)
	assert.True(t, exists)

	var sawOverlay bool
	for _, call := range executor.Calls() {
		if call.Name == "ffmpeg" && strings.Contains(strings.Join(call.Args, " "), "drawtext=text='Caption'") {
			sawOverlay = true
			break
		}
	}
	assert.True(t, sawOverlay)
}
