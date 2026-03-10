package services

import (
	"context"
	"testing"

	"gocreator/internal/config"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEffectService_BuildKenBurnsFilterForOutput(t *testing.T) {
	service := NewEffectService(afero.NewMemMapFs(), &mockLogger{})

	filter := service.BuildKenBurnsFilterForOutput(config.EffectConfig{
		Type: "ken-burns",
		Config: config.EffectDetails{
			ZoomStart: 1.0,
			ZoomEnd:   1.2,
			Direction: "left",
		},
	}, 5, 1280, 720)

	assert.Equal(t, "zoompan=z='if(lte(zoom,1.20),zoom+0.00133,1.20)':d=150:x='iw/2-(iw/zoom/2)-t*10':y='ih/2-(ih/zoom/2)':s=1280x720:fps=30", filter)
}

func TestEffectService_BuildKenBurnsFilterForOutput_UsesDefaults(t *testing.T) {
	service := NewEffectService(afero.NewMemMapFs(), &mockLogger{})

	filter := service.BuildKenBurnsFilterForOutput(config.EffectConfig{Type: "ken-burns"}, 0, 1920, 1080)

	assert.Contains(t, filter, "if(lte(zoom,1.30),zoom+0.01000,1.30)")
	assert.Contains(t, filter, "d=30")
	assert.Contains(t, filter, "x='iw/2-(iw/zoom/2)'")
	assert.Contains(t, filter, "y='ih/2-(ih/zoom/2)'")
	assert.Contains(t, filter, "s=1920x1080")
}

func TestEffectService_BuildColorGradeFilter(t *testing.T) {
	service := NewEffectService(afero.NewMemMapFs(), &mockLogger{})

	filter := service.BuildColorGradeFilter(config.EffectConfig{
		Type: "color-grade",
		Config: config.EffectDetails{
			Brightness: 0.1,
			Contrast:   1.1,
			Saturation: 1.2,
			Hue:        15,
			Gamma:      0.9,
		},
	})

	assert.Equal(t, "eq=brightness=0.10:contrast=1.10:saturation=1.20,hue=h=15,eq=gamma=0.90", filter)
	assert.Empty(t, service.BuildColorGradeFilter(config.EffectConfig{Type: "color-grade"}))
}

func TestEffectService_BuildVignetteFilter(t *testing.T) {
	service := NewEffectService(afero.NewMemMapFs(), &mockLogger{})

	assert.Equal(t, "vignette=angle=PI/13.3", service.BuildVignetteFilter(config.EffectConfig{Type: "vignette"}))
	assert.Equal(t, "vignette=angle=PI/8.0", service.BuildVignetteFilter(config.EffectConfig{
		Type:   "vignette",
		Config: config.EffectDetails{Intensity: 0.5},
	}))
}

func TestEffectService_BuildFilmGrainFilter(t *testing.T) {
	service := NewEffectService(afero.NewMemMapFs(), &mockLogger{})

	assert.Equal(t, "noise=alls=15:allf=t+u", service.BuildFilmGrainFilter(config.EffectConfig{Type: "film-grain"}))
	assert.Equal(t, "noise=alls=10:allf=t+u", service.BuildFilmGrainFilter(config.EffectConfig{
		Type:   "film-grain",
		Config: config.EffectDetails{Intensity: 0.2},
	}))
}

func TestEffectService_BuildBlurBackgroundFilter(t *testing.T) {
	service := NewEffectService(afero.NewMemMapFs(), &mockLogger{})

	assert.Equal(t,
		"[0:v]scale=1920:1080,boxblur=20[bg];[0:v]scale=1920:-1[fg];[bg][fg]overlay=(W-w)/2:(H-h)/2",
		service.BuildBlurBackgroundFilter(config.EffectConfig{Type: "blur-background"}, 1920, 1080),
	)
}

func TestEffectService_ApplyKenBurns_UsesInjectedExecutor(t *testing.T) {
	fs := afero.NewMemMapFs()
	executor := newFakeCommandExecutor(expectedCommand{
		Name: "ffmpeg",
		Contains: []string{
			"-loop 1",
			"-vf",
			"zoompan=",
			"-t 2.50",
			"out.mp4",
		},
	})

	service := NewEffectServiceWithExecutor(fs, &mockLogger{}, executor)

	err := service.ApplyKenBurns(context.Background(), "slide.png", "out.mp4", 2.5, config.EffectConfig{
		Type:   "ken-burns",
		Config: config.EffectDetails{Direction: "center"},
	})
	require.NoError(t, err)
	executor.AssertDone(t)
}
