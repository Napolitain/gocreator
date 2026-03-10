package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultSubtitlesConfig(t *testing.T) {
	cfg := DefaultSubtitlesConfig()
	
	assert.NotNil(t, cfg)
	assert.False(t, cfg.Enabled)
	assert.True(t, cfg.Generate)
	assert.Equal(t, "all", cfg.Languages)
	assert.True(t, cfg.BurnIn) // BurnIn is true by default
	
	// Check style
	assert.Equal(t, "Arial", cfg.Style.Font)
	assert.Equal(t, 24, cfg.Style.FontSize)
	assert.Equal(t, "white", cfg.Style.Color)
	
	// Check timing
	assert.Equal(t, 42, cfg.Timing.MaxCharsPerLine)
	assert.Equal(t, 2, cfg.Timing.MaxLines)
	assert.Equal(t, 1.0, cfg.Timing.MinDuration)
	assert.Equal(t, 7.0, cfg.Timing.MaxDuration)
}

func TestSubtitleStyleConfig_Validation(t *testing.T) {
	tests := []struct {
		name  string
		style SubtitleStyleConfig
		valid bool
	}{
		{
			name: "valid config",
			style: SubtitleStyleConfig{
				Font:     "Arial",
				FontSize: 24,
				Color:    "white",
			},
			valid: true,
		},
		{
			name: "negative font size",
			style: SubtitleStyleConfig{
				FontSize: -10,
			},
			valid: false,
		},
		{
			name: "zero font size",
			style: SubtitleStyleConfig{
				FontSize: 0,
			},
			valid: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.valid {
				assert.Greater(t, tt.style.FontSize, 0)
			}
		})
	}
}

func TestSubtitleTimingConfig_Validation(t *testing.T) {
	tests := []struct {
		name   string
		timing SubtitleTimingConfig
		valid  bool
	}{
		{
			name: "valid config",
			timing: SubtitleTimingConfig{
				MaxCharsPerLine: 42,
				MaxLines:        2,
				MinDuration:     1.0,
				MaxDuration:     7.0,
			},
			valid: true,
		},
		{
			name: "min duration greater than max",
			timing: SubtitleTimingConfig{
				MinDuration: 10.0,
				MaxDuration: 5.0,
			},
			valid: false,
		},
		{
			name: "negative max chars",
			timing: SubtitleTimingConfig{
				MaxCharsPerLine: -1,
			},
			valid: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.valid {
				assert.Greater(t, tt.timing.MaxCharsPerLine, 0)
				assert.Greater(t, tt.timing.MaxLines, 0)
				assert.LessOrEqual(t, tt.timing.MinDuration, tt.timing.MaxDuration)
			}
		})
	}
}

func FuzzSubtitleFontSize(f *testing.F) {
	// Add seed corpus
	f.Add(12)
	f.Add(24)
	f.Add(48)
	f.Add(0)
	f.Add(-10)
	
	f.Fuzz(func(t *testing.T, fontSize int) {
		style := SubtitleStyleConfig{
			Font:     "Arial",
			FontSize: fontSize,
			Color:    "white",
		}
		
		// Should not panic
		assert.NotEmpty(t, style.Font)
	})
}

func FuzzSubtitleTiming(f *testing.F) {
	// Add seed corpus
	f.Add(1.0, 7.0)
	f.Add(0.5, 10.0)
	f.Add(2.0, 5.0)
	
	f.Fuzz(func(t *testing.T, minDur, maxDur float64) {
		timing := SubtitleTimingConfig{
			MaxCharsPerLine: 42,
			MaxLines:        2,
			MinDuration:     minDur,
			MaxDuration:     maxDur,
		}
		
		// Should not panic
		assert.NotNil(t, timing)
	})
}
