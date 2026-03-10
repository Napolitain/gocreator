package services

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"gocreator/internal/config"
	"gocreator/internal/interfaces"

	"github.com/spf13/afero"
)

// PostProcessService applies optional post-render features such as subtitles,
// audio mixing, intro/outro clips, metadata, chapters, thumbnails, and exports.
type PostProcessService struct {
	fs              afero.Fs
	logger          interfaces.Logger
	commandExecutor interfaces.CommandExecutor
	audioMixer      *AudioMixer
	subtitleService *SubtitleService
	exportService   *ExportService
}

// PostProcessRequest describes the artifacts and configuration for one language render.
type PostProcessRequest struct {
	RootDir        string
	OutputDir      string
	BaseName       string
	Lang           string
	MasterVideo    string
	Slides         []string
	Texts          []string
	AudioPaths     []string
	MediaAlignment string
	Output         config.OutputConfig
	Encoding       config.EncodingConfig
	Audio          config.AudioConfig
	Subtitles      config.SubtitlesConfig
	Intro          config.IntroConfig
	Outro          config.OutroConfig
	Metadata       config.MetadataConfig
	Chapters       config.ChaptersConfig
}

// PostProcessResult summarizes the emitted artifacts for a language render.
type PostProcessResult struct {
	PrimaryOutputPath string
	ExportedPaths     []string
	SubtitlePaths     []string
	ThumbnailPath     string
}

type edgeClipConfig struct {
	Name               string
	Video              string
	Transition         string
	TransitionDuration float64
	Template           config.TemplateConfig
}

// NewPostProcessService creates a new post-processing service.
func NewPostProcessService(fs afero.Fs, logger interfaces.Logger) *PostProcessService {
	return NewPostProcessServiceWithExecutor(fs, logger, nil)
}

// NewPostProcessServiceWithExecutor creates a new post-processing service with an injected executor.
func NewPostProcessServiceWithExecutor(fs afero.Fs, logger interfaces.Logger, executor interfaces.CommandExecutor) *PostProcessService {
	if executor == nil {
		executor = newCommandExecutor()
	}

	return &PostProcessService{
		fs:              fs,
		logger:          logger,
		commandExecutor: executor,
		audioMixer:      NewAudioMixerWithExecutor(fs, logger, executor),
		subtitleService: NewSubtitleServiceWithExecutor(fs, logger, executor),
		exportService:   NewExportServiceWithExecutor(fs, logger, executor),
	}
}

// Run applies all configured post-processing features for one language output.
func (s *PostProcessService) Run(ctx context.Context, req PostProcessRequest) (PostProcessResult, error) {
	if err := s.fs.MkdirAll(req.OutputDir, 0755); err != nil {
		return PostProcessResult{}, fmt.Errorf("failed to create output directory: %w", err)
	}

	tempDir := filepath.Join(req.OutputDir, ".temp")
	if err := s.fs.MkdirAll(tempDir, 0755); err != nil {
		return PostProcessResult{}, fmt.Errorf("failed to create postprocess temp directory: %w", err)
	}

	tempFiles := make([]string, 0, 8)
	defer func() {
		for _, tempPath := range tempFiles {
			_ = s.fs.Remove(tempPath)
		}
	}()

	audioDurations, segmentDurations, err := s.computeTimelineDurations(ctx, req.Slides, req.AudioPaths, req.MediaAlignment)
	if err != nil {
		return PostProcessResult{}, err
	}
	slideStarts := cumulativeDurations(segmentDurations)

	workingVideo := req.MasterVideo
	introDuration := 0.0

	if req.Intro.Enabled {
		workingVideo, introDuration, err = s.applyEdgeClip(ctx, req.RootDir, tempDir, workingVideo, introClip(req.Intro), true, &tempFiles)
		if err != nil {
			return PostProcessResult{}, err
		}
	}
	if req.Outro.Enabled {
		workingVideo, _, err = s.applyEdgeClip(ctx, req.RootDir, tempDir, workingVideo, outroClip(req.Outro), false, &tempFiles)
		if err != nil {
			return PostProcessResult{}, err
		}
	}

	workingVideo, err = s.applyAudioPostProcessing(ctx, req, tempDir, workingVideo, slideStarts, introDuration, &tempFiles)
	if err != nil {
		return PostProcessResult{}, err
	}

	subtitlePaths, subtitleSRT, err := s.generateSubtitles(req, audioDurations, introDuration)
	if err != nil {
		return PostProcessResult{}, err
	}
	if req.Subtitles.Enabled && req.Subtitles.BurnIn && subtitleSRT != "" {
		burnedPath := filepath.Join(tempDir, req.BaseName+".burned.mp4")
		tempFiles = append(tempFiles, burnedPath)
		if err := s.subtitleService.BurnSubtitles(ctx, workingVideo, subtitleSRT, burnedPath, req.Subtitles); err != nil {
			return PostProcessResult{}, err
		}
		workingVideo = burnedPath
	}

	chapters := buildMetadataChapters(req.Chapters, segmentDurations, introDuration)
	metadata := req.Metadata
	if metadata.Language == "" {
		metadata.Language = req.Lang
	}

	primaryFormat := primaryExportFormat(req.Output)
	primaryOutputPath := filepath.Join(req.OutputDir, req.BaseName+"."+formatExtension(primaryFormat.Type))
	if err := s.materializeOutput(ctx, workingVideo, primaryOutputPath, primaryFormat, req.Encoding, req.Output.Quality, metadata, chapters, tempDir, &tempFiles); err != nil {
		return PostProcessResult{}, err
	}

	exported := []string{primaryOutputPath}
	for index, format := range req.Output.Formats {
		formatType := normalizeFormatType(format.Type)
		if formatType == "" {
			continue
		}
		outputPath := filepath.Join(req.OutputDir, variantFileName(req.BaseName, format, index))
		if outputPath == primaryOutputPath {
			continue
		}
		if err := s.materializeOutput(ctx, workingVideo, outputPath, format, req.Encoding, req.Output.Quality, metadata, chapters, tempDir, &tempFiles); err != nil {
			return PostProcessResult{}, err
		}
		exported = append(exported, outputPath)
	}

	thumbnailPath, err := s.generateThumbnail(ctx, req, primaryOutputPath, tempDir, &tempFiles)
	if err != nil {
		return PostProcessResult{}, err
	}

	return PostProcessResult{
		PrimaryOutputPath: primaryOutputPath,
		ExportedPaths:     exported,
		SubtitlePaths:     subtitlePaths,
		ThumbnailPath:     thumbnailPath,
	}, nil
}

func (s *PostProcessService) applyAudioPostProcessing(
	ctx context.Context,
	req PostProcessRequest,
	tempDir string,
	workingVideo string,
	slideStarts []float64,
	introDuration float64,
	tempFiles *[]string,
) (string, error) {
	backgroundMusicEnabled := req.Audio.BackgroundMusic.Enabled && strings.TrimSpace(req.Audio.BackgroundMusic.File) != ""
	if backgroundMusicEnabled {
		musicPath := s.resolveAssetPath(req.RootDir, req.Audio.BackgroundMusic.File)
		exists, err := afero.Exists(s.fs, musicPath)
		if err != nil {
			return "", fmt.Errorf("failed to check background music file: %w", err)
		}
		if !exists {
			return "", fmt.Errorf("background music file not found: %s", musicPath)
		}

		nextPath := filepath.Join(tempDir, req.BaseName+".music.mp4")
		*tempFiles = append(*tempFiles, nextPath)

		if req.Audio.Ducking.Enabled {
			if err := s.audioMixer.ApplyDucking(ctx, workingVideo, musicPath, nextPath, req.Audio.BackgroundMusic, req.Audio.Ducking); err != nil {
				return "", err
			}
		} else {
			if err := s.audioMixer.MixBackgroundMusic(ctx, workingVideo, musicPath, nextPath, req.Audio.BackgroundMusic); err != nil {
				return "", err
			}
		}

		workingVideo = nextPath
	}

	for index, effect := range req.Audio.SoundEffects {
		if effect.Slide < 0 || effect.Slide >= len(slideStarts) {
			return "", fmt.Errorf("sound effect %d targets invalid slide index %d", index, effect.Slide)
		}
		effectPath := s.resolveAssetPath(req.RootDir, effect.File)
		exists, err := afero.Exists(s.fs, effectPath)
		if err != nil {
			return "", fmt.Errorf("failed to check sound effect file: %w", err)
		}
		if !exists {
			return "", fmt.Errorf("sound effect file not found: %s", effectPath)
		}

		volume := effect.Volume
		if volume <= 0 {
			volume = 1.0
		}
		delay := introDuration + slideStarts[effect.Slide] + effect.Delay

		nextPath := filepath.Join(tempDir, fmt.Sprintf("%s.sfx-%d.mp4", req.BaseName, index))
		*tempFiles = append(*tempFiles, nextPath)
		if err := s.audioMixer.AddSoundEffect(ctx, workingVideo, effectPath, nextPath, delay, volume); err != nil {
			return "", err
		}
		workingVideo = nextPath
	}

	return workingVideo, nil
}

func (s *PostProcessService) generateSubtitles(req PostProcessRequest, audioDurations []float64, introOffset float64) ([]string, string, error) {
	if !req.Subtitles.Enabled || !subtitleLanguageEnabled(req.Subtitles, req.Lang) {
		return nil, "", nil
	}

	segments := make([]SubtitleSegment, 0, len(req.Texts))
	currentTime := introOffset
	for index, text := range req.Texts {
		duration := audioDurations[index]
		if duration < 0 {
			duration = 0
		}
		if strings.TrimSpace(text) != "" && duration > 0 {
			segments = append(segments, SubtitleSegment{
				Index:     len(segments) + 1,
				StartTime: currentTime,
				EndTime:   currentTime + duration,
				Text:      prepareSubtitleText(s.subtitleService, text, req.Subtitles.Timing),
			})
		}
		currentTime += duration
	}

	if len(segments) == 0 {
		return nil, "", nil
	}

	srtPath := filepath.Join(req.OutputDir, req.BaseName+".srt")
	vttPath := filepath.Join(req.OutputDir, req.BaseName+".vtt")
	if err := s.subtitleService.GenerateSRT(segments, srtPath); err != nil {
		return nil, "", err
	}
	if err := s.subtitleService.GenerateVTT(segments, vttPath); err != nil {
		return nil, "", err
	}
	return []string{srtPath, vttPath}, srtPath, nil
}

func (s *PostProcessService) generateThumbnail(
	ctx context.Context,
	req PostProcessRequest,
	primaryOutputPath string,
	tempDir string,
	tempFiles *[]string,
) (string, error) {
	thumbnail := req.Metadata.Thumbnail
	if !thumbnail.Enabled {
		return "", nil
	}

	ext := ".jpg"
	if thumbnail.Source == "custom" && thumbnail.CustomFile != "" {
		ext = filepath.Ext(thumbnail.CustomFile)
		if ext == "" {
			ext = ".jpg"
		}
	}

	outputPath := filepath.Join(req.OutputDir, req.BaseName+"-thumbnail"+ext)
	switch strings.ToLower(strings.TrimSpace(thumbnail.Source)) {
	case "custom":
		sourcePath := s.resolveAssetPath(req.RootDir, thumbnail.CustomFile)
		if err := copyFileWithinFS(s.fs, sourcePath, outputPath); err != nil {
			return "", err
		}
	case "slide":
		slideIndex := thumbnail.SlideIndex
		if slideIndex < 0 || slideIndex >= len(req.Slides) {
			slideIndex = 0
		}
		slidePath := req.Slides[slideIndex]
		isVideo, err := s.isVideoFile(ctx, slidePath)
		if err != nil {
			return "", err
		}
		if isVideo {
			if err := s.exportService.GenerateThumbnail(ctx, slidePath, outputPath, thumbnail.FrameTime); err != nil {
				return "", err
			}
		} else {
			if err := copyFileWithinFS(s.fs, slidePath, outputPath); err != nil {
				return "", err
			}
		}
	default:
		if err := s.exportService.GenerateThumbnail(ctx, primaryOutputPath, outputPath, thumbnail.FrameTime); err != nil {
			return "", err
		}
	}

	if strings.TrimSpace(thumbnail.OverlayText) != "" {
		overlaid := filepath.Join(tempDir, req.BaseName+".thumbnail-overlay"+ext)
		*tempFiles = append(*tempFiles, overlaid)
		if err := s.overlayThumbnailText(ctx, outputPath, overlaid, thumbnail.OverlayText); err != nil {
			return "", err
		}
		if err := moveOrCopyWithinFS(s.fs, overlaid, outputPath); err != nil {
			return "", err
		}
	}

	return outputPath, nil
}

func (s *PostProcessService) materializeOutput(
	ctx context.Context,
	inputPath string,
	outputPath string,
	format config.FormatConfig,
	baseEncoding config.EncodingConfig,
	defaultQuality string,
	metadata config.MetadataConfig,
	chapters []MetadataChapter,
	tempDir string,
	tempFiles *[]string,
) error {
	normalized := normalizeExportFormat(format, defaultQuality)
	currentPath := inputPath

	if needsFormatExport(normalized, baseEncoding) {
		exportPath := filepath.Join(tempDir, shortHash(outputPath)+"-export."+formatExtension(normalized.Type))
		*tempFiles = append(*tempFiles, exportPath)
		if err := s.exportService.ExportToFormat(ctx, inputPath, exportPath, normalized, baseEncoding); err != nil {
			return err
		}
		currentPath = exportPath
	}

	if normalizeFormatType(normalized.Type) != "gif" && (!isEmptyMetadata(metadata) || len(chapters) > 0) {
		metadataPath := filepath.Join(tempDir, shortHash(outputPath)+"-metadata."+formatExtension(normalized.Type))
		*tempFiles = append(*tempFiles, metadataPath)
		if err := s.exportService.ApplyMetadata(ctx, currentPath, metadataPath, metadata, chapters); err != nil {
			return err
		}
		currentPath = metadataPath
	}

	if currentPath == outputPath {
		return nil
	}
	return moveOrCopyWithinFS(s.fs, currentPath, outputPath)
}

func (s *PostProcessService) computeTimelineDurations(ctx context.Context, slides, audioPaths []string, mediaAlignment string) ([]float64, []float64, error) {
	audioDurations := make([]float64, len(audioPaths))
	segmentDurations := make([]float64, len(slides))

	for index := range slides {
		audioDuration, err := s.getDuration(ctx, audioPaths[index])
		if err != nil {
			return nil, nil, fmt.Errorf("failed to inspect narration duration for slide %d: %w", index, err)
		}
		audioDurations[index] = audioDuration

		isVideo, err := s.isVideoFile(ctx, slides[index])
		if err != nil {
			return nil, nil, fmt.Errorf("failed to inspect slide %d: %w", index, err)
		}
		if !isVideo || strings.EqualFold(mediaAlignment, config.MediaAlignmentSlide) {
			segmentDurations[index] = audioDuration
			continue
		}

		videoDuration, err := s.getDuration(ctx, slides[index])
		if err != nil {
			return nil, nil, fmt.Errorf("failed to inspect video duration for slide %d: %w", index, err)
		}
		segmentDurations[index] = videoDuration
	}

	return audioDurations, segmentDurations, nil
}

func (s *PostProcessService) applyEdgeClip(
	ctx context.Context,
	rootDir string,
	tempDir string,
	workingVideo string,
	clip edgeClipConfig,
	prepend bool,
	tempFiles *[]string,
) (string, float64, error) {
	clipPath, duration, cleanupPath, err := s.resolveEdgeClip(ctx, rootDir, tempDir, workingVideo, clip)
	if err != nil {
		return "", 0, err
	}
	if cleanupPath != "" {
		*tempFiles = append(*tempFiles, cleanupPath)
	}

	withAudioPath, err := s.ensureAudioTrack(ctx, tempDir, clipPath, clip.Name, tempFiles)
	if err != nil {
		return "", 0, err
	}

	transition := edgeClipTransition(clip)
	outputPath := filepath.Join(tempDir, fmt.Sprintf("%s.%s-joined.mp4", shortHash(workingVideo+clip.Name), clip.Name))
	*tempFiles = append(*tempFiles, outputPath)

	var first, second string
	if prepend {
		first, second = withAudioPath, workingVideo
	} else {
		first, second = workingVideo, withAudioPath
	}

	if err := s.concatenatePair(ctx, first, second, transition, outputPath); err != nil {
		return "", 0, err
	}
	return outputPath, duration, nil
}

func (s *PostProcessService) resolveEdgeClip(
	ctx context.Context,
	rootDir string,
	tempDir string,
	workingVideo string,
	clip edgeClipConfig,
) (string, float64, string, error) {
	if strings.TrimSpace(clip.Video) != "" {
		clipPath := s.resolveAssetPath(rootDir, clip.Video)
		exists, err := afero.Exists(s.fs, clipPath)
		if err != nil {
			return "", 0, "", fmt.Errorf("failed to check %s clip: %w", clip.Name, err)
		}
		if !exists {
			return "", 0, "", fmt.Errorf("%s clip not found: %s", clip.Name, clipPath)
		}
		duration, err := s.getDuration(ctx, clipPath)
		if err != nil {
			return "", 0, "", fmt.Errorf("failed to inspect %s clip duration: %w", clip.Name, err)
		}
		return clipPath, duration, "", nil
	}

	if !clip.Template.Enabled {
		return "", 0, "", fmt.Errorf("%s is enabled but no video or template is configured", clip.Name)
	}

	width, height, err := s.getVideoDimensions(ctx, workingVideo)
	if err != nil {
		return "", 0, "", fmt.Errorf("failed to inspect output dimensions for %s template: %w", clip.Name, err)
	}

	templatePath := filepath.Join(tempDir, fmt.Sprintf("%s.%s-template.mp4", shortHash(workingVideo+clip.Name), clip.Name))
	if err := s.generateTemplateClip(ctx, rootDir, templatePath, width, height, clip.Template); err != nil {
		return "", 0, "", err
	}

	duration := clip.Template.Duration
	if duration <= 0 {
		duration = 3
	}
	return templatePath, duration, templatePath, nil
}

func (s *PostProcessService) concatenatePair(ctx context.Context, firstPath, secondPath string, transition TransitionConfig, outputPath string) error {
	args := []string{"-y", "-i", firstPath, "-i", secondPath}

	if !transition.IsEnabled() {
		args = append(args,
			"-filter_complex", "[0:v][0:a][1:v][1:a]concat=n=2:v=1:a=1[outv][outa]",
			"-map", "[outv]",
			"-map", "[outa]",
			outputPath,
		)
		return s.runFFmpeg(ctx, args)
	}

	firstDuration, err := s.getDuration(ctx, firstPath)
	if err != nil {
		return fmt.Errorf("failed to inspect intro/outro duration: %w", err)
	}
	offset := firstDuration - transition.Duration
	if offset < 0 {
		offset = 0
	}

	filter := fmt.Sprintf(
		"[0:v][1:v]xfade=transition=%s:duration=%.2f:offset=%.2f[outv];[0:a][1:a]concat=n=2:v=0:a=1[outa]",
		transition.GetFFmpegTransitionName(),
		transition.Duration,
		offset,
	)
	args = append(args,
		"-filter_complex", filter,
		"-map", "[outv]",
		"-map", "[outa]",
		outputPath,
	)
	return s.runFFmpeg(ctx, args)
}

func (s *PostProcessService) ensureAudioTrack(ctx context.Context, tempDir, inputPath, label string, tempFiles *[]string) (string, error) {
	hasAudio, err := s.hasAudioStream(ctx, inputPath)
	if err != nil {
		return "", err
	}
	if hasAudio {
		return inputPath, nil
	}

	duration, err := s.getDuration(ctx, inputPath)
	if err != nil {
		return "", err
	}

	outputPath := filepath.Join(tempDir, fmt.Sprintf("%s.%s-silence.mp4", shortHash(inputPath+label), label))
	*tempFiles = append(*tempFiles, outputPath)
	args := []string{
		"-y",
		"-i", inputPath,
		"-f", "lavfi",
		"-t", fmt.Sprintf("%.2f", duration),
		"-i", "anullsrc=r=48000:cl=stereo",
		"-map", "0:v:0",
		"-map", "1:a:0",
		"-c:v", "copy",
		"-c:a", "aac",
		"-shortest",
		outputPath,
	}
	if err := s.runFFmpeg(ctx, args); err != nil {
		return "", err
	}

	return outputPath, nil
}

func (s *PostProcessService) generateTemplateClip(ctx context.Context, rootDir, outputPath string, width, height int, template config.TemplateConfig) error {
	duration := template.Duration
	if duration <= 0 {
		duration = 3
	}

	background := template.BackgroundColor
	if strings.TrimSpace(background) == "" {
		background = "black"
	}
	textColor := template.TextColor
	if strings.TrimSpace(textColor) == "" {
		textColor = "white"
	}

	args := []string{
		"-y",
		"-f", "lavfi",
		"-i", fmt.Sprintf("color=c=%s:s=%dx%d:d=%.2f", background, width, height, duration),
	}
	audioInputIndex := 1

	filters := make([]string, 0, 3)
	currentInput := "[0:v]"
	lastLabel := currentInput
	filterIndex := 0
	nextLabel := func() string {
		label := fmt.Sprintf("tpl%d", filterIndex)
		filterIndex++
		return label
	}

	appendDrawtext := func(text string, fontSize int, yExpr string) {
		if strings.TrimSpace(text) == "" {
			return
		}
		if fontSize <= 0 {
			fontSize = 48
		}
		label := nextLabel()
		filters = append(filters, fmt.Sprintf(
			"%sdrawtext=text='%s':fontcolor=%s:fontsize=%d:x=(w-text_w)/2:y=%s[%s]",
			lastLabel,
			escapeDrawTextText(text),
			textColor,
			fontSize,
			yExpr,
			label,
		))
		lastLabel = "[" + label + "]"
	}

	appendDrawtext(template.Text, 52, "(h/2)-70")
	appendDrawtext(template.Subtext, 28, "(h/2)+10")

	if strings.TrimSpace(template.Logo) != "" {
		logoPath := s.resolveAssetPath(rootDir, template.Logo)
		args = append(args, "-loop", "1", "-i", logoPath)
		audioInputIndex = 2
		label := nextLabel()
		scaleWidth := width / 6
		if scaleWidth < 120 {
			scaleWidth = 120
		}
		filters = append(filters, fmt.Sprintf("[1:v]scale=%d:-1[logo]", scaleWidth))
		filters = append(filters, fmt.Sprintf("%s[logo]overlay=(W-w)/2:(H-h)-120[%s]", lastLabel, label))
		lastLabel = "[" + label + "]"
	}

	args = append(args,
		"-f", "lavfi",
		"-i", "anullsrc=r=48000:cl=stereo",
	)

	if len(filters) > 0 {
		args = append(args, "-filter_complex", strings.Join(filters, ";"), "-map", lastLabel)
	} else {
		args = append(args, "-map", "0:v:0")
	}

	args = append(args,
		"-map", fmt.Sprintf("%d:a:0", audioInputIndex),
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-c:a", "aac",
		"-shortest",
		outputPath,
	)

	return s.runFFmpeg(ctx, args)
}

func (s *PostProcessService) overlayThumbnailText(ctx context.Context, inputPath, outputPath, text string) error {
	args := []string{
		"-y",
		"-i", inputPath,
		"-vf", fmt.Sprintf("drawtext=text='%s':fontcolor=white:fontsize=36:borderw=2:bordercolor=black:x=(w-text_w)/2:y=h-th-40", escapeDrawTextText(text)),
		outputPath,
	}
	return s.runFFmpeg(ctx, args)
}

func (s *PostProcessService) runFFmpeg(ctx context.Context, args []string) error {
	s.logger.Debug("Running postprocess ffmpeg", "command", formatCommand("ffmpeg", args...))
	result, err := s.commandExecutor.Run(ctx, "ffmpeg", args...)
	if err != nil {
		return fmt.Errorf("ffmpeg error: %w, stderr: %s", err, string(result.Stderr))
	}
	return nil
}

func (s *PostProcessService) getDuration(ctx context.Context, mediaPath string) (float64, error) {
	result, err := s.commandExecutor.Run(ctx, "ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		mediaPath,
	)
	if err != nil {
		return 0, fmt.Errorf("ffprobe duration check failed: %w", err)
	}

	var duration float64
	if _, err := fmt.Sscanf(strings.TrimSpace(string(result.Stdout)), "%f", &duration); err != nil {
		return 0, fmt.Errorf("failed to parse duration: %w", err)
	}
	return duration, nil
}

func (s *PostProcessService) isVideoFile(ctx context.Context, mediaPath string) (bool, error) {
	result, err := s.commandExecutor.Run(ctx, "ffprobe",
		"-v", "error",
		"-show_entries", "stream=codec_type:format=duration",
		"-of", "default=noprint_wrappers=1",
		mediaPath,
	)
	if err != nil {
		return false, fmt.Errorf("ffprobe check failed: %w", err)
	}

	output := strings.TrimSpace(string(result.CombinedOutput()))
	if !strings.Contains(output, "codec_type=video") {
		return false, nil
	}

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "duration=") {
			continue
		}
		durationValue := strings.TrimPrefix(line, "duration=")
		duration, parseErr := strconv.ParseFloat(durationValue, 64)
		if parseErr == nil && duration <= 0 {
			return false, nil
		}
	}

	return true, nil
}

func (s *PostProcessService) hasAudioStream(ctx context.Context, mediaPath string) (bool, error) {
	result, err := s.commandExecutor.Run(ctx, "ffprobe",
		"-v", "error",
		"-show_entries", "stream=codec_type",
		"-of", "default=noprint_wrappers=1:nokey=1",
		mediaPath,
	)
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

func (s *PostProcessService) getVideoDimensions(ctx context.Context, mediaPath string) (int, int, error) {
	result, err := s.commandExecutor.Run(ctx, "ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height",
		"-of", "csv=s=x:p=0",
		mediaPath,
	)
	if err != nil {
		return 0, 0, fmt.Errorf("ffprobe dimension check failed: %w", err)
	}

	parts := strings.Split(strings.TrimSpace(string(result.Stdout)), "x")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("failed to parse dimensions from ffprobe output")
	}
	width, err1 := strconv.Atoi(parts[0])
	height, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return 0, 0, fmt.Errorf("failed to parse dimensions from ffprobe output")
	}
	return width, height, nil
}

func (s *PostProcessService) resolveAssetPath(rootDir, configuredPath string) string {
	if filepath.IsAbs(configuredPath) {
		return filepath.Clean(configuredPath)
	}
	return filepath.Clean(filepath.Join(rootDir, configuredPath))
}

func primaryExportFormat(output config.OutputConfig) config.FormatConfig {
	formatType := normalizeFormatType(output.Format)
	if formatType == "" {
		formatType = "mp4"
	}
	return config.FormatConfig{
		Type:    formatType,
		Quality: output.Quality,
	}
}

func normalizeExportFormat(format config.FormatConfig, defaultQuality string) config.FormatConfig {
	normalized := format
	normalized.Type = normalizeFormatType(normalized.Type)
	if normalized.Type == "" {
		normalized.Type = "mp4"
	}
	if normalized.Quality == "" {
		normalized.Quality = defaultQuality
	}
	return normalized
}

func normalizeFormatType(formatType string) string {
	switch strings.ToLower(strings.TrimSpace(formatType)) {
	case "", "mp4":
		return "mp4"
	case "webm":
		return "webm"
	case "gif":
		return "gif"
	default:
		return strings.ToLower(strings.TrimSpace(formatType))
	}
}

func formatExtension(formatType string) string {
	switch normalizeFormatType(formatType) {
	case "webm":
		return "webm"
	case "gif":
		return "gif"
	default:
		return "mp4"
	}
}

func needsFormatExport(format config.FormatConfig, baseEncoding config.EncodingConfig) bool {
	normalizedType := normalizeFormatType(format.Type)
	if normalizedType != "mp4" {
		return true
	}
	if format.Resolution != "" || format.Codec != "" || format.FPS > 0 {
		return true
	}
	if format.Quality != "" && format.Quality != "medium" {
		return true
	}
	return !reflect.DeepEqual(baseEncoding, config.EncodingConfig{}) && !reflect.DeepEqual(baseEncoding, config.DefaultEncodingConfig())
}

func variantFileName(baseName string, format config.FormatConfig, index int) string {
	typeName := normalizeFormatType(format.Type)
	if typeName == "" {
		typeName = "mp4"
	}
	suffixParts := []string{typeName}
	if format.Resolution != "" {
		suffixParts = append(suffixParts, strings.ReplaceAll(format.Resolution, "x", "x"))
	}
	if format.Quality != "" {
		suffixParts = append(suffixParts, format.Quality)
	}
	if format.FPS > 0 {
		suffixParts = append(suffixParts, fmt.Sprintf("%dfps", format.FPS))
	}
	return fmt.Sprintf("%s-%s.%s", baseName, strings.Join(suffixParts, "-"), formatExtension(typeName))
}

func subtitleLanguageEnabled(cfg config.SubtitlesConfig, lang string) bool {
	switch value := cfg.Languages.(type) {
	case nil:
		return true
	case string:
		normalized := strings.TrimSpace(strings.ToLower(value))
		return normalized == "" || normalized == "all" || normalized == strings.ToLower(lang)
	case []string:
		for _, candidate := range value {
			if strings.EqualFold(candidate, lang) {
				return true
			}
		}
	case []interface{}:
		for _, candidate := range value {
			if text, ok := candidate.(string); ok && strings.EqualFold(text, lang) {
				return true
			}
		}
	}
	return false
}

func prepareSubtitleText(service *SubtitleService, text string, timing config.SubtitleTimingConfig) string {
	lines := service.SplitTextIntoLines(strings.TrimSpace(text), timing.MaxCharsPerLine)
	if timing.MaxLines > 0 && len(lines) > timing.MaxLines {
		trimmed := append([]string{}, lines[:timing.MaxLines-1]...)
		trimmed = append(trimmed, strings.Join(lines[timing.MaxLines-1:], " "))
		lines = trimmed
	}
	return strings.Join(lines, "\n")
}

func introClip(cfg config.IntroConfig) edgeClipConfig {
	return edgeClipConfig{
		Name:               "intro",
		Video:              cfg.Video,
		Transition:         cfg.Transition,
		TransitionDuration: cfg.TransitionDuration,
		Template:           cfg.Template,
	}
}

func outroClip(cfg config.OutroConfig) edgeClipConfig {
	return edgeClipConfig{
		Name:               "outro",
		Video:              cfg.Video,
		Transition:         cfg.Transition,
		TransitionDuration: cfg.TransitionDuration,
		Template:           cfg.Template,
	}
}

func edgeClipTransition(cfg edgeClipConfig) TransitionConfig {
	transition := TransitionConfig{
		Type:     TransitionType(strings.TrimSpace(cfg.Transition)),
		Duration: cfg.TransitionDuration,
	}
	if err := transition.Validate(); err != nil {
		return TransitionConfig{Type: TransitionNone}
	}
	return transition
}

func buildMetadataChapters(cfg config.ChaptersConfig, segmentDurations []float64, introOffset float64) []MetadataChapter {
	if !cfg.Enabled || len(cfg.Markers) == 0 {
		return nil
	}

	slideStarts := cumulativeDurations(segmentDurations)
	markers := make([]config.ChapterMarker, 0, len(cfg.Markers))
	for _, marker := range cfg.Markers {
		if marker.Slide >= 0 && marker.Slide < len(slideStarts) && strings.TrimSpace(marker.Title) != "" {
			markers = append(markers, marker)
		}
	}
	sort.Slice(markers, func(i, j int) bool { return markers[i].Slide < markers[j].Slide })

	totalDuration := introOffset
	for _, duration := range segmentDurations {
		totalDuration += duration
	}

	chapters := make([]MetadataChapter, 0, len(markers))
	for index, marker := range markers {
		start := introOffset + slideStarts[marker.Slide]
		end := totalDuration
		if index+1 < len(markers) {
			end = introOffset + slideStarts[markers[index+1].Slide]
		}
		if end <= start {
			continue
		}
		chapters = append(chapters, MetadataChapter{
			StartTime: start,
			EndTime:   end,
			Title:     marker.Title,
		})
	}
	return chapters
}

func cumulativeDurations(durations []float64) []float64 {
	offsets := make([]float64, len(durations))
	total := 0.0
	for index, duration := range durations {
		offsets[index] = total
		total += duration
	}
	return offsets
}

func moveOrCopyWithinFS(fs afero.Fs, sourcePath, targetPath string) error {
	if sourcePath == targetPath {
		return nil
	}
	_ = fs.Remove(targetPath)
	if err := fs.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}
	if err := fs.Rename(sourcePath, targetPath); err == nil {
		return nil
	}
	if err := copyFileWithinFS(fs, sourcePath, targetPath); err != nil {
		return err
	}
	if err := fs.Remove(sourcePath); err != nil {
		return fmt.Errorf("failed to remove original file %s after copy: %w", sourcePath, err)
	}
	return nil
}

func escapeDrawTextText(text string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"'", "\\'",
		":", "\\:",
		"%", "\\%",
		"\n", "\\n",
	)
	return replacer.Replace(text)
}
