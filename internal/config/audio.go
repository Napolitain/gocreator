package config

// AudioConfig represents audio mixing configuration
type AudioConfig struct {
	BackgroundMusic BackgroundMusicConfig `yaml:"background_music,omitempty"`
	SoundEffects    []SoundEffectConfig   `yaml:"sound_effects,omitempty"`
	Ducking         DuckingConfig         `yaml:"ducking,omitempty"`
}

// BackgroundMusicConfig represents background music settings
type BackgroundMusicConfig struct {
	Enabled  bool    `yaml:"enabled,omitempty"`
	File     string  `yaml:"file,omitempty"`
	Volume   float64 `yaml:"volume,omitempty"`   // 0.0 to 1.0
	FadeIn   float64 `yaml:"fade_in,omitempty"`  // seconds
	FadeOut  float64 `yaml:"fade_out,omitempty"` // seconds
	Loop     bool    `yaml:"loop,omitempty"`
}

// SoundEffectConfig represents a sound effect configuration
type SoundEffectConfig struct {
	Slide  int     `yaml:"slide"`
	File   string  `yaml:"file"`
	Delay  float64 `yaml:"delay,omitempty"`  // seconds
	Volume float64 `yaml:"volume,omitempty"` // 0.0 to 1.0
}

// DuckingConfig represents audio ducking settings
type DuckingConfig struct {
	Enabled   bool    `yaml:"enabled,omitempty"`
	Threshold float64 `yaml:"threshold,omitempty"` // dB
	Ratio     float64 `yaml:"ratio,omitempty"`     // 0.0 to 1.0
	Attack    float64 `yaml:"attack,omitempty"`    // seconds
	Release   float64 `yaml:"release,omitempty"`   // seconds
}

// DefaultAudioConfig returns default audio configuration
func DefaultAudioConfig() AudioConfig {
	return AudioConfig{
		BackgroundMusic: BackgroundMusicConfig{
			Enabled: false,
			Volume:  0.15,
			FadeIn:  2.0,
			FadeOut: 3.0,
			Loop:    true,
		},
		Ducking: DuckingConfig{
			Enabled:   false,
			Threshold: -30,
			Ratio:     0.3,
			Attack:    0.5,
			Release:   1.0,
		},
	}
}
