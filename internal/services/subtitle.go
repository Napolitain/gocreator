package services

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"gocreator/internal/config"
	"gocreator/internal/interfaces"

	"github.com/spf13/afero"
)

// SubtitleService handles subtitle generation and styling
type SubtitleService struct {
	fs     afero.Fs
	logger interfaces.Logger
}

// NewSubtitleService creates a new subtitle service
func NewSubtitleService(fs afero.Fs, logger interfaces.Logger) *SubtitleService {
	return &SubtitleService{
		fs:     fs,
		logger: logger,
	}
}

// SubtitleSegment represents a subtitle entry
type SubtitleSegment struct {
	Index     int
	StartTime float64
	EndTime   float64
	Text      string
}

// GenerateSRT generates an SRT subtitle file
func (s *SubtitleService) GenerateSRT(segments []SubtitleSegment, outputPath string) error {
	var content strings.Builder

	for _, seg := range segments {
		content.WriteString(fmt.Sprintf("%d\n", seg.Index))
		content.WriteString(fmt.Sprintf("%s --> %s\n",
			formatSRTTime(seg.StartTime),
			formatSRTTime(seg.EndTime)))
		content.WriteString(fmt.Sprintf("%s\n\n", seg.Text))
	}

	if err := afero.WriteFile(s.fs, outputPath, []byte(content.String()), 0644); err != nil {
		return fmt.Errorf("failed to write SRT file: %w", err)
	}

	s.logger.Info("Generated SRT file", "path", outputPath)
	return nil
}

// GenerateVTT generates a WebVTT subtitle file
func (s *SubtitleService) GenerateVTT(segments []SubtitleSegment, outputPath string) error {
	var content strings.Builder
	content.WriteString("WEBVTT\n\n")

	for _, seg := range segments {
		content.WriteString(fmt.Sprintf("%s --> %s\n",
			formatVTTTime(seg.StartTime),
			formatVTTTime(seg.EndTime)))
		content.WriteString(fmt.Sprintf("%s\n\n", seg.Text))
	}

	if err := afero.WriteFile(s.fs, outputPath, []byte(content.String()), 0644); err != nil {
		return fmt.Errorf("failed to write VTT file: %w", err)
	}

	s.logger.Info("Generated VTT file", "path", outputPath)
	return nil
}

// BurnSubtitles burns subtitles into video
func (s *SubtitleService) BurnSubtitles(ctx context.Context, videoPath, subtitlePath, outputPath string, cfg config.SubtitlesConfig) error {
	style := s.buildSubtitleStyle(cfg.Style)
	filter := fmt.Sprintf("subtitles=%s:force_style='%s'", subtitlePath, style)

	args := []string{
		"-y",
		"-i", videoPath,
		"-vf", filter,
		"-c:a", "copy",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	s.logger.Debug("Burning subtitles", "command", cmd.String())

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg error: %w, stderr: %s", err, stderr.String())
	}

	s.logger.Info("Subtitles burned successfully", "output", outputPath)
	return nil
}

func (s *SubtitleService) buildSubtitleStyle(style config.SubtitleStyleConfig) string {
	parts := []string{}

	if style.Font != "" {
		parts = append(parts, fmt.Sprintf("FontName=%s", style.Font))
	}

	if style.FontSize > 0 {
		parts = append(parts, fmt.Sprintf("FontSize=%d", style.FontSize))
	}

	if style.Bold {
		parts = append(parts, "Bold=1")
	}

	if style.Italic {
		parts = append(parts, "Italic=1")
	}

	if style.Color != "" {
		parts = append(parts, fmt.Sprintf("PrimaryColour=&H%s", colorToASS(style.Color)))
	}

	if style.OutlineColor != "" {
		parts = append(parts, fmt.Sprintf("OutlineColour=&H%s", colorToASS(style.OutlineColor)))
	}

	if style.OutlineWidth > 0 {
		parts = append(parts, fmt.Sprintf("Outline=%d", style.OutlineWidth))
	}

	if style.BackgroundColor != "" {
		alpha := int((1.0 - style.BackgroundOpacity) * 255)
		parts = append(parts, fmt.Sprintf("BackColour=&H%02X%s", alpha, colorToASS(style.BackgroundColor)))
	}

	if style.MarginVertical > 0 {
		parts = append(parts, fmt.Sprintf("MarginV=%d", style.MarginVertical))
	}

	if style.MarginHorizontal > 0 {
		parts = append(parts, fmt.Sprintf("MarginL=%d", style.MarginHorizontal))
		parts = append(parts, fmt.Sprintf("MarginR=%d", style.MarginHorizontal))
	}

	// Alignment (1-9, numpad style)
	alignment := 2 // bottom center default
	switch style.Position {
	case "top":
		alignment = 8
	case "middle":
		alignment = 5
	case "bottom":
		alignment = 2
	}
	switch style.Alignment {
	case "left":
		alignment -= 1
	case "right":
		alignment += 1
	}
	parts = append(parts, fmt.Sprintf("Alignment=%d", alignment))

	return strings.Join(parts, ",")
}

// CreateSegmentsFromTexts creates subtitle segments from text array
func (s *SubtitleService) CreateSegmentsFromTexts(texts []string, durations []float64) []SubtitleSegment {
	segments := make([]SubtitleSegment, len(texts))
	currentTime := 0.0

	for i, text := range texts {
		duration := durations[i]
		segments[i] = SubtitleSegment{
			Index:     i + 1,
			StartTime: currentTime,
			EndTime:   currentTime + duration,
			Text:      text,
		}
		currentTime += duration
	}

	return segments
}

// SplitTextIntoLines splits text into multiple lines based on max chars per line
func (s *SubtitleService) SplitTextIntoLines(text string, maxCharsPerLine int) []string {
	if maxCharsPerLine <= 0 {
		return []string{text}
	}

	words := strings.Fields(text)
	lines := []string{}
	currentLine := ""

	for _, word := range words {
		testLine := currentLine
		if currentLine != "" {
			testLine += " "
		}
		testLine += word

		if len(testLine) > maxCharsPerLine && currentLine != "" {
			lines = append(lines, currentLine)
			currentLine = word
		} else {
			currentLine = testLine
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}

func formatSRTTime(seconds float64) string {
	h := int(seconds / 3600)
	m := int((seconds - float64(h*3600)) / 60)
	s := int(seconds - float64(h*3600) - float64(m*60))
	ms := int((seconds - float64(int(seconds))) * 1000)

	return fmt.Sprintf("%02d:%02d:%02d,%03d", h, m, s, ms)
}

func formatVTTTime(seconds float64) string {
	h := int(seconds / 3600)
	m := int((seconds - float64(h*3600)) / 60)
	s := int(seconds - float64(h*3600) - float64(m*60))
	ms := int((seconds - float64(int(seconds))) * 1000)

	return fmt.Sprintf("%02d:%02d:%02d.%03d", h, m, s, ms)
}

func colorToASS(color string) string {
	// Convert common color names to BGR hex for ASS format
	colorMap := map[string]string{
		"white":  "FFFFFF",
		"black":  "000000",
		"red":    "0000FF",
		"green":  "00FF00",
		"blue":   "FF0000",
		"yellow": "00FFFF",
		"cyan":   "FFFF00",
		"magenta":"FF00FF",
	}

	if hex, ok := colorMap[strings.ToLower(color)]; ok {
		return hex
	}

	// If it's already a hex color, convert RGB to BGR
	color = strings.TrimPrefix(color, "#")
	if len(color) == 6 {
		// Convert RGB to BGR
		return color[4:6] + color[2:4] + color[0:2]
	}

	return "FFFFFF" // default to white
}
