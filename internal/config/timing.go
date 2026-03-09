package config

const (
	// MediaAlignmentVideo keeps video slides aligned to their clip duration.
	MediaAlignmentVideo = "video"
	// MediaAlignmentSlide aligns video slides to narration duration.
	MediaAlignmentSlide = "slide"
)

// TimingConfig represents timing configuration
type TimingConfig struct {
	MediaAlignment       string              `yaml:"media_alignment,omitempty"` // video or slide
	PerSlide             []SlideTimingConfig `yaml:"per_slide,omitempty"`
	DefaultImageDuration interface{}         `yaml:"default_image_duration,omitempty"` // "auto" or float64
	MinSlideDuration     float64             `yaml:"min_slide_duration,omitempty"`
	MaxSlideDuration     float64             `yaml:"max_slide_duration,omitempty"`
}

// SlideTimingConfig represents timing for a specific slide
type SlideTimingConfig struct {
	Slide    int     `yaml:"slide"`
	Speed    float64 `yaml:"speed,omitempty"`    // Speed multiplier (1.0 = normal)
	Duration float64 `yaml:"duration,omitempty"` // Explicit duration override
}

// DefaultTimingConfig returns default timing configuration.
func DefaultTimingConfig() TimingConfig {
	return TimingConfig{
		MediaAlignment: MediaAlignmentVideo,
	}
}
