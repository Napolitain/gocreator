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

// MultiViewService handles multi-view video layouts
type MultiViewService struct {
	fs     afero.Fs
	logger interfaces.Logger
}

// NewMultiViewService creates a new multi-view service
func NewMultiViewService(fs afero.Fs, logger interfaces.Logger) *MultiViewService {
	return &MultiViewService{
		fs:     fs,
		logger: logger,
	}
}

// GenerateMultiViewVideo generates a video with multi-view layout
func (s *MultiViewService) GenerateMultiViewVideo(
	ctx context.Context,
	layout config.LayoutConfig,
	outputPath string,
	outputWidth, outputHeight int,
) error {
	// Build filter complex
	filterComplex, err := s.BuildFilterComplex(layout, outputWidth, outputHeight)
	if err != nil {
		return fmt.Errorf("failed to build filter: %w", err)
	}

	// Build FFmpeg command
	args := []string{"-y"}

	// Add input files
	inputs := s.getInputFiles(layout)
	for _, input := range inputs {
		args = append(args, "-i", input)
	}

	// Add filter complex
	args = append(args, "-filter_complex", filterComplex, "-map", "[out]")

	// Add audio (mix all sources or use first)
	audioFilter := s.buildAudioFilter(len(inputs))
	if audioFilter != "" {
		args = append(args, "-filter_complex", audioFilter, "-map", "[aout]")
	} else {
		args = append(args, "-map", "0:a?") // Optional audio
	}

	// Encoding settings
	args = append(args,
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", "23",
		"-pix_fmt", "yuv420p",
		"-c:a", "aac",
		"-b:a", "192k",
		"-shortest", // End when shortest input ends
		outputPath,
	)

	// Execute FFmpeg
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	s.logger.Debug("Generating multi-view video", "command", cmd.String())

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg error: %w, stderr: %s", err, stderr.String())
	}

	s.logger.Info("Multi-view video generated", "output", outputPath)
	return nil
}

// BuildFilterComplex builds FFmpeg filter for multi-view layout
func (s *MultiViewService) BuildFilterComplex(layout config.LayoutConfig, outputWidth, outputHeight int) (string, error) {
	switch layout.Type {
	case "split-horizontal":
		return s.buildSplitHorizontal(layout, outputWidth, outputHeight)
	case "split-vertical":
		return s.buildSplitVertical(layout, outputWidth, outputHeight)
	case "pip":
		return s.buildPiP(layout, outputWidth, outputHeight)
	case "grid":
		return s.buildGrid(layout, outputWidth, outputHeight)
	case "focus-gallery":
		return s.buildFocusGallery(layout, outputWidth, outputHeight)
	case "custom":
		return s.buildCustom(layout, outputWidth, outputHeight)
	default:
		return "", fmt.Errorf("unknown layout type: %s", layout.Type)
	}
}

// buildSplitHorizontal creates side-by-side layout
func (s *MultiViewService) buildSplitHorizontal(layout config.LayoutConfig, w, h int) (string, error) {
	// Parse ratio (e.g., "50:50" or "60:40")
	leftRatio, _ := s.parseRatio(layout.Ratio)

	leftWidth := int(float64(w) * leftRatio)
	rightWidth := w - leftWidth - layout.Gap

	// Build filter complex
	filter := fmt.Sprintf(
		"[0:v]scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2[left];"+
			"[1:v]scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2[right];"+
			"[left][right]hstack=inputs=2:shortest=1[out]",
		leftWidth, h, leftWidth, h,
		rightWidth, h, rightWidth, h,
	)

	return filter, nil
}

// buildSplitVertical creates top-bottom layout
func (s *MultiViewService) buildSplitVertical(layout config.LayoutConfig, w, h int) (string, error) {
	topRatio, _ := s.parseRatio(layout.Ratio)

	topHeight := int(float64(h) * topRatio)
	bottomHeight := h - topHeight - layout.Gap

	filter := fmt.Sprintf(
		"[0:v]scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2[top];"+
			"[1:v]scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2[bottom];"+
			"[top][bottom]vstack=inputs=2:shortest=1[out]",
		w, topHeight, w, topHeight,
		w, bottomHeight, w, bottomHeight,
	)

	return filter, nil
}

// buildPiP creates picture-in-picture layout
func (s *MultiViewService) buildPiP(layout config.LayoutConfig, w, h int) (string, error) {
	// Parse size (e.g., "20%" or "320x180")
	pipWidth, pipHeight := s.parseSize(layout.Size, w, h)

	// Calculate position
	x, y := s.calculatePosition(layout.Position, layout.Offset, w, h, pipWidth, pipHeight)

	// Build filter
	filter := fmt.Sprintf(
		"[1:v]scale=%d:%d[pip];",
		pipWidth, pipHeight,
	)

	// Add border if specified
	if layout.Border.Width > 0 {
		color := layout.Border.Color
		if color == "" {
			color = "white"
		}
		filter += fmt.Sprintf(
			"[pip]drawbox=x=0:y=0:w=%d:h=%d:color=%s:t=%d[pip_bordered];",
			pipWidth, pipHeight, color, layout.Border.Width,
		)
		filter += fmt.Sprintf("[0:v][pip_bordered]overlay=%d:%d:shortest=1[out]", x, y)
	} else {
		filter += fmt.Sprintf("[0:v][pip]overlay=%d:%d:shortest=1[out]", x, y)
	}

	return filter, nil
}

// buildGrid creates grid layout (2x2, 3x3, etc.)
func (s *MultiViewService) buildGrid(layout config.LayoutConfig, w, h int) (string, error) {
	rows := layout.Rows
	cols := layout.Cols

	if rows <= 0 || cols <= 0 {
		return "", fmt.Errorf("invalid grid dimensions: %dx%d", rows, cols)
	}

	totalCells := rows * cols
	cellWidth := (w - (cols-1)*layout.Gap) / cols
	cellHeight := (h - (rows-1)*layout.Gap) / rows

	// Scale all inputs
	var filters []string
	for i := 0; i < totalCells; i++ {
		filters = append(filters, fmt.Sprintf(
			"[%d:v]scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2[v%d]",
			i, cellWidth, cellHeight, cellWidth, cellHeight, i,
		))
	}

	// Build grid layout string
	var layoutStr string
	for i := 0; i < totalCells; i++ {
		if i > 0 {
			layoutStr += "|"
		}
		col := i % cols
		row := i / cols
		x := col * (cellWidth + layout.Gap)
		y := row * (cellHeight + layout.Gap)
		layoutStr += fmt.Sprintf("%d_%d", x, y)
	}

	// Build input list for xstack
	var inputs string
	for i := 0; i < totalCells; i++ {
		inputs += fmt.Sprintf("[v%d]", i)
	}

	filter := fmt.Sprintf(
		"%s;%sxstack=inputs=%d:layout=%s:shortest=1[out]",
		strings.Join(filters, ";"),
		inputs,
		totalCells,
		layoutStr,
	)

	return filter, nil
}

// buildFocusGallery creates Zoom-style layout with main speaker and gallery
func (s *MultiViewService) buildFocusGallery(layout config.LayoutConfig, w, h int) (string, error) {
	gallerySize := s.parsePercentage(layout.GallerySize, 20)
	galleryCount := len(layout.Gallery)

	if galleryCount == 0 {
		return "", fmt.Errorf("focus-gallery requires at least one gallery video")
	}

	var mainWidth, mainHeight, galleryWidth, galleryHeight int
	var mainX, mainY, galleryX, galleryY int

	// Calculate dimensions based on gallery position
	switch layout.GalleryPosition {
	case "left":
		galleryWidth = int(float64(w) * gallerySize / 100.0)
		mainWidth = w - galleryWidth
		mainHeight = h
		galleryHeight = h
		mainX, mainY = galleryWidth, 0
		galleryX, galleryY = 0, 0

	case "top":
		galleryHeight = int(float64(h) * gallerySize / 100.0)
		mainHeight = h - galleryHeight
		mainWidth = w
		galleryWidth = w
		mainX, mainY = 0, galleryHeight
		galleryX, galleryY = 0, 0

	case "bottom":
		galleryHeight = int(float64(h) * gallerySize / 100.0)
		mainHeight = h - galleryHeight
		mainWidth = w
		galleryWidth = w
		mainX, mainY = 0, 0
		galleryX, galleryY = 0, mainHeight

	default: // "right"
		galleryWidth = int(float64(w) * gallerySize / 100.0)
		mainWidth = w - galleryWidth
		mainHeight = h
		galleryHeight = h
		mainX, mainY = 0, 0
		galleryX, galleryY = mainWidth, 0
	}

	galleryItemHeight := galleryHeight / galleryCount

	// Scale main video
	filter := fmt.Sprintf(
		"[0:v]scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2[main];",
		mainWidth, mainHeight, mainWidth, mainHeight,
	)

	// Scale gallery videos
	for i := 0; i < galleryCount; i++ {
		filter += fmt.Sprintf(
			"[%d:v]scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2[g%d];",
			i+1, galleryWidth, galleryItemHeight, galleryWidth, galleryItemHeight, i,
		)
	}

	// Stack gallery videos
	galleryInputs := ""
	for i := 0; i < galleryCount; i++ {
		galleryInputs += fmt.Sprintf("[g%d]", i)
	}
	filter += fmt.Sprintf("%svstack=inputs=%d:shortest=1[gallery];", galleryInputs, galleryCount)

	// Combine main and gallery
	filter += fmt.Sprintf("[main][gallery]overlay=%d:%d:shortest=1[out]", galleryX-mainX, galleryY-mainY)

	return filter, nil
}

// buildCustom creates custom positioned layout
func (s *MultiViewService) buildCustom(layout config.LayoutConfig, w, h int) (string, error) {
	if len(layout.CustomVideos) == 0 {
		return "", fmt.Errorf("custom layout requires at least one video")
	}

	// Sort by z-index
	videos := make([]config.CustomVideoConfig, len(layout.CustomVideos))
	copy(videos, layout.CustomVideos)

	// Build filter for each video
	var filters []string
	for i, video := range videos {
		filter := fmt.Sprintf(
			"[%d:v]scale=%d:%d[cv%d]",
			i, video.Size[0], video.Size[1], i,
		)
		filters = append(filters, filter)
	}

	// Overlay videos in order
	overlayFilter := "[0:v]"
	for i, video := range videos {
		if i == 0 {
			overlayFilter = "[cv0]"
		} else {
			x, y := video.Position[0], video.Position[1]
			overlayFilter = fmt.Sprintf("%s[cv%d]overlay=%d:%d", overlayFilter, i, x, y)
			if i < len(videos)-1 {
				overlayFilter = fmt.Sprintf("%s[tmp%d];[tmp%d]", overlayFilter, i, i)
			}
		}
	}
	overlayFilter += "[out]"

	filter := strings.Join(filters, ";") + ";" + overlayFilter
	return filter, nil
}

// Helper functions

func (s *MultiViewService) getInputFiles(layout config.LayoutConfig) []string {
	var inputs []string

	switch layout.Type {
	case "split-horizontal":
		inputs = []string{layout.Videos.Left, layout.Videos.Right}
	case "split-vertical":
		inputs = []string{layout.Videos.Top, layout.Videos.Bottom}
	case "pip":
		inputs = []string{layout.Main, layout.Overlay}
	case "grid":
		inputs = layout.GridVideos
	case "focus-gallery":
		inputs = append([]string{layout.Focus}, layout.Gallery...)
	case "custom":
		for _, v := range layout.CustomVideos {
			inputs = append(inputs, v.Source)
		}
	}

	return inputs
}

func (s *MultiViewService) buildAudioFilter(inputCount int) string {
	if inputCount <= 1 {
		return ""
	}

	// Mix all audio sources
	var inputs []string
	for i := 0; i < inputCount; i++ {
		inputs = append(inputs, fmt.Sprintf("[%d:a]", i))
	}

	return fmt.Sprintf("%samix=inputs=%d:duration=shortest[aout]", strings.Join(inputs, ""), inputCount)
}

func (s *MultiViewService) parseRatio(ratio string) (float64, float64) {
	if ratio == "" {
		return 0.5, 0.5
	}

	parts := strings.Split(ratio, ":")
	if len(parts) != 2 {
		return 0.5, 0.5
	}

	left, err1 := strconv.ParseFloat(parts[0], 64)
	right, err2 := strconv.ParseFloat(parts[1], 64)

	if err1 != nil || err2 != nil {
		return 0.5, 0.5
	}

	total := left + right
	return left / total, right / total
}

func (s *MultiViewService) parseSize(size string, refW, refH int) (int, int) {
	if size == "" {
		return refW / 5, refH / 5 // Default 20%
	}

	if strings.HasSuffix(size, "%") {
		pct, err := strconv.ParseFloat(strings.TrimSuffix(size, "%"), 64)
		if err != nil {
			return refW / 5, refH / 5
		}
		w := int(float64(refW) * pct / 100.0)
		h := int(float64(refH) * pct / 100.0)
		return w, h
	}

	parts := strings.Split(size, "x")
	if len(parts) == 2 {
		w, err1 := strconv.Atoi(parts[0])
		h, err2 := strconv.Atoi(parts[1])
		if err1 == nil && err2 == nil {
			return w, h
		}
	}

	return refW / 5, refH / 5
}

func (s *MultiViewService) calculatePosition(position string, offset []int, w, h, pipW, pipH int) (int, int) {
	offsetX, offsetY := 10, 10
	if len(offset) >= 2 {
		offsetX, offsetY = offset[0], offset[1]
	}

	switch position {
	case "top-left":
		return offsetX, offsetY
	case "top-right":
		return w - pipW - offsetX, offsetY
	case "bottom-left":
		return offsetX, h - pipH - offsetY
	case "bottom-right":
		return w - pipW - offsetX, h - pipH - offsetY
	case "center":
		return (w - pipW) / 2, (h - pipH) / 2
	default:
		return w - pipW - offsetX, h - pipH - offsetY
	}
}

func (s *MultiViewService) parsePercentage(str string, defaultVal float64) float64 {
	if str == "" {
		return defaultVal
	}

	val, err := strconv.ParseFloat(strings.TrimSuffix(str, "%"), 64)
	if err != nil {
		return defaultVal
	}
	return val
}
