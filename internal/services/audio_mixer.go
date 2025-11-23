package services

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"gocreator/internal/config"
	"gocreator/internal/interfaces"

	"github.com/spf13/afero"
)

// AudioMixer handles background music and audio mixing
type AudioMixer struct {
	fs     afero.Fs
	logger interfaces.Logger
}

// NewAudioMixer creates a new audio mixer
func NewAudioMixer(fs afero.Fs, logger interfaces.Logger) *AudioMixer {
	return &AudioMixer{
		fs:     fs,
		logger: logger,
	}
}

// MixBackgroundMusic mixes background music with video audio
func (s *AudioMixer) MixBackgroundMusic(ctx context.Context, videoPath, musicPath, outputPath string, cfg config.BackgroundMusicConfig) error {
	if !cfg.Enabled || musicPath == "" {
		s.logger.Debug("Background music disabled or no music file specified")
		return nil
	}

	// Get video duration
	duration, err := s.getVideoDuration(videoPath)
	if err != nil {
		return fmt.Errorf("failed to get video duration: %w", err)
	}

	// Build filter complex for music mixing
	filterComplex := s.buildMusicFilter(cfg, duration)

	args := []string{
		"-y",
		"-i", videoPath,
		"-i", musicPath,
		"-filter_complex", filterComplex,
		"-map", "0:v", "-map", "[a]",
		"-c:v", "copy", // Copy video stream (no re-encode)
		"-c:a", "aac", "-b:a", "192k",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	s.logger.Debug("Mixing background music", "command", cmd.String())

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg error: %w, stderr: %s", err, stderr.String())
	}

	s.logger.Info("Background music mixed successfully", "output", outputPath)
	return nil
}

func (s *AudioMixer) buildMusicFilter(cfg config.BackgroundMusicConfig, videoDuration float64) string {
	volume := cfg.Volume
	if volume <= 0 {
		volume = 0.15 // default
	}

	fadeIn := cfg.FadeIn
	if fadeIn < 0 {
		fadeIn = 0
	}

	fadeOut := cfg.FadeOut
	if fadeOut < 0 {
		fadeOut = 0
	}

	fadeOutStart := videoDuration - fadeOut
	if fadeOutStart < 0 {
		fadeOutStart = 0
	}

	// Build filter chain for music track
	musicFilter := fmt.Sprintf("[1:a]volume=%.2f", volume)

	if fadeIn > 0 {
		musicFilter += fmt.Sprintf(",afade=t=in:st=0:d=%.2f", fadeIn)
	}

	if fadeOut > 0 {
		musicFilter += fmt.Sprintf(",afade=t=out:st=%.2f:d=%.2f", fadeOutStart, fadeOut)
	}

	if cfg.Loop {
		// Loop the music indefinitely
		musicFilter += ",aloop=loop=-1:size=2e+09"
	}

	musicFilter += "[music]"

	// Mix original audio with music
	filterComplex := musicFilter + ";[0:a][music]amix=inputs=2:duration=first:dropout_transition=2[a]"

	return filterComplex
}

// AddSoundEffect adds a sound effect at a specific time
func (s *AudioMixer) AddSoundEffect(ctx context.Context, videoPath, effectPath, outputPath string, delay, volume float64) error {
	if effectPath == "" {
		return fmt.Errorf("sound effect path is empty")
	}

	filterComplex := fmt.Sprintf(
		"[1:a]adelay=%d|%d,volume=%.2f[sfx];[0:a][sfx]amix=inputs=2:duration=first[a]",
		int(delay*1000), int(delay*1000), volume,
	)

	args := []string{
		"-y",
		"-i", videoPath,
		"-i", effectPath,
		"-filter_complex", filterComplex,
		"-map", "0:v", "-map", "[a]",
		"-c:v", "copy",
		"-c:a", "aac", "-b:a", "192k",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	s.logger.Debug("Adding sound effect", "command", cmd.String())

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg error: %w, stderr: %s", err, stderr.String())
	}

	s.logger.Info("Sound effect added successfully", "output", outputPath)
	return nil
}

// ApplyDucking applies audio ducking (reduces music volume during speech)
func (s *AudioMixer) ApplyDucking(ctx context.Context, videoPath, musicPath, outputPath string, cfg config.DuckingConfig) error {
	if !cfg.Enabled {
		return nil
	}

	// Side-chain compression filter
	filterComplex := fmt.Sprintf(
		"[1:a]volume=0.2[music];[0:a]asplit=2[speech][sc];[music][sc]sidechaincompress=threshold=%.2f:ratio=%.2f:attack=%.2f:release=%.2f[compressed];[speech][compressed]amix=inputs=2:duration=first[a]",
		cfg.Threshold/100.0, 1.0/cfg.Ratio, cfg.Attack, cfg.Release,
	)

	args := []string{
		"-y",
		"-i", videoPath,
		"-i", musicPath,
		"-filter_complex", filterComplex,
		"-map", "0:v", "-map", "[a]",
		"-c:v", "copy",
		"-c:a", "aac", "-b:a", "192k",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	s.logger.Debug("Applying audio ducking", "command", cmd.String())

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg error: %w, stderr: %s", err, stderr.String())
	}

	s.logger.Info("Audio ducking applied successfully", "output", outputPath)
	return nil
}

func (s *AudioMixer) getVideoDuration(videoPath string) (float64, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		videoPath)

	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe error: %w", err)
	}

	var duration float64
	if _, err := fmt.Sscanf(string(output), "%f", &duration); err != nil {
		return 0, fmt.Errorf("failed to parse duration: %w", err)
	}

	return duration, nil
}
