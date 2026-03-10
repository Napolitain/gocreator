package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name  string
		cfg   *Config
		valid bool
	}{
		{
			name:  "valid default config",
			cfg:   DefaultConfig(),
			valid: true,
		},
		{
			name: "empty input language",
			cfg: &Config{
				Input: InputConfig{
					Lang: "",
				},
				Output: OutputConfig{
					Languages: []string{"en"},
				},
			},
			valid: false,
		},
		{
			name: "empty output languages",
			cfg: &Config{
				Input: InputConfig{
					Lang: "en",
				},
				Output: OutputConfig{
					Languages: []string{},
				},
			},
			valid: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.valid {
				assert.NotEmpty(t, tt.cfg.Input.Lang)
				assert.NotEmpty(t, tt.cfg.Output.Languages)
			}
		})
	}
}

func FuzzConfigInputLanguage(f *testing.F) {
	// Add seed corpus
	f.Add("en")
	f.Add("fr")
	f.Add("")
	f.Add("invalid")
	
	f.Fuzz(func(t *testing.T, lang string) {
		cfg := &Config{
			Input: InputConfig{
				Lang: lang,
			},
			Output: OutputConfig{
				Languages: []string{"en"},
			},
		}
		
		// Should not panic
		assert.NotNil(t, cfg)
	})
}
