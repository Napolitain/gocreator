package services

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"gocreator/internal/interfaces"

	"github.com/spf13/afero"
)

// VideoService handles video generation
type VideoService struct {
	fs     afero.Fs
	logger interfaces.Logger
}

// NewVideoService creates a new video service
func NewVideoService(fs afero.Fs, logger interfaces.Logger) *VideoService {
	return &VideoService{
		fs:     fs,
		logger: logger,
	}
}

// GenerateFromSlides generates videos from slides and audio
func (s *VideoService) GenerateFromSlides(ctx context.Context, slides, audioPaths []string, outputPath string) error {
	if len(slides) != len(audioPaths) {
		return fmt.Errorf("slides and audio count mismatch: %d vs %d", len(slides), len(audioPaths))
	}

	if len(slides) == 0 {
		return fmt.Errorf("no slides provided")
	}

	// Get dimensions from first slide
	width, height, err := s.getImageDimensions(slides[0])
	if err != nil {
		return fmt.Errorf("failed to get image dimensions: %w", err)
	}

	// Ensure even dimensions for video encoding
	if width%2 != 0 {
		width--
	}
	if height%2 != 0 {
		height--
	}

	// Create output directory
	outputDir := filepath.Dir(outputPath)
	if err := s.fs.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create temporary video directory
	tempDir := filepath.Join(outputDir, ".temp")
	if err := s.fs.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Generate individual videos
	videoFiles := make([]string, len(slides))
	errors := make([]error, len(slides))
	var wg sync.WaitGroup

	for i := 0; i < len(slides); i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			videoPath := filepath.Join(tempDir, fmt.Sprintf("video_%d.mp4", idx))
			videoFiles[idx] = videoPath

			if err := s.generateSingleVideo(slides[idx], audioPaths[idx], videoPath, width, height); err != nil {
				errors[idx] = fmt.Errorf("failed to generate video %d: %w", idx, err)
			}
		}(i)
	}

	wg.Wait()

	// Check for errors
	for _, err := range errors {
		if err != nil {
			return err
		}
	}

	// Concatenate videos
	if err := s.concatenateVideos(videoFiles, outputPath); err != nil {
		return fmt.Errorf("failed to concatenate videos: %w", err)
	}

	s.logger.Info("Video created successfully", "path", outputPath)
	return nil
}

func (s *VideoService) generateSingleVideo(slidePath, audioPath, outputPath string, targetWidth, targetHeight int) error {
	// Get slide dimensions
	iw, ih, err := s.getImageDimensions(slidePath)
	if err != nil {
		return err
	}

	var cmd *exec.Cmd
	if targetWidth != iw || targetHeight != ih {
		// Need to scale and pad
		scaleFilter := fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease", targetWidth, targetHeight)
		padFilter := fmt.Sprintf("pad=%d:%d:(ow-iw)/2:(oh-ih)/2,setsar=1", targetWidth, targetHeight)
		filterComplex := fmt.Sprintf("%s,%s", scaleFilter, padFilter)

		cmd = exec.Command("ffmpeg", "-y", "-loop", "1", "-i", slidePath, "-i", audioPath,
			"-vf", filterComplex,
			"-c:v", "libx264", "-tune", "stillimage",
			"-c:a", "mp3", "-b:a", "192k",
			"-pix_fmt", "yuv420p", "-shortest",
			outputPath)
	} else {
		// No scaling needed
		cmd = exec.Command("ffmpeg", "-y", "-loop", "1", "-i", slidePath, "-i", audioPath,
			"-c:v", "libx264", "-tune", "stillimage",
			"-c:a", "mp3", "-b:a", "192k",
			"-pix_fmt", "yuv420p", "-shortest",
			outputPath)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	s.logger.Debug("Running ffmpeg", "command", cmd.String())

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg error: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

func (s *VideoService) concatenateVideos(videoFiles []string, outputPath string) error {
	args := []string{"-y"}

	for _, video := range videoFiles {
		args = append(args, "-i", video)
	}

	var filterComplex strings.Builder
	for i := range videoFiles {
		filterComplex.WriteString(fmt.Sprintf("[%d:v][%d:a]", i, i))
	}
	filterComplex.WriteString(fmt.Sprintf("concat=n=%d:v=1:a=1[outv][outa]", len(videoFiles)))

	args = append(args, "-filter_complex", filterComplex.String())
	args = append(args, "-map", "[outv]", "-map", "[outa]", outputPath)

	cmd := exec.Command("ffmpeg", args...)
	s.logger.Debug("Concatenating videos", "command", cmd.String())

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg concat error: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

func (s *VideoService) getImageDimensions(imagePath string) (int, int, error) {
	cmd := exec.Command("ffmpeg", "-i", imagePath, "-vf", "scale", "-vframes", "1", "-f", "null", "-")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, 0, fmt.Errorf("ffmpeg dimension check failed: %w", err)
	}

	outputStr := string(output)
	re := regexp.MustCompile(`(\d+)x(\d+)`)
	matches := re.FindStringSubmatch(outputStr)
	if len(matches) < 3 {
		return 0, 0, fmt.Errorf("failed to parse dimensions from ffmpeg output")
	}

	var width, height int
	fmt.Sscanf(matches[0], "%dx%d", &width, &height)

	return width, height, nil
}
