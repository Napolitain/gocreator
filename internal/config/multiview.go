package config

// MultiViewConfig defines multi-view layout configuration
type MultiViewConfig struct {
	Enabled bool           `yaml:"enabled"`
	Layouts []LayoutConfig `yaml:"layouts"`
}

// LayoutConfig defines a single layout for specific slides
type LayoutConfig struct {
	Type   string      `yaml:"type"` // split-horizontal, split-vertical, pip, grid, focus-gallery, custom
	Slides interface{} `yaml:"slides"` // "0-5", [0, 1, 2], "all", or single number

	// Split screen
	Ratio  string       `yaml:"ratio,omitempty"`  // "50:50", "60:40"
	Videos VideoSources `yaml:"videos,omitempty"` // For split screen
	Gap    int          `yaml:"gap,omitempty"`    // pixels between videos

	// Picture-in-Picture
	Main     string       `yaml:"main,omitempty"`
	Overlay  string       `yaml:"overlay,omitempty"`
	Position string       `yaml:"position,omitempty"` // top-left, top-right, bottom-left, bottom-right, center
	Size     string       `yaml:"size,omitempty"`     // "20%" or "320x180"
	Offset   []int        `yaml:"offset,omitempty"`   // [x, y] offset from edge
	Border   BorderConfig `yaml:"border,omitempty"`

	// Grid
	Rows        int      `yaml:"rows,omitempty"`
	Cols        int      `yaml:"cols,omitempty"`
	GridVideos  []string `yaml:"grid_videos,omitempty"`

	// Focus + Gallery
	Focus          string   `yaml:"focus,omitempty"`
	Gallery        []string `yaml:"gallery,omitempty"`
	GalleryPosition string  `yaml:"gallery_position,omitempty"` // left, right, top, bottom
	GallerySize    string   `yaml:"gallery_size,omitempty"`     // "20%"

	// Custom
	CustomVideos []CustomVideoConfig `yaml:"custom_videos,omitempty"`

	// Sync
	Sync SyncConfig `yaml:"sync,omitempty"`
}

// VideoSources defines video sources for split screen layouts
type VideoSources struct {
	Left   string `yaml:"left,omitempty"`
	Right  string `yaml:"right,omitempty"`
	Top    string `yaml:"top,omitempty"`
	Bottom string `yaml:"bottom,omitempty"`
}

// CustomVideoConfig defines custom positioned video
type CustomVideoConfig struct {
	Source   string         `yaml:"source"`
	Position [2]int         `yaml:"position"`          // x, y
	Size     [2]int         `yaml:"size"`              // width, height
	ZIndex   int            `yaml:"z_index,omitempty"` // stacking order
	Opacity  float64        `yaml:"opacity,omitempty"` // 0.0 to 1.0
	Effects  []EffectConfig `yaml:"effects,omitempty"`
}

// BorderConfig defines border styling
type BorderConfig struct {
	Width int    `yaml:"width"`
	Color string `yaml:"color"`
}

// SyncConfig defines synchronization settings
type SyncConfig struct {
	Enabled bool    `yaml:"enabled"`
	Offset  float64 `yaml:"offset"` // seconds
}

// ParseSlides parses the slides specification into a list of indices
func (l *LayoutConfig) ParseSlides(totalSlides int) []int {
	var indices []int

	switch v := l.Slides.(type) {
	case string:
		if v == "all" {
			// All slides
			for i := 0; i < totalSlides; i++ {
				indices = append(indices, i)
			}
		} else if len(v) > 0 && v[0] >= '0' && v[0] <= '9' {
			// Parse range like "0-5" or single number "3"
			indices = parseRange(v, totalSlides)
		}

	case int:
		// Single slide number
		if v >= 0 && v < totalSlides {
			indices = append(indices, v)
		}

	case []interface{}:
		// Array of slide numbers
		for _, item := range v {
			if idx, ok := item.(int); ok {
				if idx >= 0 && idx < totalSlides {
					indices = append(indices, idx)
				}
			}
		}
	}

	return indices
}

// parseRange parses a range string like "0-5" or single number "3"
func parseRange(s string, totalSlides int) []int {
	var indices []int

	// Check for range format "0-5"
	if len(s) > 0 {
		// Simple number
		if idx := parseInt(s); idx >= 0 && idx < totalSlides {
			indices = append(indices, idx)
			return indices
		}

		// Range format
		parts := splitString(s, "-")
		if len(parts) == 2 {
			start := parseInt(parts[0])
			end := parseInt(parts[1])

			if start >= 0 && end >= start && end < totalSlides {
				for i := start; i <= end; i++ {
					indices = append(indices, i)
				}
			}
		}
	}

	return indices
}

// Helper functions
func parseInt(s string) int {
	result := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result = result*10 + int(c-'0')
		} else {
			return -1
		}
	}
	return result
}

func splitString(s, sep string) []string {
	var parts []string
	current := ""

	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
			i += len(sep) - 1
		} else {
			current += string(s[i])
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}
