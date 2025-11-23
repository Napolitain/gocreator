package config

// PipConfig represents picture-in-picture configuration
type PipConfig struct {
	Enabled  bool               `yaml:"enabled,omitempty"`
	Overlays []PipOverlayConfig `yaml:"overlays,omitempty"`
}

// PipOverlayConfig represents a PiP overlay configuration
type PipOverlayConfig struct {
	Slides         interface{}        `yaml:"slides"` // "0-3" or []int
	Video          string             `yaml:"video"`
	Position       string             `yaml:"position,omitempty"` // top-left, top-right, bottom-left, bottom-right, custom
	CustomPosition PipPositionConfig  `yaml:"custom_position,omitempty"`
	Size           string             `yaml:"size,omitempty"` // "20%" or "320x240"
	Border         PipBorderConfig    `yaml:"border,omitempty"`
	Opacity        float64            `yaml:"opacity,omitempty"`
	FadeIn         float64            `yaml:"fade_in,omitempty"`
	FadeOut        float64            `yaml:"fade_out,omitempty"`
}

// PipPositionConfig represents custom PiP position
type PipPositionConfig struct {
	X *int `yaml:"x,omitempty"` // pixels from left
	Y *int `yaml:"y,omitempty"` // pixels from top
}

// PipBorderConfig represents PiP border configuration
type PipBorderConfig struct {
	Enabled bool   `yaml:"enabled,omitempty"`
	Width   int    `yaml:"width,omitempty"`
	Color   string `yaml:"color,omitempty"`
}
