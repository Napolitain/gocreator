package config

// EffectConfig represents a visual effect configuration
type EffectConfig struct {
	Type   string        `yaml:"type"`             // ken-burns, text-overlay, blur-background, vignette, color-grade, film-grain, stabilize
	Slides interface{}   `yaml:"slides,omitempty"` // "all", or []int
	Config EffectDetails `yaml:",inline"`
}

// EffectDetails contains effect-specific configuration
type EffectDetails struct {
	// Ken Burns
	ZoomStart float64 `yaml:"zoom_start,omitempty"`
	ZoomEnd   float64 `yaml:"zoom_end,omitempty"`
	Direction string  `yaml:"direction,omitempty"` // left, right, up, down, center, random

	// Text Overlay
	Text              string  `yaml:"text,omitempty"`
	Position          string  `yaml:"position,omitempty"` // top-left, top-right, bottom-left, bottom-right, center
	OffsetX           int     `yaml:"offset_x,omitempty"`
	OffsetY           int     `yaml:"offset_y,omitempty"`
	Font              string  `yaml:"font,omitempty"`
	FontSize          int     `yaml:"font_size,omitempty"`
	Color             string  `yaml:"color,omitempty"`
	OutlineColor      string  `yaml:"outline_color,omitempty"`
	OutlineWidth      int     `yaml:"outline_width,omitempty"`
	BackgroundColor   string  `yaml:"background_color,omitempty"`
	BackgroundOpacity float64 `yaml:"background_opacity,omitempty"`
	FadeIn            float64 `yaml:"fade_in,omitempty"`
	FadeOut           float64 `yaml:"fade_out,omitempty"`

	// Blur Background
	BlurRadius int `yaml:"blur_radius,omitempty"`

	// Vignette
	Intensity float64 `yaml:"intensity,omitempty"`

	// Color Grade
	Brightness float64 `yaml:"brightness,omitempty"` // -1.0 to 1.0
	Contrast   float64 `yaml:"contrast,omitempty"`   // 0.0 to 2.0
	Saturation float64 `yaml:"saturation,omitempty"` // 0.0 to 3.0
	Hue        int     `yaml:"hue,omitempty"`        // -180 to 180
	Gamma      float64 `yaml:"gamma,omitempty"`      // 0.1 to 10.0

	// Stabilization
	Smoothing int `yaml:"smoothing,omitempty"` // 1-100
}

// ParseSlides parses the slides field into a slice of slide indices
func (e EffectConfig) ParseSlides(totalSlides int) []int {
	switch v := e.Slides.(type) {
	case string:
		if v == "all" {
			slides := make([]int, totalSlides)
			for i := range slides {
				slides[i] = i
			}
			return slides
		}
	case []interface{}:
		slides := make([]int, 0, len(v))
		for _, item := range v {
			if num, ok := item.(int); ok {
				slides = append(slides, num)
			}
		}
		return slides
	case []int:
		return v
	}
	return nil
}
