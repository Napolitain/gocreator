package services

import (
	"context"
	"testing"

	"gocreator/internal/config"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVideoService_BuildVideoEffectsGraph_ImageChain(t *testing.T) {
	service := NewVideoService(afero.NewMemMapFs(), &mockLogger{})

	filters, videoMap, err := service.buildVideoEffectsGraph(videoRenderInput{
		slidePath:     "slide.png",
		audioPath:     "voice.wav",
		targetWidth:   1280,
		targetHeight:  720,
		inputWidth:    1024,
		inputHeight:   768,
		audioDuration: 4,
		effects: []config.EffectConfig{
			{Type: "ken-burns", Config: config.EffectDetails{ZoomStart: 1.0, ZoomEnd: 1.2, Direction: "right"}},
			{Type: "text-overlay", Config: config.EffectDetails{Text: "Hello", Position: "center"}},
			{Type: "color-grade", Config: config.EffectDetails{Brightness: 0.1, Contrast: 1.1}},
		},
	})
	require.NoError(t, err)
	require.Len(t, filters, 2)
	assert.Contains(t, filters[0], "[0:v]zoompan=")
	assert.Contains(t, filters[0], "s=1280x720")
	assert.Contains(t, filters[1], "[vfx0]drawtext=text='Hello'")
	assert.Contains(t, filters[1], "eq=brightness=0.10:contrast=1.10:saturation=1.00")
	assert.Equal(t, "[vfx1]", videoMap)
}

func TestVideoService_BuildVideoEffectsGraph_BlurBackground(t *testing.T) {
	service := NewVideoService(afero.NewMemMapFs(), &mockLogger{})

	filters, videoMap, err := service.buildVideoEffectsGraph(videoRenderInput{
		slidePath:    "portrait.png",
		audioPath:    "voice.wav",
		targetWidth:  1920,
		targetHeight: 1080,
		inputWidth:   1080,
		inputHeight:  1920,
		effects: []config.EffectConfig{
			{Type: "blur-background", Config: config.EffectDetails{BlurRadius: 32}},
			{Type: "film-grain", Config: config.EffectDetails{Intensity: 0.2}},
		},
	})
	require.NoError(t, err)
	require.Len(t, filters, 4)
	assert.Contains(t, filters[0], "[0:v]scale=1920:1080:force_original_aspect_ratio=increase,crop=1920:1080,boxblur=32[bg0]")
	assert.Contains(t, filters[1], "[0:v]scale=1920:1080:force_original_aspect_ratio=decrease,setsar=1[fg1]")
	assert.Contains(t, filters[2], "[bg0][fg1]overlay=(W-w)/2:(H-h)/2[vfx2]")
	assert.Contains(t, filters[3], "[vfx2]noise=alls=10:allf=t+u[vfx3]")
	assert.Equal(t, "[vfx3]", videoMap)
}

func TestVideoService_ValidateEffectsForSlide(t *testing.T) {
	service := NewVideoService(afero.NewMemMapFs(), &mockLogger{})

	tests := []struct {
		name      string
		slidePath string
		isVideo   bool
		effects   []config.EffectConfig
		wantError string
	}{
		{
			name:      "ken burns rejected on video",
			slidePath: "clip.mp4",
			isVideo:   true,
			effects:   []config.EffectConfig{{Type: "ken-burns"}},
			wantError: "ken-burns",
		},
		{
			name:      "stabilize rejected on image",
			slidePath: "slide.png",
			isVideo:   false,
			effects:   []config.EffectConfig{{Type: "stabilize"}},
			wantError: "stabilize",
		},
		{
			name:      "geometry effects cannot be combined",
			slidePath: "slide.png",
			isVideo:   false,
			effects:   []config.EffectConfig{{Type: "ken-burns"}, {Type: "blur-background"}},
			wantError: "cannot combine",
		},
		{
			name:      "duplicate stabilize rejected",
			slidePath: "clip.mp4",
			isVideo:   true,
			effects:   []config.EffectConfig{{Type: "stabilize"}, {Type: "stabilize"}},
			wantError: "multiple stabilize",
		},
		{
			name:      "unknown effect rejected",
			slidePath: "slide.png",
			isVideo:   false,
			effects:   []config.EffectConfig{{Type: "warp-speed"}},
			wantError: "unsupported effect type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateEffectsForSlide(tt.slidePath, tt.isVideo, tt.effects)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantError)
		})
	}

	require.NoError(t, service.validateEffectsForSlide("clip.mp4", true, []config.EffectConfig{
		{Type: "stabilize", Config: config.EffectDetails{Smoothing: 12}},
		{Type: "color-grade", Config: config.EffectDetails{Contrast: 1.1}},
		{Type: "text-overlay", Config: config.EffectDetails{Text: "Stable"}},
	}))
}

func TestVideoService_ResolveEffectsForSlides(t *testing.T) {
	service := NewVideoService(afero.NewMemMapFs(), &mockLogger{})
	service.SetEffects([]config.EffectConfig{
		{Type: "vignette", Slides: nil},
		{Type: "text-overlay", Slides: []int{1, 3}, Config: config.EffectDetails{Text: "Callout"}},
		{Type: "film-grain", Slides: "1-2"},
	})

	resolved := service.resolveEffectsForSlides([]string{"a.png", "b.png", "c.png", "d.png"})

	require.Len(t, resolved[0], 1)
	require.Len(t, resolved[1], 3)
	require.Len(t, resolved[2], 2)
	require.Len(t, resolved[3], 2)
	assert.Equal(t, "vignette", resolved[0][0].Type)
	assert.Equal(t, "text-overlay", resolved[1][1].Type)
	assert.Equal(t, "film-grain", resolved[1][2].Type)
}

func TestSerializeEffectsForCache_IsDeterministic(t *testing.T) {
	effects := []config.EffectConfig{
		{Type: "vignette", Config: config.EffectDetails{Intensity: 0.3}},
		{Type: "text-overlay", Config: config.EffectDetails{Text: "Hello"}},
	}

	first, err := serializeEffectsForCache(effects)
	require.NoError(t, err)
	second, err := serializeEffectsForCache(effects)
	require.NoError(t, err)

	assert.Equal(t, first, second)
	assert.Contains(t, first, `"type":"vignette"`)
	assert.Contains(t, first, `"type":"text-overlay"`)
}

func TestEscapeFFmpegFilterPath(t *testing.T) {
	assert.Equal(t, `C\:/work dir/clip\'s.trf`, escapeFFmpegFilterPath(`C:\work dir\clip's.trf`))
}

func TestVideoService_RunStabilizationDetect_UsesInjectedExecutor(t *testing.T) {
	fs := afero.NewMemMapFs()
	executor := newFakeCommandExecutor(expectedCommand{
		Name: "ffmpeg",
		Contains: []string{
			"-vf",
			`vidstabdetect=shakiness=5:accuracy=15:result='C\:/work dir/clip\'s.trf'`,
			"-f null -",
		},
	})
	service := NewVideoServiceWithExecutor(fs, &mockLogger{}, executor)

	err := service.runStabilizationDetect(context.Background(), "clip.mp4", config.EffectConfig{
		Type:   "stabilize",
		Config: config.EffectDetails{Smoothing: 12},
	}, `C:\work dir\clip's.trf`)
	require.NoError(t, err)
	executor.AssertDone(t)
}
