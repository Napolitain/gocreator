package config

// ChaptersConfig represents chapter markers configuration
type ChaptersConfig struct {
	Enabled bool            `yaml:"enabled,omitempty"`
	Markers []ChapterMarker `yaml:"markers,omitempty"`
}

// ChapterMarker represents a single chapter marker
type ChapterMarker struct {
	Slide int    `yaml:"slide"`
	Title string `yaml:"title"`
}

// MetadataConfig represents video metadata configuration
type MetadataConfig struct {
	Title       string            `yaml:"title,omitempty"`
	Description string            `yaml:"description,omitempty"`
	Author      string            `yaml:"author,omitempty"`
	Copyright   string            `yaml:"copyright,omitempty"`
	Tags        []string          `yaml:"tags,omitempty"`
	Category    string            `yaml:"category,omitempty"`
	Language    string            `yaml:"language,omitempty"`
	Thumbnail   ThumbnailConfig   `yaml:"thumbnail,omitempty"`
}

// ThumbnailConfig represents thumbnail generation configuration
type ThumbnailConfig struct {
	Enabled     bool    `yaml:"enabled,omitempty"`
	Source      string  `yaml:"source,omitempty"`      // slide, frame, custom
	SlideIndex  int     `yaml:"slide_index,omitempty"`
	FrameTime   float64 `yaml:"frame_time,omitempty"`
	CustomFile  string  `yaml:"custom_file,omitempty"`
	OverlayText string  `yaml:"overlay_text,omitempty"`
}
