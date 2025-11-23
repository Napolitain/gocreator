package services

import (
	"fmt"
	"strconv"

	"gocreator/internal/config"
)

// EncodingService handles video encoding parameters
type EncodingService struct {
	config config.EncodingConfig
}

// NewEncodingService creates a new encoding service
func NewEncodingService(cfg config.EncodingConfig) *EncodingService {
	return &EncodingService{
		config: cfg,
	}
}

// BuildVideoArgs builds FFmpeg video encoding arguments
func (s *EncodingService) BuildVideoArgs() []string {
	args := make([]string, 0)

	// Video codec
	codec := s.config.Video.Codec
	if codec == "" {
		codec = "libx264"
	}
	args = append(args, "-c:v", codec)

	// Preset
	if s.config.Video.Preset != "" {
		args = append(args, "-preset", s.config.Video.Preset)
	}

	// CRF (Constant Rate Factor)
	if s.config.Video.CRF > 0 {
		args = append(args, "-crf", strconv.Itoa(s.config.Video.CRF))
	}

	// Bitrate
	if s.config.Video.Bitrate != "" && s.config.Video.Bitrate != "auto" {
		args = append(args, "-b:v", s.config.Video.Bitrate)
	}

	// FPS
	if s.config.Video.FPS > 0 {
		args = append(args, "-r", strconv.Itoa(s.config.Video.FPS))
	}

	// Pixel format
	pixfmt := s.config.Video.PixelFormat
	if pixfmt == "" {
		pixfmt = "yuv420p"
	}
	args = append(args, "-pix_fmt", pixfmt)

	// Add movflags for MP4 (fast start for web)
	if codec == "libx264" || codec == "libx265" {
		args = append(args, "-movflags", "+faststart")
	}

	return args
}

// BuildAudioArgs builds FFmpeg audio encoding arguments
func (s *EncodingService) BuildAudioArgs() []string {
	args := make([]string, 0)

	// Audio codec
	codec := s.config.Audio.Codec
	if codec == "" {
		codec = "aac"
	}
	args = append(args, "-c:a", codec)

	// Bitrate
	bitrate := s.config.Audio.Bitrate
	if bitrate == "" {
		bitrate = "192k"
	}
	args = append(args, "-b:a", bitrate)

	// Sample rate
	if s.config.Audio.SampleRate > 0 {
		args = append(args, "-ar", strconv.Itoa(s.config.Audio.SampleRate))
	}

	return args
}

// BuildAllArgs builds all encoding arguments
func (s *EncodingService) BuildAllArgs() []string {
	args := make([]string, 0)
	args = append(args, s.BuildVideoArgs()...)
	args = append(args, s.BuildAudioArgs()...)
	return args
}

// GetQualityPreset returns encoding config for a quality preset
func GetQualityPreset(quality string) config.EncodingConfig {
	switch quality {
	case "low":
		return config.EncodingConfig{
			Video: config.VideoEncodingConfig{
				Codec:       "libx264",
				Preset:      "veryfast",
				CRF:         28,
				Bitrate:     "1M",
				FPS:         30,
				PixelFormat: "yuv420p",
			},
			Audio: config.AudioEncodingConfig{
				Codec:      "aac",
				Bitrate:    "128k",
				SampleRate: 44100,
			},
		}
	case "medium":
		return config.EncodingConfig{
			Video: config.VideoEncodingConfig{
				Codec:       "libx264",
				Preset:      "medium",
				CRF:         23,
				Bitrate:     "auto",
				FPS:         30,
				PixelFormat: "yuv420p",
			},
			Audio: config.AudioEncodingConfig{
				Codec:      "aac",
				Bitrate:    "192k",
				SampleRate: 48000,
			},
		}
	case "high":
		return config.EncodingConfig{
			Video: config.VideoEncodingConfig{
				Codec:       "libx264",
				Preset:      "slow",
				CRF:         18,
				Bitrate:     "5M",
				FPS:         30,
				PixelFormat: "yuv420p",
			},
			Audio: config.AudioEncodingConfig{
				Codec:      "aac",
				Bitrate:    "256k",
				SampleRate: 48000,
			},
		}
	case "ultra":
		return config.EncodingConfig{
			Video: config.VideoEncodingConfig{
				Codec:       "libx265",
				Preset:      "slow",
				CRF:         16,
				Bitrate:     "10M",
				FPS:         60,
				PixelFormat: "yuv420p",
			},
			Audio: config.AudioEncodingConfig{
				Codec:      "aac",
				Bitrate:    "320k",
				SampleRate: 48000,
			},
		}
	default:
		return config.DefaultEncodingConfig()
	}
}

// DetectHardwareAcceleration detects available hardware acceleration
func DetectHardwareAcceleration() string {
	// This is a simplified version
	// In production, you'd actually probe FFmpeg for available encoders
	// For now, we'll return "none" and let users configure manually
	return "none"
}

// GetHardwareEncodingArgs returns hardware-accelerated encoding arguments
func GetHardwareEncodingArgs(hwType string) ([]string, error) {
	switch hwType {
	case "nvenc": // NVIDIA
		return []string{"-hwaccel", "cuda", "-c:v", "h264_nvenc"}, nil
	case "qsv": // Intel Quick Sync
		return []string{"-hwaccel", "qsv", "-c:v", "h264_qsv"}, nil
	case "videotoolbox": // Apple
		return []string{"-hwaccel", "videotoolbox", "-c:v", "h264_videotoolbox"}, nil
	case "none", "":
		return []string{}, nil
	default:
		return nil, fmt.Errorf("unsupported hardware acceleration: %s", hwType)
	}
}
