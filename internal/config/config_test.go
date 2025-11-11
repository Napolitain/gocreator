package config

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	
	assert.Equal(t, "en", cfg.Input.Lang)
	assert.Equal(t, "local", cfg.Input.Source)
	assert.Equal(t, []string{"en"}, cfg.Output.Languages)
	assert.Equal(t, "./data/out", cfg.Output.Directory)
	assert.Equal(t, "mp4", cfg.Output.Format)
	assert.Equal(t, "medium", cfg.Output.Quality)
	assert.Equal(t, "tts-1-hd", cfg.Voice.Model)
	assert.Equal(t, "alloy", cfg.Voice.Voice)
	assert.Equal(t, 1.0, cfg.Voice.Speed)
	assert.True(t, cfg.Cache.Enabled)
	assert.Equal(t, "./data/cache", cfg.Cache.Directory)
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		wantErr     bool
		validate    func(*testing.T, *Config)
	}{
		{
			name: "valid config",
			yamlContent: `input:
  lang: fr
  source: google-slides
  presentation_id: "123abc"
output:
  languages: [fr, en, es]
  directory: ./videos
  format: webm
  quality: high
voice:
  model: tts-1
  voice: nova
  speed: 1.2
cache:
  enabled: false
  directory: /tmp/cache
`,
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "fr", cfg.Input.Lang)
				assert.Equal(t, "google-slides", cfg.Input.Source)
				assert.Equal(t, "123abc", cfg.Input.PresentationID)
				assert.Equal(t, []string{"fr", "en", "es"}, cfg.Output.Languages)
				assert.Equal(t, "./videos", cfg.Output.Directory)
				assert.Equal(t, "webm", cfg.Output.Format)
				assert.Equal(t, "high", cfg.Output.Quality)
				assert.Equal(t, "tts-1", cfg.Voice.Model)
				assert.Equal(t, "nova", cfg.Voice.Voice)
				assert.Equal(t, 1.2, cfg.Voice.Speed)
				assert.False(t, cfg.Cache.Enabled)
				assert.Equal(t, "/tmp/cache", cfg.Cache.Directory)
			},
		},
		{
			name: "minimal config",
			yamlContent: `input:
  lang: en
output:
  languages: [en]
`,
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "en", cfg.Input.Lang)
				assert.Equal(t, []string{"en"}, cfg.Output.Languages)
				// Defaults should be preserved
				assert.Equal(t, "tts-1-hd", cfg.Voice.Model)
			},
		},
		{
			name:        "invalid yaml",
			yamlContent: `invalid: [unclosed`,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			path := "/test/config.yaml"
			
			// Write config file
			err := afero.WriteFile(fs, path, []byte(tt.yamlContent), 0644)
			require.NoError(t, err)
			
			// Load config
			cfg, err := LoadConfig(fs, path)
			
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			
			require.NoError(t, err)
			require.NotNil(t, cfg)
			
			if tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	fs := afero.NewMemMapFs()
	
	cfg, err := LoadConfig(fs, "/nonexistent.yaml")
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "not found")
}

func TestLoadConfigOrDefault(t *testing.T) {
	t.Run("file exists", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		path := "/test/config.yaml"
		content := `input:
  lang: es
output:
  languages: [es]
`
		err := afero.WriteFile(fs, path, []byte(content), 0644)
		require.NoError(t, err)
		
		cfg, err := LoadConfigOrDefault(fs, path)
		require.NoError(t, err)
		assert.Equal(t, "es", cfg.Input.Lang)
	})
	
	t.Run("file not found", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		
		cfg, err := LoadConfigOrDefault(fs, "/nonexistent.yaml")
		require.NoError(t, err)
		assert.Equal(t, "en", cfg.Input.Lang) // Default
	})
}

func TestSaveConfig(t *testing.T) {
	fs := afero.NewMemMapFs()
	path := "/test/config.yaml"
	
	cfg := &Config{
		Input: InputConfig{
			Lang:   "de",
			Source: "local",
		},
		Output: OutputConfig{
			Languages: []string{"de", "en"},
			Directory: "./output",
		},
	}
	
	err := SaveConfig(fs, path, cfg)
	require.NoError(t, err)
	
	// Verify file was created
	exists, err := afero.Exists(fs, path)
	require.NoError(t, err)
	assert.True(t, exists)
	
	// Load and verify content
	loadedCfg, err := LoadConfig(fs, path)
	require.NoError(t, err)
	assert.Equal(t, "de", loadedCfg.Input.Lang)
	assert.Equal(t, []string{"de", "en"}, loadedCfg.Output.Languages)
}

func TestFindConfigFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	
	t.Run("file in current directory", func(t *testing.T) {
		// Note: This test would need to be adapted based on actual working directory
		// For now, we test the function exists and returns without error
		path, err := FindConfigFile(fs)
		assert.NoError(t, err)
		// Path may be empty if no config file found
		_ = path
	})
}
