package services

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"gocreator/internal/config"
	"gocreator/internal/interfaces"

	"github.com/spf13/afero"
)

// ExportService handles multi-format video export
type ExportService struct {
	fs     afero.Fs
	logger interfaces.Logger
}

// NewExportService creates a new export service
func NewExportService(fs afero.Fs, logger interfaces.Logger) *ExportService {
	return &ExportService{
		fs:     fs,
		logger: logger,
	}
}

// ExportToFormat exports video to a specific format
func (s *ExportService) ExportToFormat(ctx context.Context, inputPath, outputPath string, format config.FormatConfig) error {
	switch strings.ToLower(format.Type) {
	case "mp4":
		return s.exportToMP4(ctx, inputPath, outputPath, format)
	case "webm":
		return s.exportToWebM(ctx, inputPath, outputPath, format)
	case "gif":
		return s.exportToGIF(ctx, inputPath, outputPath, format)
	default:
		return fmt.Errorf("unsupported format: %s", format.Type)
	}
}

func (s *ExportService) exportToMP4(ctx context.Context, inputPath, outputPath string, format config.FormatConfig) error {
	args := []string{"-y", "-i", inputPath}

	// Resolution
	if format.Resolution != "" {
		width, height := parseResolution(format.Resolution)
		if width > 0 && height > 0 {
			args = append(args, "-vf", fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2",
				width, height, width, height))
		}
	}

	// Quality/encoding
	codec := format.Codec
	if codec == "" {
		codec = "libx264"
	}
	args = append(args, "-c:v", codec)

	// Add quality preset
	quality := format.Quality
	if quality == "" {
		quality = "medium"
	}
	encodingCfg := GetQualityPreset(quality)
	args = append(args, "-preset", encodingCfg.Video.Preset)
	args = append(args, "-crf", strconv.Itoa(encodingCfg.Video.CRF))

	// Audio
	args = append(args, "-c:a", "aac", "-b:a", "192k")

	// Output
	args = append(args, outputPath)

	return s.runFFmpeg(ctx, args)
}

func (s *ExportService) exportToWebM(ctx context.Context, inputPath, outputPath string, format config.FormatConfig) error {
	args := []string{"-y", "-i", inputPath}

	// Resolution
	if format.Resolution != "" {
		width, height := parseResolution(format.Resolution)
		if width > 0 && height > 0 {
			args = append(args, "-vf", fmt.Sprintf("scale=%d:%d", width, height))
		}
	}

	// VP9 encoding
	codec := format.Codec
	if codec == "" {
		codec = "libvpx-vp9"
	}
	args = append(args, "-c:v", codec)
	args = append(args, "-crf", "30")
	args = append(args, "-b:v", "0")

	// Audio
	args = append(args, "-c:a", "libopus", "-b:a", "128k")

	// Output
	args = append(args, outputPath)

	return s.runFFmpeg(ctx, args)
}

func (s *ExportService) exportToGIF(ctx context.Context, inputPath, outputPath string, format config.FormatConfig) error {
	fps := format.FPS
	if fps <= 0 {
		fps = 15
	}

	width, height := parseResolution(format.Resolution)
	if width <= 0 {
		width = 640
	}

	// Two-step process for optimized GIF
	// Step 1: Generate palette
	paletteFile := outputPath + ".palette.png"
	defer s.fs.Remove(paletteFile)

	paletteArgs := []string{
		"-y",
		"-i", inputPath,
		"-vf", fmt.Sprintf("fps=%d,scale=%d:-1:flags=lanczos,palettegen", fps, width),
		paletteFile,
	}

	if err := s.runFFmpeg(ctx, paletteArgs); err != nil {
		return fmt.Errorf("failed to generate palette: %w", err)
	}

	// Step 2: Generate GIF with palette
	gifArgs := []string{
		"-y",
		"-i", inputPath,
		"-i", paletteFile,
		"-filter_complex", fmt.Sprintf("fps=%d,scale=%d:-1:flags=lanczos[x];[x][1:v]paletteuse", fps, width),
	}

	if format.Optimize {
		// Add optimization flags
		gifArgs = append(gifArgs, "-loop", "0")
	}

	gifArgs = append(gifArgs, outputPath)

	return s.runFFmpeg(ctx, gifArgs)
}

// ExportForPlatform exports video optimized for a specific platform
func (s *ExportService) ExportForPlatform(ctx context.Context, inputPath, outputPath, platform string) error {
	var args []string

	switch strings.ToLower(platform) {
	case "youtube":
		// 1920x1080, high quality
		args = []string{
			"-y", "-i", inputPath,
			"-c:v", "libx264", "-preset", "slow", "-crf", "18",
			"-c:a", "aac", "-b:a", "256k",
			"-pix_fmt", "yuv420p",
			"-movflags", "+faststart",
			outputPath,
		}

	case "instagram":
		// 1080x1080 square
		args = []string{
			"-y", "-i", inputPath,
			"-vf", "scale=1080:1080:force_original_aspect_ratio=decrease,pad=1080:1080:(ow-iw)/2:(oh-ih)/2",
			"-c:v", "libx264", "-preset", "medium", "-crf", "23",
			"-c:a", "aac", "-b:a", "192k",
			"-t", "60", // Instagram limit
			outputPath,
		}

	case "tiktok":
		// 1080x1920 vertical
		args = []string{
			"-y", "-i", inputPath,
			"-vf", "scale=1080:1920:force_original_aspect_ratio=decrease,pad=1080:1920:(ow-iw)/2:(oh-ih)/2",
			"-c:v", "libx264", "-preset", "medium", "-crf", "23",
			"-c:a", "aac", "-b:a", "192k",
			outputPath,
		}

	case "twitter":
		// 1280x720, 2:20 max
		args = []string{
			"-y", "-i", inputPath,
			"-vf", "scale=1280:720:force_original_aspect_ratio=decrease,pad=1280:720:(ow-iw)/2:(oh-ih)/2",
			"-c:v", "libx264", "-preset", "medium", "-crf", "23",
			"-c:a", "aac", "-b:a", "192k",
			"-t", "140", // Twitter limit
			outputPath,
		}

	default:
		return fmt.Errorf("unsupported platform: %s", platform)
	}

	return s.runFFmpeg(ctx, args)
}

// GenerateThumbnail generates a thumbnail from video
func (s *ExportService) GenerateThumbnail(ctx context.Context, videoPath, outputPath string, timestamp float64) error {
	args := []string{
		"-y",
		"-ss", fmt.Sprintf("%.2f", timestamp),
		"-i", videoPath,
		"-vframes", "1",
		"-q:v", "2",
		outputPath,
	}

	return s.runFFmpeg(ctx, args)
}

func (s *ExportService) runFFmpeg(ctx context.Context, args []string) error {
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	s.logger.Debug("Running FFmpeg export", "command", cmd.String())

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg error: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

func parseResolution(res string) (int, int) {
	parts := strings.Split(res, "x")
	if len(parts) != 2 {
		return 0, 0
	}

	width, err1 := strconv.Atoi(parts[0])
	height, err2 := strconv.Atoi(parts[1])

	if err1 != nil || err2 != nil {
		return 0, 0
	}

	return width, height
}
