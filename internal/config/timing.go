package config

// TimingConfig represents timing configuration
type TimingConfig struct {
	PerSlide              []SlideTimingConfig `yaml:"per_slide,omitempty"`
	DefaultImageDuration  interface{}         `yaml:"default_image_duration,omitempty"` // "auto" or float64
	MinSlideDuration      float64             `yaml:"min_slide_duration,omitempty"`
	MaxSlideDuration      float64             `yaml:"max_slide_duration,omitempty"`
}

// SlideTimingConfig represents timing for a specific slide
type SlideTimingConfig struct {
	Slide    int     `yaml:"slide"`
	Speed    float64 `yaml:"speed,omitempty"`    // Speed multiplier (1.0 = normal)
	Duration float64 `yaml:"duration,omitempty"` // Explicit duration override
}
