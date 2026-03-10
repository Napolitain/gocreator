package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"gocreator/internal/config"
)

func (s *VideoService) buildSingleVideoArgsWithEffects(input videoRenderInput) ([]string, error) {
	args := []string{"-y"}

	if input.isVideo {
		if input.alignToSlide {
			args = append(args, "-stream_loop", "-1")
		}
		args = append(args, "-i", input.slidePath, "-i", input.audioPath)
	} else {
		args = append(args, "-loop", "1", "-i", input.slidePath, "-i", input.audioPath)
	}

	filterSegments, videoMap, err := s.buildVideoEffectsGraph(input)
	if err != nil {
		return nil, err
	}

	audioMap := "1:a:0"
	if input.isVideo && input.hasEmbeddedAudio {
		durationMode := "first"
		if input.alignToSlide {
			durationMode = "shortest"
		}
		filterSegments = append(filterSegments, fmt.Sprintf("[0:a:0][1:a:0]amix=inputs=2:duration=%s:dropout_transition=0[a]", durationMode))
		audioMap = "[a]"
	}

	if len(filterSegments) > 0 {
		args = append(args, "-filter_complex", strings.Join(filterSegments, ";"))
	}

	args = append(args, "-map", videoMap, "-map", audioMap, "-c:v", "libx264")
	if !input.isVideo {
		args = append(args, "-tune", "stillimage")
	}

	args = append(args, "-c:a", "mp3", "-b:a", "192k", "-pix_fmt", "yuv420p")
	if input.isVideo {
		args = append(args, "-t", fmt.Sprintf("%.2f", input.segmentDuration()))
	} else {
		args = append(args, "-shortest")
	}

	args = append(args, input.outputPath)
	return args, nil
}

func (s *VideoService) buildVideoEffectsGraph(input videoRenderInput) ([]string, string, error) {
	filters := make([]string, 0, 6)
	postFilters := make([]string, 0, len(input.effects))
	currentInput := "[0:v]"
	videoMap := "0:v:0"
	labelIndex := 0
	nextLabel := func(prefix string) string {
		label := fmt.Sprintf("%s%d", prefix, labelIndex)
		labelIndex++
		return label
	}
	appendSimpleChain := func(chain string) {
		if strings.TrimSpace(chain) == "" {
			return
		}
		label := nextLabel("vfx")
		filters = append(filters, fmt.Sprintf("%s%s[%s]", currentInput, chain, label))
		currentInput = fmt.Sprintf("[%s]", label)
		videoMap = currentInput
	}

	var geometryEffect *config.EffectConfig

	for _, effect := range input.effects {
		switch strings.ToLower(strings.TrimSpace(effect.Type)) {
		case "stabilize":
			if input.stabilizationTransformPath == "" {
				return nil, "", fmt.Errorf("stabilization transform path is required for slide %s", filepath.Base(input.slidePath))
			}
			appendSimpleChain(s.effectService.BuildStabilizationFilterWithInput(effect, escapeFFmpegFilterPath(input.stabilizationTransformPath)))
		case "ken-burns", "blur-background":
			if geometryEffect == nil {
				effectCopy := effect
				geometryEffect = &effectCopy
			}
		case "color-grade":
			postFilters = append(postFilters, s.effectService.BuildColorGradeFilter(effect))
		case "vignette":
			postFilters = append(postFilters, s.effectService.BuildVignetteFilter(effect))
		case "film-grain":
			postFilters = append(postFilters, s.effectService.BuildFilmGrainFilter(effect))
		case "text-overlay":
			postFilters = append(postFilters, s.overlayService.BuildTextOverlayFilterWithDuration(effect, input.segmentDuration()))
		default:
			return nil, "", fmt.Errorf("unsupported effect type %q for slide %s", effect.Type, filepath.Base(input.slidePath))
		}
	}

	switch {
	case geometryEffect != nil && strings.EqualFold(geometryEffect.Type, "ken-burns"):
		appendSimpleChain(s.effectService.BuildKenBurnsFilterForOutput(*geometryEffect, input.segmentDuration(), input.targetWidth, input.targetHeight))
	case geometryEffect != nil && strings.EqualFold(geometryEffect.Type, "blur-background"):
		blurRadius := geometryEffect.Config.BlurRadius
		if blurRadius <= 0 {
			blurRadius = 20
		}
		bgLabel := nextLabel("bg")
		fgLabel := nextLabel("fg")
		outLabel := nextLabel("vfx")
		filters = append(filters,
			fmt.Sprintf("%sscale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d,boxblur=%d[%s]", currentInput, input.targetWidth, input.targetHeight, input.targetWidth, input.targetHeight, blurRadius, bgLabel),
			fmt.Sprintf("%sscale=%d:%d:force_original_aspect_ratio=decrease,setsar=1[%s]", currentInput, input.targetWidth, input.targetHeight, fgLabel),
			fmt.Sprintf("[%s][%s]overlay=(W-w)/2:(H-h)/2[%s]", bgLabel, fgLabel, outLabel),
		)
		currentInput = fmt.Sprintf("[%s]", outLabel)
		videoMap = currentInput
	case input.targetWidth != input.inputWidth || input.targetHeight != input.inputHeight:
		appendSimpleChain(fmt.Sprintf(
			"scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2,setsar=1",
			input.targetWidth, input.targetHeight, input.targetWidth, input.targetHeight,
		))
	}

	appendSimpleChain(strings.Join(compactFilterParts(postFilters), ","))

	return filters, videoMap, nil
}

func (s *VideoService) resolveEffectsForSlides(slides []string) map[int][]config.EffectConfig {
	if len(s.effects) == 0 {
		return nil
	}

	effectsBySlide := make(map[int][]config.EffectConfig)
	for _, effect := range s.effects {
		for _, index := range effect.ParseSlides(len(slides)) {
			if index < 0 || index >= len(slides) {
				continue
			}
			effectsBySlide[index] = append(effectsBySlide[index], effect)
		}
	}

	return effectsBySlide
}

func (s *VideoService) resolveEffectsForSlide(slidePath string, effects []config.EffectConfig) []config.EffectConfig {
	if len(effects) == 0 {
		return nil
	}

	resolved := make([]config.EffectConfig, len(effects))
	for i, effect := range effects {
		resolved[i] = effect
		if !strings.EqualFold(effect.Type, "ken-burns") || !strings.EqualFold(effect.Config.Direction, "random") {
			continue
		}

		payload, err := json.Marshal(struct {
			SlidePath string
			Config    config.EffectDetails
		}{
			SlidePath: slidePath,
			Config:    effect.Config,
		})
		if err != nil {
			resolved[i].Config.Direction = "center"
			continue
		}

		directions := []string{"left", "right", "up", "down", "center"}
		hash := sha256.Sum256(payload)
		resolved[i].Config.Direction = directions[int(hash[0])%len(directions)]
	}

	return resolved
}

func (s *VideoService) validateEffectsForSlide(slidePath string, isVideo bool, effects []config.EffectConfig) error {
	if len(effects) == 0 {
		return nil
	}

	var geometryEffect string
	stabilizeCount := 0

	for _, effect := range effects {
		effectType := strings.ToLower(strings.TrimSpace(effect.Type))

		switch effectType {
		case "ken-burns":
			if isVideo {
				return fmt.Errorf("effect ken-burns only supports still-image slides: %s", filepath.Base(slidePath))
			}
			if geometryEffect != "" {
				return fmt.Errorf("cannot combine %s with ken-burns on slide %s", geometryEffect, filepath.Base(slidePath))
			}
			geometryEffect = effectType
		case "blur-background":
			if geometryEffect != "" {
				return fmt.Errorf("cannot combine %s with blur-background on slide %s", geometryEffect, filepath.Base(slidePath))
			}
			geometryEffect = effectType
		case "stabilize":
			if !isVideo {
				return fmt.Errorf("effect stabilize only supports video slides: %s", filepath.Base(slidePath))
			}
			stabilizeCount++
			if stabilizeCount > 1 {
				return fmt.Errorf("multiple stabilize effects configured for slide %s", filepath.Base(slidePath))
			}
		case "color-grade", "vignette", "film-grain", "text-overlay":
			continue
		default:
			return fmt.Errorf("unsupported effect type %q for slide %s", effect.Type, filepath.Base(slidePath))
		}
	}

	return nil
}

func (s *VideoService) findSlideEffect(effects []config.EffectConfig, effectType string) (config.EffectConfig, bool) {
	for _, effect := range effects {
		if strings.EqualFold(strings.TrimSpace(effect.Type), effectType) {
			return effect, true
		}
	}

	return config.EffectConfig{}, false
}

func serializeEffectsForCache(effects []config.EffectConfig) (string, error) {
	type cachedEffect struct {
		Type   string               `json:"type"`
		Config config.EffectDetails `json:"config"`
	}

	normalized := make([]cachedEffect, len(effects))
	for i, effect := range effects {
		normalized[i] = cachedEffect{
			Type:   effect.Type,
			Config: effect.Config,
		}
	}

	data, err := json.Marshal(normalized)
	if err != nil {
		return "", fmt.Errorf("failed to serialize effects for cache: %w", err)
	}

	return string(data), nil
}

func (s *VideoService) runStabilizationDetect(
	ctx context.Context,
	slidePath string,
	effect config.EffectConfig,
	transformPath string,
) error {
	filter := fmt.Sprintf("vidstabdetect=shakiness=5:accuracy=15:result='%s'", escapeFFmpegFilterPath(transformPath))
	args := []string{
		"-y",
		"-i", slidePath,
		"-vf", filter,
		"-an",
		"-f", "null",
		"-",
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	s.logger.Debug("Running stabilization detection",
		"slide", slidePath,
		"smoothing", effect.Config.Smoothing,
		"command", cmd.String(),
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg stabilization detection failed: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

func escapeFFmpegFilterPath(path string) string {
	normalized := filepath.ToSlash(path)
	replacer := strings.NewReplacer(":", "\\:", "'", "\\'")
	return replacer.Replace(normalized)
}

func compactFilterParts(filters []string) []string {
	compacted := make([]string, 0, len(filters))
	for _, filter := range filters {
		if strings.TrimSpace(filter) == "" {
			continue
		}
		compacted = append(compacted, filter)
	}
	return compacted
}

func (input videoRenderInput) segmentDuration() float64 {
	if input.isVideo {
		if input.alignToSlide {
			return input.audioDuration
		}
		return input.videoDuration
	}

	return input.audioDuration
}
