package config

// SubtitlesConfig represents subtitle configuration
type SubtitlesConfig struct {
	Enabled   bool                `yaml:"enabled,omitempty"`
	Generate  bool                `yaml:"generate,omitempty"`
	Languages interface{}         `yaml:"languages,omitempty"` // "all" or []string
	BurnIn    bool                `yaml:"burn_in,omitempty"`
	Style     SubtitleStyleConfig `yaml:"style,omitempty"`
	Timing    SubtitleTimingConfig `yaml:"timing,omitempty"`
}

// SubtitleStyleConfig represents subtitle styling
type SubtitleStyleConfig struct {
	Font                string  `yaml:"font,omitempty"`
	FontSize            int     `yaml:"font_size,omitempty"`
	Bold                bool    `yaml:"bold,omitempty"`
	Italic              bool    `yaml:"italic,omitempty"`
	Color               string  `yaml:"color,omitempty"`
	OutlineColor        string  `yaml:"outline_color,omitempty"`
	OutlineWidth        int     `yaml:"outline_width,omitempty"`
	ShadowColor         string  `yaml:"shadow_color,omitempty"`
	ShadowOffset        int     `yaml:"shadow_offset,omitempty"`
	BackgroundColor     string  `yaml:"background_color,omitempty"`
	BackgroundOpacity   float64 `yaml:"background_opacity,omitempty"`
	BackgroundPadding   int     `yaml:"background_padding,omitempty"`
	Position            string  `yaml:"position,omitempty"`   // top, bottom, middle
	Alignment           string  `yaml:"alignment,omitempty"`  // left, center, right
	MarginVertical      int     `yaml:"margin_vertical,omitempty"`
	MarginHorizontal    int     `yaml:"margin_horizontal,omitempty"`
}

// SubtitleTimingConfig represents subtitle timing settings
type SubtitleTimingConfig struct {
	MaxCharsPerLine int     `yaml:"max_chars_per_line,omitempty"`
	MaxLines        int     `yaml:"max_lines,omitempty"`
	MinDuration     float64 `yaml:"min_duration,omitempty"` // seconds
	MaxDuration     float64 `yaml:"max_duration,omitempty"` // seconds
}

// DefaultSubtitlesConfig returns default subtitle configuration
func DefaultSubtitlesConfig() SubtitlesConfig {
	return SubtitlesConfig{
		Enabled:   false,
		Generate:  true,
		Languages: "all",
		BurnIn:    true,
		Style: SubtitleStyleConfig{
			Font:              "Arial",
			FontSize:          24,
			Bold:              false,
			Italic:            false,
			Color:             "white",
			OutlineColor:      "black",
			OutlineWidth:      2,
			ShadowColor:       "black",
			ShadowOffset:      2,
			BackgroundColor:   "black",
			BackgroundOpacity: 0.5,
			BackgroundPadding: 5,
			Position:          "bottom",
			Alignment:         "center",
			MarginVertical:    20,
			MarginHorizontal:  10,
		},
		Timing: SubtitleTimingConfig{
			MaxCharsPerLine: 42,
			MaxLines:        2,
			MinDuration:     1.0,
			MaxDuration:     7.0,
		},
	}
}
