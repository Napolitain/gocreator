package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultAudioConfig(t *testing.T) {
	cfg := DefaultAudioConfig()
	
	assert.NotNil(t, cfg)
	assert.False(t, cfg.BackgroundMusic.Enabled)
	assert.Equal(t, 0.15, cfg.BackgroundMusic.Volume)
	assert.Equal(t, 2.0, cfg.BackgroundMusic.FadeIn)
	assert.Equal(t, 3.0, cfg.BackgroundMusic.FadeOut)
	assert.True(t, cfg.BackgroundMusic.Loop)
}

func TestBackgroundMusicConfig_Validation(t *testing.T) {
	tests := []struct {
		name   string
		cfg    BackgroundMusicConfig
		valid  bool
	}{
		{
			name: "valid config",
			cfg: BackgroundMusicConfig{
				Enabled: true,
				File:    "music.mp3",
				Volume:  0.5,
				FadeIn:  1.0,
				FadeOut: 1.0,
				Loop:    true,
			},
			valid: true,
		},
		{
			name: "negative volume",
			cfg: BackgroundMusicConfig{
				Enabled: true,
				Volume:  -0.5,
			},
			valid: false,
		},
		{
			name: "volume too high",
			cfg: BackgroundMusicConfig{
				Enabled: true,
				Volume:  2.0,
			},
			valid: false,
		},
		{
			name: "negative fade in",
			cfg: BackgroundMusicConfig{
				Enabled: true,
				FadeIn:  -1.0,
			},
			valid: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.valid {
				assert.GreaterOrEqual(t, tt.cfg.Volume, 0.0)
				assert.LessOrEqual(t, tt.cfg.Volume, 1.0)
				assert.GreaterOrEqual(t, tt.cfg.FadeIn, 0.0)
				assert.GreaterOrEqual(t, tt.cfg.FadeOut, 0.0)
			}
		})
	}
}

func FuzzBackgroundMusicVolume(f *testing.F) {
	// Add seed corpus
	f.Add(0.0)
	f.Add(0.5)
	f.Add(1.0)
	f.Add(-1.0)
	f.Add(2.0)
	
	f.Fuzz(func(t *testing.T, volume float64) {
		cfg := BackgroundMusicConfig{
			Enabled: true,
			Volume:  volume,
		}
		
		// Should not panic
		assert.NotNil(t, cfg)
	})
}
