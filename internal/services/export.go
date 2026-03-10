package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"gocreator/internal/config"
	"gocreator/internal/interfaces"

	"github.com/spf13/afero"
)

// ExportService handles multi-format video export
type ExportService struct {
	fs              afero.Fs
	logger          interfaces.Logger
	commandExecutor interfaces.CommandExecutor
}

// NewExportService creates a new export service
func NewExportService(fs afero.Fs, logger interfaces.Logger) *ExportService {
	return NewExportServiceWithExecutor(fs, logger, nil)
}

// NewExportServiceWithExecutor creates a new export service with an injected command executor.
func NewExportServiceWithExecutor(fs afero.Fs, logger interfaces.Logger, executor interfaces.CommandExecutor) *ExportService {
	if executor == nil {
		executor = newCommandExecutor()
	}

	return &ExportService{
		fs:              fs,
		logger:          logger,
		commandExecutor: executor,
	}
}

// ExportToFormat exports video to a specific format.
func (s *ExportService) ExportToFormat(ctx context.Context, inputPath, outputPath string, format config.FormatConfig, baseEncoding config.EncodingConfig) error {
	effectiveEncoding := resolveExportEncoding(format, baseEncoding)

	switch strings.ToLower(format.Type) {
	case "mp4":
		return s.exportToMP4(ctx, inputPath, outputPath, format, effectiveEncoding)
	case "webm":
		return s.exportToWebM(ctx, inputPath, outputPath, format, effectiveEncoding)
	case "gif":
		return s.exportToGIF(ctx, inputPath, outputPath, format)
	default:
		return fmt.Errorf("unsupported format: %s", format.Type)
	}
}

func (s *ExportService) exportToMP4(ctx context.Context, inputPath, outputPath string, format config.FormatConfig, encodingCfg config.EncodingConfig) error {
	args := []string{"-y", "-i", inputPath}

	// Resolution
	if format.Resolution != "" {
		width, height := parseResolution(format.Resolution)
		if width > 0 && height > 0 {
			args = append(args, "-vf", fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2",
				width, height, width, height))
		}
	}

	args = append(args, NewEncodingService(encodingCfg).BuildAllArgs()...)

	// Output
	args = append(args, outputPath)

	return s.runFFmpeg(ctx, args)
}

func (s *ExportService) exportToWebM(ctx context.Context, inputPath, outputPath string, format config.FormatConfig, encodingCfg config.EncodingConfig) error {
	args := []string{"-y", "-i", inputPath}

	// Resolution
	if format.Resolution != "" {
		width, height := parseResolution(format.Resolution)
		if width > 0 && height > 0 {
			args = append(args, "-vf", fmt.Sprintf("scale=%d:%d", width, height))
		}
	}

	if encodingCfg.Video.Codec == "" || strings.HasPrefix(encodingCfg.Video.Codec, "libx26") {
		encodingCfg.Video.Codec = "libvpx-vp9"
	}
	if encodingCfg.Video.CRF == 0 {
		encodingCfg.Video.CRF = 30
	}
	if encodingCfg.Video.Bitrate == "" || encodingCfg.Video.Bitrate == "auto" {
		encodingCfg.Video.Bitrate = "0"
	}
	if encodingCfg.Audio.Codec == "" || encodingCfg.Audio.Codec == "aac" {
		encodingCfg.Audio.Codec = "libopus"
	}
	if encodingCfg.Audio.Bitrate == "" {
		encodingCfg.Audio.Bitrate = "128k"
	}
	args = append(args, NewEncodingService(encodingCfg).BuildAllArgs()...)

	// Output
	args = append(args, outputPath)

	return s.runFFmpeg(ctx, args)
}

func (s *ExportService) exportToGIF(ctx context.Context, inputPath, outputPath string, format config.FormatConfig) error {
	fps := format.FPS
	if fps <= 0 {
		fps = 15
	}

	width, _ := parseResolution(format.Resolution)
	if width <= 0 {
		width = 640
	}

	// Two-step process for optimized GIF
	// Step 1: Generate palette
	paletteFile := outputPath + ".palette.png"
	defer func() {
		_ = s.fs.Remove(paletteFile) // Ignore error on cleanup
	}()

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
	s.logger.Debug("Running FFmpeg export", "command", formatCommand("ffmpeg", args...))

	result, err := s.commandExecutor.Run(ctx, "ffmpeg", args...)
	if err != nil {
		return fmt.Errorf("ffmpeg error: %w, stderr: %s", err, string(result.Stderr))
	}

	return nil
}

// MetadataChapter represents a chapter interval to embed in exported video metadata.
type MetadataChapter struct {
	StartTime float64
	EndTime   float64
	Title     string
}

// ApplyMetadata copies a video while embedding container metadata and chapter markers.
func (s *ExportService) ApplyMetadata(ctx context.Context, inputPath, outputPath string, metadata config.MetadataConfig, chapters []MetadataChapter) error {
	if isEmptyMetadata(metadata) && len(chapters) == 0 {
		return copyFileWithinFS(s.fs, inputPath, outputPath)
	}

	ffmetadata := s.buildFFMetadata(metadata, chapters)
	metadataPath := outputPath + "." + shortHash(outputPath) + ".ffmeta"
	if err := afero.WriteFile(s.fs, metadataPath, []byte(ffmetadata), 0644); err != nil {
		return fmt.Errorf("failed to write ffmetadata file: %w", err)
	}
	defer func() { _ = s.fs.Remove(metadataPath) }()

	args := []string{
		"-y",
		"-i", inputPath,
		"-i", metadataPath,
		"-map", "0",
		"-map_metadata", "1",
		"-map_chapters", "1",
		"-c", "copy",
		outputPath,
	}
	return s.runFFmpeg(ctx, args)
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

func (s *ExportService) buildFFMetadata(metadata config.MetadataConfig, chapters []MetadataChapter) string {
	var content strings.Builder
	content.WriteString(";FFMETADATA1\n")

	writeMetadataLine := func(key, value string) {
		value = strings.ReplaceAll(value, "\\", "\\\\")
		value = strings.ReplaceAll(value, "\n", "\\\n")
		value = strings.ReplaceAll(value, ";", "\\;")
		value = strings.ReplaceAll(value, "#", "\\#")
		value = strings.ReplaceAll(value, "=", "\\=")
		content.WriteString(fmt.Sprintf("%s=%s\n", key, value))
	}

	if metadata.Title != "" {
		writeMetadataLine("title", metadata.Title)
	}
	if metadata.Description != "" {
		writeMetadataLine("comment", metadata.Description)
	}
	if metadata.Author != "" {
		writeMetadataLine("artist", metadata.Author)
	}
	if metadata.Copyright != "" {
		writeMetadataLine("copyright", metadata.Copyright)
	}
	if len(metadata.Tags) > 0 {
		writeMetadataLine("keywords", strings.Join(metadata.Tags, ","))
	}
	if metadata.Category != "" {
		writeMetadataLine("genre", metadata.Category)
	}
	if metadata.Language != "" {
		writeMetadataLine("language", metadata.Language)
	}

	for _, chapter := range chapters {
		if chapter.EndTime <= chapter.StartTime || strings.TrimSpace(chapter.Title) == "" {
			continue
		}
		content.WriteString("[CHAPTER]\n")
		content.WriteString("TIMEBASE=1/1000\n")
		content.WriteString(fmt.Sprintf("START=%d\n", int(chapter.StartTime*1000)))
		content.WriteString(fmt.Sprintf("END=%d\n", int(chapter.EndTime*1000)))
		writeMetadataLine("title", chapter.Title)
	}

	return content.String()
}

func isEmptyMetadata(metadata config.MetadataConfig) bool {
	return metadata.Title == "" &&
		metadata.Description == "" &&
		metadata.Author == "" &&
		metadata.Copyright == "" &&
		len(metadata.Tags) == 0 &&
		metadata.Category == "" &&
		metadata.Language == ""
}

func shortHash(input string) string {
	sum := sha256.Sum256([]byte(input))
	return hex.EncodeToString(sum[:])[:12]
}

func copyFileWithinFS(fs afero.Fs, sourcePath, targetPath string) error {
	source, err := fs.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", sourcePath, err)
	}
	defer func() { _ = source.Close() }()

	if err := fs.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("failed to create target directory %s: %w", filepath.Dir(targetPath), err)
	}

	target, err := fs.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target file %s: %w", targetPath, err)
	}
	defer func() { _ = target.Close() }()

	if _, err := io.Copy(target, source); err != nil {
		return fmt.Errorf("failed to copy %s to %s: %w", sourcePath, targetPath, err)
	}

	return nil
}

func resolveExportEncoding(format config.FormatConfig, base config.EncodingConfig) config.EncodingConfig {
	encodingCfg := base
	if reflect.DeepEqual(encodingCfg, config.EncodingConfig{}) {
		encodingCfg = config.DefaultEncodingConfig()
	}

	if format.Quality != "" {
		encodingCfg = GetQualityPreset(format.Quality)
	}
	if format.Codec != "" {
		encodingCfg.Video.Codec = format.Codec
	}
	if format.FPS > 0 {
		encodingCfg.Video.FPS = format.FPS
	}
	return encodingCfg
}
