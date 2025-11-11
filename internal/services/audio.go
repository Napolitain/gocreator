package services

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"path/filepath"
	"sync"

	"gocreator/internal/interfaces"

	"github.com/spf13/afero"
)

// AudioService handles audio generation
type AudioService struct {
	fs          afero.Fs
	client      interfaces.OpenAIClient
	textService *TextService
	logger      interfaces.Logger
}

// NewAudioService creates a new audio service
func NewAudioService(fs afero.Fs, client interfaces.OpenAIClient, textService *TextService, logger interfaces.Logger) *AudioService {
	return &AudioService{
		fs:          fs,
		client:      client,
		textService: textService,
		logger:      logger,
	}
}

// Generate generates audio from text
func (s *AudioService) Generate(ctx context.Context, text, outputPath string) error {
	// Check cache
	cached, err := s.checkCache(ctx, text, outputPath)
	if err != nil {
		return fmt.Errorf("failed to check cache: %w", err)
	}
	if cached {
		s.logger.Info("Using cached audio", "path", outputPath)
		return nil
	}

	// Generate audio
	body, err := s.client.GenerateSpeech(ctx, text)
	if err != nil {
		return fmt.Errorf("failed to generate speech: %w", err)
	}
	defer func() { _ = body.Close() }()

	// Ensure directory exists
	dir := filepath.Dir(outputPath)
	if err := s.fs.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to file
	file, err := s.fs.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create audio file: %w", err)
	}
	defer func() { _ = file.Close() }()

	if _, err := io.Copy(file, body); err != nil {
		return fmt.Errorf("failed to write audio: %w", err)
	}

	// Save hash for cache validation
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(text)))
	hashPath := outputPath + ".hash"
	if err := afero.WriteFile(s.fs, hashPath, []byte(hash), 0644); err != nil {
		return fmt.Errorf("failed to write hash file: %w", err)
	}

	return nil
}

// GenerateBatch generates audio for multiple texts in parallel
func (s *AudioService) GenerateBatch(ctx context.Context, texts []string, outputDir string) ([]string, error) {
	if err := s.fs.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Compute hashes and load cached hashes
	hashes := make([]string, len(texts))
	for i, text := range texts {
		hashes[i] = s.textService.Hash(text)
	}

	hashFile := filepath.Join(outputDir, "hashes")
	cachedHashes, err := s.textService.LoadHashes(ctx, hashFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load cached hashes: %w", err)
	}

	// Save current hashes
	if err := s.textService.SaveHashes(ctx, hashFile, hashes); err != nil {
		return nil, fmt.Errorf("failed to save hashes: %w", err)
	}

	// Generate audio files
	audioPaths := make([]string, len(texts))
	errors := make([]error, len(texts))
	var wg sync.WaitGroup

	for i, text := range texts {
		wg.Add(1)
		go func(idx int, txt, hash string) {
			defer wg.Done()

			audioPath := filepath.Join(outputDir, fmt.Sprintf("%d.mp3", idx))
			audioPaths[idx] = audioPath

			// Check if cached
			if idx < len(cachedHashes) && cachedHashes[idx] == hash {
				exists, err := afero.Exists(s.fs, audioPath)
				if err == nil && exists {
					return
				}
			}

			// Generate new audio
			if err := s.Generate(ctx, txt, audioPath); err != nil {
				errors[idx] = err
			}
		}(i, text, hashes[i])
	}

	wg.Wait()

	// Check for errors
	for i, err := range errors {
		if err != nil {
			return nil, fmt.Errorf("audio generation failed for text %d: %w", i, err)
		}
	}

	return audioPaths, nil
}

func (s *AudioService) checkCache(ctx context.Context, text, outputPath string) (bool, error) {
	exists, err := afero.Exists(s.fs, outputPath)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}

	// Check hash
	hashPath := outputPath + ".hash"
	hashExists, err := afero.Exists(s.fs, hashPath)
	if err != nil {
		return false, err
	}
	if !hashExists {
		return false, nil
	}

	// Read stored hash
	data, err := afero.ReadFile(s.fs, hashPath)
	if err != nil {
		return false, err
	}

	// Compute current hash
	currentHash := fmt.Sprintf("%x", sha256.Sum256([]byte(text)))
	return string(data) == currentHash, nil
}
