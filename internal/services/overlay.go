package services

import (
	"fmt"
	"strings"

	"gocreator/internal/config"
)

// OverlayService handles text overlays and watermarks
type OverlayService struct{}

// NewOverlayService creates a new overlay service
func NewOverlayService() *OverlayService {
	return &OverlayService{}
}

// BuildTextOverlayFilter builds an FFmpeg drawtext filter
func (s *OverlayService) BuildTextOverlayFilter(cfg config.EffectConfig) string {
	if cfg.Type != "text-overlay" {
		return ""
	}

	text := strings.ReplaceAll(cfg.Config.Text, "'", "\\'")
	text = strings.ReplaceAll(text, ":", "\\:")

	// Map position to coordinates
	x, y := s.getPosition(cfg.Config.Position, cfg.Config.OffsetX, cfg.Config.OffsetY)

	// Build drawtext filter
	parts := []string{
		fmt.Sprintf("text='%s'", text),
	}

	// Font settings
	if cfg.Config.Font != "" {
		parts = append(parts, fmt.Sprintf("font='%s'", cfg.Config.Font))
	}

	if cfg.Config.FontSize > 0 {
		parts = append(parts, fmt.Sprintf("fontsize=%d", cfg.Config.FontSize))
	}

	// Colors
	color := cfg.Config.Color
	if color == "" {
		color = "white"
	}
	parts = append(parts, fmt.Sprintf("fontcolor=%s", color))

	// Outline
	if cfg.Config.OutlineWidth > 0 {
		outlineColor := cfg.Config.OutlineColor
		if outlineColor == "" {
			outlineColor = "black"
		}
		parts = append(parts, fmt.Sprintf("borderw=%d", cfg.Config.OutlineWidth))
		parts = append(parts, fmt.Sprintf("bordercolor=%s", outlineColor))
	}

	// Background box
	if cfg.Config.BackgroundOpacity > 0 {
		parts = append(parts, "box=1")
		if cfg.Config.BackgroundColor != "" {
			parts = append(parts, fmt.Sprintf("boxcolor=%s", cfg.Config.BackgroundColor))
		}
		parts = append(parts, fmt.Sprintf("boxborderw=5"))
	}

	// Position
	parts = append(parts, fmt.Sprintf("x=%s", x))
	parts = append(parts, fmt.Sprintf("y=%s", y))

	// Build the complete filter
	filter := "drawtext=" + strings.Join(parts, ":")

	// Add fade in/out if specified
	if cfg.Config.FadeIn > 0 || cfg.Config.FadeOut > 0 {
		filter = s.addTextFade(filter, cfg.Config.FadeIn, cfg.Config.FadeOut)
	}

	return filter
}

func (s *OverlayService) getPosition(position string, offsetX, offsetY int) (string, string) {
	var x, y string

	switch position {
	case "top-left":
		x = fmt.Sprintf("%d", offsetX)
		y = fmt.Sprintf("%d", offsetY)
	case "top-right":
		x = fmt.Sprintf("w-tw-%d", offsetX)
		y = fmt.Sprintf("%d", offsetY)
	case "bottom-left":
		x = fmt.Sprintf("%d", offsetX)
		y = fmt.Sprintf("h-th-%d", offsetY)
	case "bottom-right":
		x = fmt.Sprintf("w-tw-%d", offsetX)
		y = fmt.Sprintf("h-th-%d", offsetY)
	case "center":
		x = "(w-tw)/2"
		y = "(h-th)/2"
	default:
		// Default to bottom-right with default offsets
		x = "w-tw-10"
		y = "h-th-10"
	}

	return x, y
}

func (s *OverlayService) addTextFade(filter string, fadeIn, fadeOut float64) string {
	// This would require more complex expression in drawtext
	// For simplicity, we'll just return the filter as-is
	// Full implementation would use enable='between(t,0,...)' expressions
	return filter
}

// BuildLogoOverlay builds a logo overlay filter
func (s *OverlayService) BuildLogoOverlay(logoPath, position string, opacity float64, offsetX, offsetY int) string {
	x, y := s.getLogoPosition(position, offsetX, offsetY)

	filter := fmt.Sprintf("movie=%s[logo];[in][logo]overlay=%s:%s", logoPath, x, y)

	if opacity > 0 && opacity < 1.0 {
		filter = fmt.Sprintf("movie=%s,format=rgba,colorchannelmixer=aa=%.2f[logo];[in][logo]overlay=%s:%s",
			logoPath, opacity, x, y)
	}

	return filter
}

func (s *OverlayService) getLogoPosition(position string, offsetX, offsetY int) (string, string) {
	var x, y string

	switch position {
	case "top-left":
		x = fmt.Sprintf("%d", offsetX)
		y = fmt.Sprintf("%d", offsetY)
	case "top-right":
		x = fmt.Sprintf("W-w-%d", offsetX)
		y = fmt.Sprintf("%d", offsetY)
	case "bottom-left":
		x = fmt.Sprintf("%d", offsetX)
		y = fmt.Sprintf("H-h-%d", offsetY)
	case "bottom-right":
		x = fmt.Sprintf("W-w-%d", offsetX)
		y = fmt.Sprintf("H-h-%d", offsetY)
	case "center":
		x = "(W-w)/2"
		y = "(H-h)/2"
	default:
		x = "W-w-10"
		y = "H-h-10"
	}

	return x, y
}
