package services

import "fmt"

// TransitionType defines the type of transition between slides
type TransitionType string

const (
	// TransitionNone represents no transition (direct cut)
	TransitionNone TransitionType = "none"
	
	// TransitionFade represents a fade transition
	TransitionFade TransitionType = "fade"
	
	// TransitionWipeleft represents a wipe from right to left
	TransitionWipeleft TransitionType = "wipeleft"
	
	// TransitionWiperight represents a wipe from left to right
	TransitionWiperight TransitionType = "wiperight"
	
	// TransitionWipeup represents a wipe from bottom to top
	TransitionWipeup TransitionType = "wipeup"
	
	// TransitionWipedown represents a wipe from top to bottom
	TransitionWipedown TransitionType = "wipedown"
	
	// TransitionSlideleft represents a slide from right to left
	TransitionSlideleft TransitionType = "slideleft"
	
	// TransitionSlideright represents a slide from left to right
	TransitionSlideright TransitionType = "slideright"
	
	// TransitionSlideup represents a slide from bottom to top
	TransitionSlideup TransitionType = "slideup"
	
	// TransitionSlidedown represents a slide from top to bottom
	TransitionSlidedown TransitionType = "slidedown"
	
	// TransitionDissolve represents a dissolve transition
	TransitionDissolve TransitionType = "dissolve"
)

// TransitionConfig holds configuration for video transitions
type TransitionConfig struct {
	// Type specifies the transition effect to use
	Type TransitionType
	
	// Duration specifies the transition duration in seconds
	// Default is 0.5 seconds if not specified
	Duration float64
}

// DefaultTransitionConfig returns a default transition configuration
func DefaultTransitionConfig() TransitionConfig {
	return TransitionConfig{
		Type:     TransitionFade,
		Duration: 0.5,
	}
}

// Validate validates the transition configuration
func (tc TransitionConfig) Validate() error {
	if tc.Duration < 0 {
		return fmt.Errorf("transition duration must be non-negative, got %f", tc.Duration)
	}
	
	if tc.Duration > 5.0 {
		return fmt.Errorf("transition duration is too long (max 5.0s), got %f", tc.Duration)
	}
	
	// Validate transition type
	validTypes := map[TransitionType]bool{
		TransitionNone:        true,
		TransitionFade:        true,
		TransitionWipeleft:    true,
		TransitionWiperight:   true,
		TransitionWipeup:      true,
		TransitionWipedown:    true,
		TransitionSlideleft:   true,
		TransitionSlideright:  true,
		TransitionSlideup:     true,
		TransitionSlidedown:   true,
		TransitionDissolve:    true,
	}
	
	if !validTypes[tc.Type] {
		return fmt.Errorf("invalid transition type: %s", tc.Type)
	}
	
	return nil
}

// IsEnabled returns true if transitions are enabled (type is not "none")
func (tc TransitionConfig) IsEnabled() bool {
	return tc.Type != TransitionNone && tc.Duration > 0
}

// GetFFmpegTransitionName returns the FFmpeg xfade transition name
func (tc TransitionConfig) GetFFmpegTransitionName() string {
	switch tc.Type {
	case TransitionFade:
		return "fade"
	case TransitionWipeleft:
		return "wipeleft"
	case TransitionWiperight:
		return "wiperight"
	case TransitionWipeup:
		return "wipeup"
	case TransitionWipedown:
		return "wipedown"
	case TransitionSlideleft:
		return "slideleft"
	case TransitionSlideright:
		return "slideright"
	case TransitionSlideup:
		return "slideup"
	case TransitionSlidedown:
		return "slidedown"
	case TransitionDissolve:
		// Dissolve is similar to fade in FFmpeg
		return "fade"
	default:
		return ""
	}
}
