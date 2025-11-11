package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"gocreator/internal/interfaces"

	"github.com/spf13/afero"
)

// VideoService handles video generation
type VideoService struct {
	fs         afero.Fs
	logger     interfaces.Logger
	transition TransitionConfig
}

// NewVideoService creates a new video service
func NewVideoService(fs afero.Fs, logger interfaces.Logger) *VideoService {
	return &VideoService{
		fs:         fs,
		logger:     logger,
		transition: TransitionConfig{Type: TransitionNone}, // Default: no transitions
	}
}

// SetTransition sets the transition configuration
func (s *VideoService) SetTransition(transition TransitionConfig) {
	s.transition = transition
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
	width, height, err := s.getMediaDimensions(slides[0])
	if err != nil {
		return fmt.Errorf("failed to get media dimensions: %w", err)
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
	// Check segment cache first
	cached, err := s.checkSegmentCache(slidePath, audioPath, outputPath, targetWidth, targetHeight)
	if err != nil {
		s.logger.Warn("Failed to check segment cache", "error", err)
	}
	if cached {
		s.logger.Info("Using cached video segment", "path", outputPath)
		return nil
	}
	
	// Check if the slide is actually a video
	isVideo, err := s.isVideoFile(slidePath)
	if err != nil {
		s.logger.Warn("Failed to check if file is video, treating as image", "path", slidePath, "error", err)
		isVideo = false
	}

	// Get slide/video dimensions
	iw, ih, err := s.getMediaDimensions(slidePath)
	if err != nil {
		return err
	}

	var cmd *exec.Cmd

	if isVideo {
		// For video input: use video duration, align audio at beginning
		// Video determines the duration, audio is aligned at the start
		s.logger.Debug("Processing video input", "path", slidePath)

		// Get video duration to use as the final duration
		videoDuration, err := s.getVideoDuration(slidePath)
		if err != nil {
			return fmt.Errorf("failed to get video duration: %w", err)
		}

		// Get audio duration and warn if significantly shorter than video
		audioDuration, err := s.getVideoDuration(audioPath)
		if err != nil {
			s.logger.Warn("Failed to get audio duration, proceeding anyway", "path", audioPath, "error", err)
		} else if audioDuration < videoDuration*0.8 { // Audio is less than 80% of video duration
			s.logger.Warn("Audio is significantly shorter than video, remainder will be silent",
				"video_duration", videoDuration,
				"audio_duration", audioDuration,
				"video_path", slidePath)
		}

		if targetWidth != iw || targetHeight != ih {
			// Need to scale and pad video
			scaleFilter := fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease", targetWidth, targetHeight)
			padFilter := fmt.Sprintf("pad=%d:%d:(ow-iw)/2:(oh-ih)/2,setsar=1", targetWidth, targetHeight)
			filterComplex := fmt.Sprintf("[0:v]%s,%s[v]", scaleFilter, padFilter)

			// Use video duration as primary, trim audio to match if longer
			cmd = exec.Command("ffmpeg", "-y",
				"-i", slidePath,
				"-i", audioPath,
				"-filter_complex", filterComplex,
				"-map", "[v]", "-map", "1:a:0",
				"-c:v", "libx264",
				"-c:a", "mp3", "-b:a", "192k",
				"-pix_fmt", "yuv420p",
				"-t", fmt.Sprintf("%.2f", videoDuration),
				outputPath)
		} else {
			// No scaling needed for video
			cmd = exec.Command("ffmpeg", "-y",
				"-i", slidePath,
				"-i", audioPath,
				"-map", "0:v:0", "-map", "1:a:0",
				"-c:v", "libx264",
				"-c:a", "mp3", "-b:a", "192k",
				"-pix_fmt", "yuv420p",
				"-t", fmt.Sprintf("%.2f", videoDuration),
				outputPath)
		}
	} else {
		// For image input: use audio duration (current behavior)
		s.logger.Debug("Processing image input", "path", slidePath)

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
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	s.logger.Debug("Running ffmpeg", "command", cmd.String())

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg error: %w, stderr: %s", err, stderr.String())
	}

	// Save segment hash for future cache hits
	if err := s.saveSegmentHash(slidePath, audioPath, outputPath, targetWidth, targetHeight); err != nil {
		s.logger.Warn("Failed to save segment hash", "error", err)
		// Don't fail the operation if hash saving fails
	}

	return nil
}

func (s *VideoService) concatenateVideos(videoFiles []string, outputPath string) error {
	// Check final video cache first
	cached, err := s.checkFinalVideoCache(videoFiles, outputPath)
	if err != nil {
		s.logger.Warn("Failed to check final video cache", "error", err)
	}
	if cached {
		s.logger.Info("Using cached final video", "path", outputPath)
		return nil
	}
	
	// If transitions are disabled or only one video, use simple concatenation
	if !s.transition.IsEnabled() || len(videoFiles) == 1 {
		if err := s.concatenateVideosSimple(videoFiles, outputPath); err != nil {
			return err
		}
	} else {
		// Use transitions with xfade filter
		if err := s.concatenateVideosWithTransitions(videoFiles, outputPath); err != nil {
			return err
		}
	}
	
	// Save final video hash for future cache hits
	if err := s.saveFinalVideoHash(videoFiles, outputPath); err != nil {
		s.logger.Warn("Failed to save final video hash", "error", err)
		// Don't fail the operation if hash saving fails
	}
	
	return nil
}

// concatenateVideosSimple concatenates videos without transitions
func (s *VideoService) concatenateVideosSimple(videoFiles []string, outputPath string) error {
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
	s.logger.Debug("Concatenating videos (no transitions)", "command", cmd.String())

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg concat error: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

// concatenateVideosWithTransitions concatenates videos with transition effects
func (s *VideoService) concatenateVideosWithTransitions(videoFiles []string, outputPath string) error {
	// Guard: This function requires at least 2 videos for transitions
	if len(videoFiles) < 2 {
		return fmt.Errorf("concatenateVideosWithTransitions requires at least 2 videos, got %d", len(videoFiles))
	}

	args := []string{"-y"}

	// Add all video inputs
	for _, video := range videoFiles {
		args = append(args, "-i", video)
	}

	// Build complex filter for transitions
	var filterComplex strings.Builder
	var audioMix strings.Builder

	transitionName := s.transition.GetFFmpegTransitionName()
	transitionDuration := s.transition.Duration

	// Get duration of each video segment for offset calculation
	durations := make([]float64, len(videoFiles))
	for i, video := range videoFiles {
		duration, err := s.getVideoDuration(video)
		if err != nil {
			s.logger.Warn("Failed to get video duration, using default", "video", video, "error", err)
			duration = 5.0 // Default fallback
		}
		durations[i] = duration
		
		// Warn if transition duration exceeds video duration
		if transitionDuration >= duration {
			s.logger.Warn("Transition duration meets or exceeds video duration, may cause unexpected behavior",
				"video", video,
				"video_duration", duration,
				"transition_duration", transitionDuration)
		}
	}

	// Generate xfade transitions between consecutive videos
	currentVideoLabel := "[0:v]"
	offset := 0.0

	for i := 0; i < len(videoFiles)-1; i++ {
		nextVideoLabel := fmt.Sprintf("[%d:v]", i+1)
		outputLabel := fmt.Sprintf("[v%d]", i)

		// Calculate offset: accumulated duration minus transition duration
		offset += durations[i] - transitionDuration

		// Add xfade filter
		filterComplex.WriteString(fmt.Sprintf(
			"%s%sxfade=transition=%s:duration=%.2f:offset=%.2f%s",
			currentVideoLabel, nextVideoLabel,
			transitionName, transitionDuration, offset,
			outputLabel,
		))

		if i < len(videoFiles)-2 {
			filterComplex.WriteString(";")
		}

		currentVideoLabel = outputLabel
	}

	// Final video output label
	finalVideoLabel := fmt.Sprintf("[v%d]", len(videoFiles)-2)

	// Mix audio streams
	audioMix.WriteString(";")
	for i := range videoFiles {
		audioMix.WriteString(fmt.Sprintf("[%d:a]", i))
	}
	audioMix.WriteString(fmt.Sprintf("concat=n=%d:v=0:a=1[outa]", len(videoFiles)))

	// Combine video and audio filters
	fullFilter := filterComplex.String() + audioMix.String()
	args = append(args, "-filter_complex", fullFilter)
	args = append(args, "-map", finalVideoLabel, "-map", "[outa]", outputPath)

	cmd := exec.Command("ffmpeg", args...)
	s.logger.Debug("Concatenating videos with transitions",
		"transition", transitionName,
		"duration", transitionDuration,
		"command", cmd.String())

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg concat with transitions error: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

func (s *VideoService) getMediaDimensions(mediaPath string) (int, int, error) {
	cmd := exec.Command("ffmpeg", "-i", mediaPath, "-vf", "scale", "-vframes", "1", "-f", "null", "-")
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
	if _, err := fmt.Sscanf(matches[0], "%dx%d", &width, &height); err != nil {
		return 0, 0, fmt.Errorf("failed to parse dimensions: %w", err)
	}

	return width, height, nil
}

// isVideoFile checks if a file is a video (not a static image)
func (s *VideoService) isVideoFile(filePath string) (bool, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0",
		"-show_entries", "stream=codec_type,duration", "-of", "default=noprint_wrappers=1", filePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("ffprobe check failed: %w", err)
	}

	outputStr := string(output)
	
	var hasVideoCodec bool
	var duration float64

	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "codec_type=video" {
			hasVideoCodec = true
		}
		if strings.HasPrefix(line, "duration=") {
			val := strings.TrimPrefix(line, "duration=")
			val = strings.TrimSpace(val)
			if d, err := strconv.ParseFloat(val, 64); err == nil {
				duration = d
			}
		}
	}

	// A video file must have video codec type and a positive duration
	return hasVideoCodec && duration > 0, nil
}

// getVideoDuration gets the duration of a video file in seconds
func (s *VideoService) getVideoDuration(videoPath string) (float64, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries",
		"format=duration", "-of", "default=noprint_wrappers=1:nokey=1", videoPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("ffprobe duration check failed: %w", err)
	}

	var duration float64
	if _, err := fmt.Sscanf(strings.TrimSpace(string(output)), "%f", &duration); err != nil {
		return 0, fmt.Errorf("failed to parse duration: %w", err)
	}

	return duration, nil
}

// computeSegmentHash computes a cache key for a video segment
func (s *VideoService) computeSegmentHash(slidePath, audioPath string, width, height int) (string, error) {
	// Read slide file
	slideData, err := afero.ReadFile(s.fs, slidePath)
	if err != nil {
		return "", fmt.Errorf("failed to read slide file: %w", err)
	}
	
	// Read audio file
	audioData, err := afero.ReadFile(s.fs, audioPath)
	if err != nil {
		return "", fmt.Errorf("failed to read audio file: %w", err)
	}
	
	// Compute hash of slide + audio + dimensions
	hasher := sha256.New()
	hasher.Write(slideData)
	hasher.Write(audioData)
	if _, err := fmt.Fprintf(hasher, "%dx%d", width, height); err != nil {
		return "", fmt.Errorf("failed to write dimensions to hash: %w", err)
	}
	
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// checkSegmentCache checks if a cached video segment exists and is valid
func (s *VideoService) checkSegmentCache(slidePath, audioPath, outputPath string, width, height int) (bool, error) {
	// Check if output file exists
	exists, err := afero.Exists(s.fs, outputPath)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}
	
	// Check if hash file exists
	hashPath := outputPath + ".hash"
	hashExists, err := afero.Exists(s.fs, hashPath)
	if err != nil {
		return false, err
	}
	if !hashExists {
		return false, nil
	}
	
	// Read stored hash
	storedHash, err := afero.ReadFile(s.fs, hashPath)
	if err != nil {
		return false, err
	}
	
	// Compute current hash
	currentHash, err := s.computeSegmentHash(slidePath, audioPath, width, height)
	if err != nil {
		return false, err
	}
	
	return string(storedHash) == currentHash, nil
}

// saveSegmentHash saves the hash for a video segment
func (s *VideoService) saveSegmentHash(slidePath, audioPath, outputPath string, width, height int) error {
	hash, err := s.computeSegmentHash(slidePath, audioPath, width, height)
	if err != nil {
		return err
	}
	
	hashPath := outputPath + ".hash"
	return afero.WriteFile(s.fs, hashPath, []byte(hash), 0644)
}

// computeFinalVideoHash computes a cache key for the final concatenated video
func (s *VideoService) computeFinalVideoHash(videoFiles []string) (string, error) {
	hasher := sha256.New()
	
	// Hash each video segment file
	for _, videoFile := range videoFiles {
		data, err := afero.ReadFile(s.fs, videoFile)
		if err != nil {
			return "", fmt.Errorf("failed to read video file %s: %w", videoFile, err)
		}
		hasher.Write(data)
	}
	
	// Include transition configuration in hash
	if _, err := fmt.Fprintf(hasher, "%s:%.2f", s.transition.Type, s.transition.Duration); err != nil {
		return "", fmt.Errorf("failed to write transition config to hash: %w", err)
	}
	
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// checkFinalVideoCache checks if a cached final video exists and is valid
func (s *VideoService) checkFinalVideoCache(videoFiles []string, outputPath string) (bool, error) {
	// Check if output file exists
	exists, err := afero.Exists(s.fs, outputPath)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}
	
	// Check if hash file exists
	hashPath := outputPath + ".hash"
	hashExists, err := afero.Exists(s.fs, hashPath)
	if err != nil {
		return false, err
	}
	if !hashExists {
		return false, nil
	}
	
	// Read stored hash
	storedHash, err := afero.ReadFile(s.fs, hashPath)
	if err != nil {
		return false, err
	}
	
	// Compute current hash
	currentHash, err := s.computeFinalVideoHash(videoFiles)
	if err != nil {
		return false, err
	}
	
	return string(storedHash) == currentHash, nil
}

// saveFinalVideoHash saves the hash for the final video
func (s *VideoService) saveFinalVideoHash(videoFiles []string, outputPath string) error {
	hash, err := s.computeFinalVideoHash(videoFiles)
	if err != nil {
		return err
	}
	
	hashPath := outputPath + ".hash"
	return afero.WriteFile(s.fs, hashPath, []byte(hash), 0644)
}
