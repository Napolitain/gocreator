package config

// IntroConfig represents intro configuration
type IntroConfig struct {
	Enabled            bool           `yaml:"enabled,omitempty"`
	Video              string         `yaml:"video,omitempty"`
	Transition         string         `yaml:"transition,omitempty"`
	TransitionDuration float64        `yaml:"transition_duration,omitempty"`
	Template           TemplateConfig `yaml:"template,omitempty"`
}

// OutroConfig represents outro configuration
type OutroConfig struct {
	Enabled            bool           `yaml:"enabled,omitempty"`
	Video              string         `yaml:"video,omitempty"`
	Transition         string         `yaml:"transition,omitempty"`
	TransitionDuration float64        `yaml:"transition_duration,omitempty"`
	Template           TemplateConfig `yaml:"template,omitempty"`
}

// TemplateConfig represents intro/outro template configuration
type TemplateConfig struct {
	Enabled         bool    `yaml:"enabled,omitempty"`
	Type            string  `yaml:"type,omitempty"` // simple, professional, animated, call-to-action
	Text            string  `yaml:"text,omitempty"`
	Subtext         string  `yaml:"subtext,omitempty"`
	Logo            string  `yaml:"logo,omitempty"`
	BackgroundColor string  `yaml:"background_color,omitempty"`
	TextColor       string  `yaml:"text_color,omitempty"`
	Duration        float64 `yaml:"duration,omitempty"`
}
