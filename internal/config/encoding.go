package config

// EncodingConfig represents video/audio encoding configuration
type EncodingConfig struct {
	Video VideoEncodingConfig `yaml:"video,omitempty"`
	Audio AudioEncodingConfig `yaml:"audio,omitempty"`
}

// VideoEncodingConfig represents video encoding settings
type VideoEncodingConfig struct {
	Codec       string `yaml:"codec,omitempty"`        // libx264, libx265, libvpx-vp9
	Preset      string `yaml:"preset,omitempty"`       // ultrafast, fast, medium, slow, veryslow
	CRF         int    `yaml:"crf,omitempty"`          // 0-51 (lower = better quality)
	Bitrate     string `yaml:"bitrate,omitempty"`      // auto or specific like "5M"
	FPS         int    `yaml:"fps,omitempty"`          // Frame rate
	PixelFormat string `yaml:"pixel_format,omitempty"` // yuv420p, yuv444p
}

// AudioEncodingConfig represents audio encoding settings
type AudioEncodingConfig struct {
	Codec      string `yaml:"codec,omitempty"`       // aac, mp3, opus
	Bitrate    string `yaml:"bitrate,omitempty"`     // 128k, 192k, 256k, 320k
	SampleRate int    `yaml:"sample_rate,omitempty"` // 44100, 48000
}

// DefaultEncodingConfig returns default encoding settings
func DefaultEncodingConfig() EncodingConfig {
	return EncodingConfig{
		Video: VideoEncodingConfig{
			Codec:       "libx264",
			Preset:      "medium",
			CRF:         23,
			Bitrate:     "auto",
			FPS:         30,
			PixelFormat: "yuv420p",
		},
		Audio: AudioEncodingConfig{
			Codec:      "aac",
			Bitrate:    "192k",
			SampleRate: 48000,
		},
	}
}

// Validate validates encoding configuration
func (c EncodingConfig) Validate() error {
	// Validate video codec
	validVideoCodecs := map[string]bool{
		"libx264": true, "libx265": true, "libvpx-vp9": true,
		"h264_nvenc": true, "h264_qsv": true, "h264_videotoolbox": true,
	}
	if c.Video.Codec != "" && !validVideoCodecs[c.Video.Codec] {
		return &ValidationError{Field: "encoding.video.codec", Value: c.Video.Codec}
	}

	// Validate preset
	validPresets := map[string]bool{
		"ultrafast": true, "superfast": true, "veryfast": true, "faster": true,
		"fast": true, "medium": true, "slow": true, "slower": true, "veryslow": true,
	}
	if c.Video.Preset != "" && !validPresets[c.Video.Preset] {
		return &ValidationError{Field: "encoding.video.preset", Value: c.Video.Preset}
	}

	// Validate CRF range
	if c.Video.CRF < 0 || c.Video.CRF > 51 {
		return &ValidationError{Field: "encoding.video.crf", Value: c.Video.CRF}
	}

	// Validate audio codec
	validAudioCodecs := map[string]bool{
		"aac": true, "mp3": true, "opus": true, "libmp3lame": true,
	}
	if c.Audio.Codec != "" && !validAudioCodecs[c.Audio.Codec] {
		return &ValidationError{Field: "encoding.audio.codec", Value: c.Audio.Codec}
	}

	return nil
}
