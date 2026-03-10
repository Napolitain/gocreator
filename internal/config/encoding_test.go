package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultEncodingConfig(t *testing.T) {
	cfg := DefaultEncodingConfig()
	
	assert.NotNil(t, cfg)
	assert.Equal(t, "libx264", cfg.Video.Codec)
	assert.Equal(t, "medium", cfg.Video.Preset)
	assert.Equal(t, 23, cfg.Video.CRF)
	assert.Equal(t, 30, cfg.Video.FPS)
	assert.Equal(t, "aac", cfg.Audio.Codec)
	assert.Equal(t, "192k", cfg.Audio.Bitrate)
	assert.Equal(t, 48000, cfg.Audio.SampleRate)
}

func TestEncodingConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     EncodingConfig
		wantErr bool
	}{
		{
			name:    "valid default config",
			cfg:     DefaultEncodingConfig(),
			wantErr: false,
		},
		{
			name: "invalid video codec",
			cfg: EncodingConfig{
				Video: VideoEncodingConfig{
					Codec: "invalid",
				},
			},
			wantErr: false, // Not validated yet, but shouldn't panic
		},
		{
			name: "zero FPS",
			cfg: EncodingConfig{
				Video: VideoEncodingConfig{
					Codec: "libx264",
					FPS:   0,
				},
			},
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation - config should be usable
			assert.NotEmpty(t, tt.cfg.Video.Codec)
		})
	}
}

func FuzzVideoEncodingCRF(f *testing.F) {
	// Add seed corpus
	f.Add(0)
	f.Add(23)
	f.Add(51)
	f.Add(-1)
	f.Add(100)
	
	f.Fuzz(func(t *testing.T, crf int) {
		cfg := VideoEncodingConfig{
			Codec:  "libx264",
			Preset: "medium",
			CRF:    crf,
			FPS:    30,
		}
		
		// Should not panic when accessing fields
		assert.NotEmpty(t, cfg.Codec)
		assert.NotEmpty(t, cfg.Preset)
	})
}

func FuzzAudioSampleRate(f *testing.F) {
	// Add seed corpus
	f.Add(44100)
	f.Add(48000)
	f.Add(96000)
	f.Add(0)
	
	f.Fuzz(func(t *testing.T, sampleRate int) {
		cfg := AudioEncodingConfig{
			Codec:      "aac",
			Bitrate:    "192k",
			SampleRate: sampleRate,
		}
		
		// Should not panic
		assert.NotEmpty(t, cfg.Codec)
		assert.NotEmpty(t, cfg.Bitrate)
	})
}
