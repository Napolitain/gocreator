package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
	"github.com/spf13/afero"
)

// Config represents the application configuration
type Config struct {
	Input      InputConfig      `yaml:"input"`
	Output     OutputConfig     `yaml:"output"`
	Voice      VoiceConfig      `yaml:"voice,omitempty"`
	Cache      CacheConfig      `yaml:"cache,omitempty"`
	Transition TransitionConfig `yaml:"transition,omitempty"`
}

// InputConfig represents input configuration
type InputConfig struct {
	Lang           string `yaml:"lang"`
	Source         string `yaml:"source,omitempty"`         // "local" or "google-slides"
	PresentationID string `yaml:"presentation_id,omitempty"` // Google Slides ID
}

// OutputConfig represents output configuration
type OutputConfig struct {
	Languages []string `yaml:"languages"`
	Directory string   `yaml:"directory,omitempty"`
	Format    string   `yaml:"format,omitempty"`    // mp4, webm, etc
	Quality   string   `yaml:"quality,omitempty"`   // low, medium, high, ultra
}

// VoiceConfig represents TTS voice configuration
type VoiceConfig struct {
	Model string  `yaml:"model,omitempty"` // tts-1, tts-1-hd
	Voice string  `yaml:"voice,omitempty"` // alloy, echo, fable, onyx, nova, shimmer
	Speed float64 `yaml:"speed,omitempty"` // 0.25 to 4.0
}

// CacheConfig represents cache configuration
type CacheConfig struct {
	Enabled   bool   `yaml:"enabled,omitempty"`
	Directory string `yaml:"directory,omitempty"`
}

// TransitionConfig represents transition configuration
type TransitionConfig struct {
	Type     string  `yaml:"type,omitempty"`     // none, fade, wipeleft, wiperight, etc.
	Duration float64 `yaml:"duration,omitempty"` // Duration in seconds
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Input: InputConfig{
			Lang:   "en",
			Source: "local",
		},
		Output: OutputConfig{
			Languages: []string{"en"},
			Directory: "./data/out",
			Format:    "mp4",
			Quality:   "medium",
		},
		Voice: VoiceConfig{
			Model: "tts-1-hd",
			Voice: "alloy",
			Speed: 1.0,
		},
		Cache: CacheConfig{
			Enabled:   true,
			Directory: "./data/cache",
		},
		Transition: TransitionConfig{
			Type:     "none",
			Duration: 0.0,
		},
	}
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(fs afero.Fs, path string) (*Config, error) {
	// Check if file exists
	exists, err := afero.Exists(fs, path)
	if err != nil {
		return nil, fmt.Errorf("failed to check config file: %w", err)
	}
	
	if !exists {
		return nil, fmt.Errorf("config file not found: %s", path)
	}

	// Read file
	data, err := afero.ReadFile(fs, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

// LoadConfigOrDefault loads config from file or returns default if not found
func LoadConfigOrDefault(fs afero.Fs, path string) (*Config, error) {
	exists, err := afero.Exists(fs, path)
	if err != nil {
		return nil, fmt.Errorf("failed to check config file: %w", err)
	}

	if !exists {
		return DefaultConfig(), nil
	}

	return LoadConfig(fs, path)
}

// SaveConfig saves configuration to a YAML file
func SaveConfig(fs afero.Fs, path string, cfg *Config) error {
	// Marshal to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Create directory if needed
	dir := filepath.Dir(path)
	if err := fs.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write file
	if err := afero.WriteFile(fs, path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// FindConfigFile searches for config file in current directory and parent directories
func FindConfigFile(fs afero.Fs) (string, error) {
	// Get current working directory
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Check common config file names
	configNames := []string{"gocreator.yaml", "gocreator.yml", ".gocreator.yaml", ".gocreator.yml"}

	// Search current directory and parent directories
	dir := wd
	for {
		for _, name := range configNames {
			path := filepath.Join(dir, name)
			exists, err := afero.Exists(fs, path)
			if err == nil && exists {
				return path, nil
			}
		}

		// Move to parent directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			break
		}
		dir = parent
	}

	return "", nil // Not found
}
