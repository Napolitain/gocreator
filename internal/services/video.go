package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"gocreator/internal/config"
	"gocreator/internal/interfaces"

	"github.com/spf13/afero"
)

// VideoService handles video generation
type VideoService struct {
	fs               afero.Fs
	logger           interfaces.Logger
	commandExecutor  interfaces.CommandExecutor
	transition       TransitionConfig
	mediaAlignment   string
	effects          []config.EffectConfig
	effectService    *EffectService
	overlayService   *OverlayService
	multiViewService *MultiViewService
	multiViewConfig  *config.MultiViewConfig
}

// NewVideoService creates a new video service
func NewVideoService(fs afero.Fs, logger interfaces.Logger) *VideoService {
	return NewVideoServiceWithExecutor(fs, logger, nil)
}

// NewVideoServiceWithExecutor creates a new video service with an injected command executor.
func NewVideoServiceWithExecutor(fs afero.Fs, logger interfaces.Logger, executor interfaces.CommandExecutor) *VideoService {
	if executor == nil {
		executor = newCommandExecutor()
	}

	return &VideoService{
		fs:               fs,
		logger:           logger,
		commandExecutor:  executor,
		transition:       TransitionConfig{Type: TransitionNone}, // Default: no transitions
		mediaAlignment:   config.MediaAlignmentVideo,
		effectService:    NewEffectServiceWithExecutor(fs, logger, executor),
		overlayService:   NewOverlayService(),
		multiViewService: NewMultiViewService(fs, logger),
	}
}

// SetTransition sets the transition configuration
func (s *VideoService) SetTransition(transition TransitionConfig) {
	s.transition = transition
}

// SetMediaAlignment sets how video slides are aligned against narration.
func (s *VideoService) SetMediaAlignment(alignment string) error {
	normalized, err := normalizeMediaAlignment(alignment)
	if err != nil {
		return err
	}

	s.mediaAlignment = normalized
	return nil
}

// SetEffects sets the per-slide effects configuration.
func (s *VideoService) SetEffects(effects []config.EffectConfig) {
	if len(effects) == 0 {
		s.effects = nil
		return
	}

	s.effects = make([]config.EffectConfig, len(effects))
	copy(s.effects, effects)
}

// SetMultiView sets the multi-view configuration
func (s *VideoService) SetMultiView(multiViewConfig *config.MultiViewConfig) {
	s.multiViewConfig = multiViewConfig
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
	effectsBySlide := s.resolveEffectsForSlides(slides)
	errors := make([]error, len(slides))
	var wg sync.WaitGroup

	for i := 0; i < len(slides); i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			videoPath := filepath.Join(tempDir, fmt.Sprintf("video_%d.mp4", idx))
			videoFiles[idx] = videoPath

			if err := s.generateSingleVideo(ctx, slides[idx], audioPaths[idx], videoPath, width, height, effectsBySlide[idx]); err != nil {
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

	// Apply multi-view layouts if configured
	if err := s.applyMultiViewLayouts(ctx, videoFiles, tempDir, width, height); err != nil {
		return fmt.Errorf("failed to apply multi-view layouts: %w", err)
	}

	// Concatenate videos
	if err := s.concatenateVideos(videoFiles, outputPath); err != nil {
		return fmt.Errorf("failed to concatenate videos: %w", err)
	}

	s.logger.Info("Video created successfully", "path", outputPath)
	return nil
}

func (s *VideoService) generateSingleVideo(
	ctx context.Context,
	slidePath,
	audioPath,
	outputPath string,
	targetWidth,
	targetHeight int,
	effects []config.EffectConfig,
) error {
	resolvedEffects := s.resolveEffectsForSlide(slidePath, effects)

	if len(resolvedEffects) == 0 {
		cached, err := s.checkSegmentCache(slidePath, audioPath, outputPath, targetWidth, targetHeight, s.mediaAlignment, nil)
		if err != nil {
			s.logger.Warn("Failed to check segment cache", "error", err)
		}
		if cached {
			s.logger.Info("Using cached video segment", "path", outputPath)
			return nil
		}
	}

	// Check if the slide is actually a video.
	isVideo, err := s.isVideoFile(slidePath)
	if err != nil {
		s.logger.Warn("Failed to check if file is video, treating as image", "path", slidePath, "error", err)
		isVideo = false
	}

	if err := s.validateEffectsForSlide(slidePath, isVideo, resolvedEffects); err != nil {
		return err
	}

	if len(resolvedEffects) > 0 {
		cached, err := s.checkSegmentCache(slidePath, audioPath, outputPath, targetWidth, targetHeight, s.mediaAlignment, resolvedEffects)
		if err != nil {
			s.logger.Warn("Failed to check segment cache", "error", err)
		}
		if cached {
			s.logger.Info("Using cached video segment", "path", outputPath)
			return nil
		}
	}

	// Get slide/video dimensions
	iw, ih, err := s.getMediaDimensions(slidePath)
	if err != nil {
		return err
	}

	input := videoRenderInput{
		slidePath:    slidePath,
		audioPath:    audioPath,
		outputPath:   outputPath,
		targetWidth:  targetWidth,
		targetHeight: targetHeight,
		inputWidth:   iw,
		inputHeight:  ih,
		effects:      resolvedEffects,
	}

	if isVideo {
		s.logger.Debug("Processing video input", "path", slidePath)

		videoDuration, err := s.getVideoDuration(slidePath)
		if err != nil {
			return fmt.Errorf("failed to get video duration: %w", err)
		}

		audioDuration, err := s.getVideoDuration(audioPath)
		if err != nil {
			return fmt.Errorf("failed to get audio duration: %w", err)
		}

		if audioDuration < videoDuration*0.8 && s.mediaAlignment != config.MediaAlignmentSlide {
			s.logger.Warn("Audio is significantly shorter than video, remainder will follow clip audio or stay silent",
				"video_duration", videoDuration,
				"audio_duration", audioDuration,
				"video_path", slidePath)
		}

		hasEmbeddedAudio, err := s.hasAudioStream(slidePath)
		if err != nil {
			return fmt.Errorf("failed to inspect embedded audio for %s: %w", slidePath, err)
		}

		input.videoDuration = videoDuration
		input.audioDuration = audioDuration
		input.isVideo = true
		input.hasEmbeddedAudio = hasEmbeddedAudio
		input.alignToSlide = s.mediaAlignment == config.MediaAlignmentSlide
	} else {
		s.logger.Debug("Processing image input", "path", slidePath)

		if len(resolvedEffects) > 0 {
			audioDuration, err := s.getVideoDuration(audioPath)
			if err != nil {
				return fmt.Errorf("failed to get audio duration: %w", err)
			}
			input.audioDuration = audioDuration
		}
	}

	if stabilizeEffect, ok := s.findSlideEffect(resolvedEffects, "stabilize"); ok {
		input.stabilizationTransformPath = outputPath + ".transforms.trf"
		defer func() { _ = s.fs.Remove(input.stabilizationTransformPath) }()

		if err := s.runStabilizationDetect(ctx, slidePath, stabilizeEffect, input.stabilizationTransformPath); err != nil {
			return err
		}
	}

	args, err := s.buildSingleVideoArgs(input)
	if err != nil {
		return err
	}

	s.logger.Debug("Running ffmpeg", "command", formatCommand("ffmpeg", args...))

	result, err := s.commandExecutor.Run(ctx, "ffmpeg", args...)
	if err != nil {
		return fmt.Errorf("ffmpeg error: %w, stderr: %s", err, string(result.Stderr))
	}

	// Save segment hash for future cache hits
	if err := s.saveSegmentHash(slidePath, audioPath, outputPath, targetWidth, targetHeight, s.mediaAlignment, resolvedEffects); err != nil {
		s.logger.Warn("Failed to save segment hash", "error", err)
		// Don't fail the operation if hash saving fails
	}

	return nil
}

type videoRenderInput struct {
	slidePath                  string
	audioPath                  string
	outputPath                 string
	targetWidth                int
	targetHeight               int
	inputWidth                 int
	inputHeight                int
	videoDuration              float64
	audioDuration              float64
	isVideo                    bool
	hasEmbeddedAudio           bool
	alignToSlide               bool
	effects                    []config.EffectConfig
	stabilizationTransformPath string
}

func (s *VideoService) buildSingleVideoArgs(input videoRenderInput) ([]string, error) {
	if len(input.effects) > 0 {
		return s.buildSingleVideoArgsWithEffects(input)
	}

	args := []string{"-y"}

	if input.isVideo {
		if input.alignToSlide {
			args = append(args, "-stream_loop", "-1")
		}
		args = append(args, "-i", input.slidePath, "-i", input.audioPath)

		videoMap := "0:v:0"
		audioMap := "1:a:0"
		filters := make([]string, 0, 2)

		if input.targetWidth != input.inputWidth || input.targetHeight != input.inputHeight {
			scaleFilter := fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease", input.targetWidth, input.targetHeight)
			padFilter := fmt.Sprintf("pad=%d:%d:(ow-iw)/2:(oh-ih)/2,setsar=1", input.targetWidth, input.targetHeight)
			filters = append(filters, fmt.Sprintf("[0:v]%s,%s[v]", scaleFilter, padFilter))
			videoMap = "[v]"
		}

		if input.hasEmbeddedAudio {
			durationMode := "first"
			if input.alignToSlide {
				durationMode = "shortest"
			}
			filters = append(filters, fmt.Sprintf("[0:a:0][1:a:0]amix=inputs=2:duration=%s:dropout_transition=0[a]", durationMode))
			audioMap = "[a]"
		}

		if len(filters) > 0 {
			args = append(args, "-filter_complex", strings.Join(filters, ";"))
		}

		targetDuration := input.videoDuration
		if input.alignToSlide {
			targetDuration = input.audioDuration
		}

		args = append(args,
			"-map", videoMap,
			"-map", audioMap,
			"-c:v", "libx264",
			"-c:a", "mp3", "-b:a", "192k",
			"-pix_fmt", "yuv420p",
			"-t", fmt.Sprintf("%.2f", targetDuration),
			input.outputPath,
		)

		return args, nil
	}

	args = append(args, "-loop", "1", "-i", input.slidePath, "-i", input.audioPath)

	if input.targetWidth != input.inputWidth || input.targetHeight != input.inputHeight {
		scaleFilter := fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease", input.targetWidth, input.targetHeight)
		padFilter := fmt.Sprintf("pad=%d:%d:(ow-iw)/2:(oh-ih)/2,setsar=1", input.targetWidth, input.targetHeight)
		args = append(args, "-vf", fmt.Sprintf("%s,%s", scaleFilter, padFilter))
	}

	args = append(args,
		"-c:v", "libx264", "-tune", "stillimage",
		"-c:a", "mp3", "-b:a", "192k",
		"-pix_fmt", "yuv420p", "-shortest",
		input.outputPath,
	)

	return args, nil
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

	s.logger.Debug("Concatenating videos (no transitions)", "command", formatCommand("ffmpeg", args...))

	result, err := s.commandExecutor.Run(context.Background(), "ffmpeg", args...)
	if err != nil {
		return fmt.Errorf("ffmpeg concat error: %w, stderr: %s", err, string(result.Stderr))
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

	s.logger.Debug("Concatenating videos with transitions",
		"transition", transitionName,
		"duration", transitionDuration,
		"command", formatCommand("ffmpeg", args...))

	result, err := s.commandExecutor.Run(context.Background(), "ffmpeg", args...)
	if err != nil {
		return fmt.Errorf("ffmpeg concat with transitions error: %w, stderr: %s", err, string(result.Stderr))
	}

	return nil
}

func (s *VideoService) getMediaDimensions(mediaPath string) (int, int, error) {
	result, err := s.commandExecutor.Run(context.Background(), "ffmpeg", "-i", mediaPath, "-vf", "scale", "-vframes", "1", "-f", "null", "-")
	if err != nil {
		return 0, 0, fmt.Errorf("ffmpeg dimension check failed: %w", err)
	}

	outputStr := string(result.CombinedOutput())
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
	result, err := s.commandExecutor.Run(context.Background(), "ffprobe", "-v", "error", "-select_streams", "v:0",
		"-show_entries", "stream=codec_type,duration", "-of", "default=noprint_wrappers=1", filePath)
	if err != nil {
		return false, fmt.Errorf("ffprobe check failed: %w", err)
	}

	outputStr := string(result.CombinedOutput())

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

func (s *VideoService) hasAudioStream(filePath string) (bool, error) {
	result, err := s.commandExecutor.Run(context.Background(), "ffprobe", "-v", "error",
		"-show_entries", "stream=codec_type", "-of", "default=noprint_wrappers=1:nokey=1", filePath)
	if err != nil {
		return false, fmt.Errorf("ffprobe audio stream check failed: %w", err)
	}

	for _, line := range strings.Split(string(result.CombinedOutput()), "\n") {
		if strings.TrimSpace(line) == "audio" {
			return true, nil
		}
	}

	return false, nil
}

// getVideoDuration gets the duration of a video file in seconds
func (s *VideoService) getVideoDuration(videoPath string) (float64, error) {
	result, err := s.commandExecutor.Run(context.Background(), "ffprobe", "-v", "error", "-show_entries",
		"format=duration", "-of", "default=noprint_wrappers=1:nokey=1", videoPath)
	if err != nil {
		return 0, fmt.Errorf("ffprobe duration check failed: %w", err)
	}

	var duration float64
	if _, err := fmt.Sscanf(strings.TrimSpace(string(result.CombinedOutput())), "%f", &duration); err != nil {
		return 0, fmt.Errorf("failed to parse duration: %w", err)
	}

	return duration, nil
}

// computeSegmentHash computes a cache key for a video segment.
func (s *VideoService) computeSegmentHash(
	slidePath,
	audioPath string,
	width,
	height int,
	mediaAlignment string,
	effects []config.EffectConfig,
) (string, error) {
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
	if _, err := hasher.Write([]byte(mediaAlignment)); err != nil {
		return "", fmt.Errorf("failed to write media alignment to hash: %w", err)
	}
	if len(effects) > 0 {
		effectSignature, err := serializeEffectsForCache(effects)
		if err != nil {
			return "", err
		}
		if _, err := hasher.Write([]byte(effectSignature)); err != nil {
			return "", fmt.Errorf("failed to write effects to hash: %w", err)
		}
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// checkSegmentCache checks if a cached video segment exists and is valid
func (s *VideoService) checkSegmentCache(
	slidePath,
	audioPath,
	outputPath string,
	width,
	height int,
	mediaAlignment string,
	effects []config.EffectConfig,
) (bool, error) {
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
	currentHash, err := s.computeSegmentHash(slidePath, audioPath, width, height, mediaAlignment, effects)
	if err != nil {
		return false, err
	}

	return string(storedHash) == currentHash, nil
}

// saveSegmentHash saves the hash for a video segment
func (s *VideoService) saveSegmentHash(
	slidePath,
	audioPath,
	outputPath string,
	width,
	height int,
	mediaAlignment string,
	effects []config.EffectConfig,
) error {
	hash, err := s.computeSegmentHash(slidePath, audioPath, width, height, mediaAlignment, effects)
	if err != nil {
		return err
	}

	hashPath := outputPath + ".hash"
	return afero.WriteFile(s.fs, hashPath, []byte(hash), 0644)
}

func normalizeMediaAlignment(alignment string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(alignment)) {
	case "", config.MediaAlignmentVideo:
		return config.MediaAlignmentVideo, nil
	case config.MediaAlignmentSlide:
		return config.MediaAlignmentSlide, nil
	default:
		return "", fmt.Errorf("unsupported media alignment %q (expected %q or %q)", alignment, config.MediaAlignmentVideo, config.MediaAlignmentSlide)
	}
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

// applyMultiViewLayouts applies multi-view layouts to video segments
func (s *VideoService) applyMultiViewLayouts(ctx context.Context, videoFiles []string, tempDir string, width, height int) error {
	if s.multiViewConfig == nil || !s.multiViewConfig.Enabled {
		return nil
	}

	s.logger.Info("Applying multi-view layouts", "layouts", len(s.multiViewConfig.Layouts))

	// Build a map of which slides have multi-view layouts
	multiViewMap := make(map[int]config.LayoutConfig)
	for _, layout := range s.multiViewConfig.Layouts {
		slideIndices := layout.ParseSlides(len(videoFiles))
		for _, idx := range slideIndices {
			multiViewMap[idx] = layout
			s.logger.Debug("Multi-view layout for slide", "slide", idx, "type", layout.Type)
		}
	}

	// Apply layouts
	var wg sync.WaitGroup
	errors := make([]error, len(videoFiles))

	for idx, layout := range multiViewMap {
		wg.Add(1)
		go func(slideIdx int, layoutCfg config.LayoutConfig) {
			defer wg.Done()

			// Generate multi-view video
			multiViewPath := filepath.Join(tempDir, fmt.Sprintf("multiview_%d.mp4", slideIdx))

			if err := s.multiViewService.GenerateMultiViewVideo(
				ctx,
				layoutCfg,
				multiViewPath,
				width,
				height,
			); err != nil {
				errors[slideIdx] = fmt.Errorf("failed to generate multi-view for slide %d: %w", slideIdx, err)
				return
			}

			// Replace the original video with the multi-view version
			videoFiles[slideIdx] = multiViewPath
			s.logger.Info("Multi-view applied", "slide", slideIdx, "type", layoutCfg.Type)
		}(idx, layout)
	}

	wg.Wait()

	// Check for errors
	for _, err := range errors {
		if err != nil {
			return err
		}
	}

	if len(multiViewMap) > 0 {
		s.logger.Info("Multi-view layouts applied successfully", "count", len(multiViewMap))
	}

	return nil
}
