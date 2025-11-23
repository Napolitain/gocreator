package services

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"os/exec"
	"time"

	"gocreator/internal/config"
	"gocreator/internal/interfaces"

	"github.com/spf13/afero"
)

// EffectService handles visual effects
type EffectService struct {
	fs     afero.Fs
	logger interfaces.Logger
}

// NewEffectService creates a new effect service
func NewEffectService(fs afero.Fs, logger interfaces.Logger) *EffectService {
	return &EffectService{
		fs:     fs,
		logger: logger,
	}
}

// ApplyKenBurns applies Ken Burns (zoom and pan) effect to an image
func (s *EffectService) ApplyKenBurns(ctx context.Context, imagePath, outputPath string, duration float64, cfg config.EffectConfig) error {
	filter := s.BuildKenBurnsFilter(cfg, duration)

	args := []string{
		"-y",
		"-loop", "1",
		"-i", imagePath,
		"-vf", filter,
		"-t", fmt.Sprintf("%.2f", duration),
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	s.logger.Debug("Applying Ken Burns effect", "command", cmd.String())

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg error: %w, stderr: %s", err, stderr.String())
	}

	s.logger.Info("Ken Burns effect applied", "output", outputPath)
	return nil
}

// BuildKenBurnsFilter builds the zoompan filter for Ken Burns effect
func (s *EffectService) BuildKenBurnsFilter(cfg config.EffectConfig, duration float64) string {
	zoomStart := cfg.Config.ZoomStart
	if zoomStart <= 0 {
		zoomStart = 1.0
	}

	zoomEnd := cfg.Config.ZoomEnd
	if zoomEnd <= 0 {
		zoomEnd = 1.3
	}

	direction := cfg.Config.Direction
	if direction == "random" {
		directions := []string{"left", "right", "up", "down", "center"}
		rand.Seed(time.Now().UnixNano())
		direction = directions[rand.Intn(len(directions))]
	}

	// Calculate pan expressions
	xExpr, yExpr := s.getPanExpressions(direction)

	// Calculate zoom increment per frame
	frames := int(duration * 30) // 30 fps

	filter := fmt.Sprintf(
		"zoompan=z='if(lte(zoom,%.2f),zoom+%.5f,%.2f)':d=%d:x='%s':y='%s':s=1920x1080:fps=30",
		zoomEnd,
		(zoomEnd-zoomStart)/float64(frames),
		zoomEnd,
		frames,
		xExpr,
		yExpr,
	)

	return filter
}

func (s *EffectService) getPanExpressions(direction string) (string, string) {
	var xExpr, yExpr string

	switch direction {
	case "left":
		xExpr = "iw/2-(iw/zoom/2)-t*10"
		yExpr = "ih/2-(ih/zoom/2)"
	case "right":
		xExpr = "iw/2-(iw/zoom/2)+t*10"
		yExpr = "ih/2-(ih/zoom/2)"
	case "up":
		xExpr = "iw/2-(iw/zoom/2)"
		yExpr = "ih/2-(ih/zoom/2)-t*10"
	case "down":
		xExpr = "iw/2-(iw/zoom/2)"
		yExpr = "ih/2-(ih/zoom/2)+t*10"
	case "center":
		xExpr = "iw/2-(iw/zoom/2)"
		yExpr = "ih/2-(ih/zoom/2)"
	default:
		xExpr = "iw/2-(iw/zoom/2)"
		yExpr = "ih/2-(ih/zoom/2)"
	}

	return xExpr, yExpr
}

// BuildColorGradeFilter builds color grading filter
func (s *EffectService) BuildColorGradeFilter(cfg config.EffectConfig) string {
	parts := []string{}

	// Brightness, contrast, saturation
	if cfg.Config.Brightness != 0 || cfg.Config.Contrast != 1.0 || cfg.Config.Saturation != 1.0 {
		eq := fmt.Sprintf("eq=brightness=%.2f:contrast=%.2f:saturation=%.2f",
			cfg.Config.Brightness, cfg.Config.Contrast, cfg.Config.Saturation)
		parts = append(parts, eq)
	}

	// Hue
	if cfg.Config.Hue != 0 {
		parts = append(parts, fmt.Sprintf("hue=h=%d", cfg.Config.Hue))
	}

	// Gamma
	if cfg.Config.Gamma != 0 && cfg.Config.Gamma != 1.0 {
		parts = append(parts, fmt.Sprintf("eq=gamma=%.2f", cfg.Config.Gamma))
	}

	if len(parts) == 0 {
		return ""
	}

	// Join all filters with comma
	filter := ""
	for i, part := range parts {
		if i > 0 {
			filter += ","
		}
		filter += part
	}

	return filter
}

// BuildVignetteFilter builds vignette effect filter
func (s *EffectService) BuildVignetteFilter(cfg config.EffectConfig) string {
	intensity := cfg.Config.Intensity
	if intensity <= 0 {
		intensity = 0.3
	}

	// Vignette with angle parameter
	return fmt.Sprintf("vignette=angle=PI/%.1f", 4.0/intensity)
}

// BuildFilmGrainFilter builds film grain effect filter
func (s *EffectService) BuildFilmGrainFilter(cfg config.EffectConfig) string {
	intensity := cfg.Config.Intensity
	if intensity <= 0 {
		intensity = 0.3
	}

	// Noise filter for film grain
	strength := int(intensity * 50)
	return fmt.Sprintf("noise=alls=%d:allf=t+u", strength)
}

// BuildBlurBackgroundFilter builds blur background filter for portrait videos
func (s *EffectService) BuildBlurBackgroundFilter(cfg config.EffectConfig, targetWidth, targetHeight int) string {
	blurRadius := cfg.Config.BlurRadius
	if blurRadius <= 0 {
		blurRadius = 20
	}

	// Create blurred background and overlay original video
	filter := fmt.Sprintf(
		"[0:v]scale=%d:%d,boxblur=%d[bg];[0:v]scale=%d:-1[fg];[bg][fg]overlay=(W-w)/2:(H-h)/2",
		targetWidth, targetHeight, blurRadius, targetWidth,
	)

	return filter
}

// BuildStabilizationFilter builds video stabilization filter
func (s *EffectService) BuildStabilizationFilter(cfg config.EffectConfig) string {
	smoothing := cfg.Config.Smoothing
	if smoothing <= 0 {
		smoothing = 10
	}

	// Note: Stabilization requires two-pass processing
	// This returns the transform filter (second pass)
	return fmt.Sprintf("vidstabtransform=smoothing=%d", smoothing)
}
