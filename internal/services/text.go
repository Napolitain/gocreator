package services

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"

	"gocreator/internal/interfaces"

	"github.com/spf13/afero"
)

// TextService handles text operations
type TextService struct {
	fs     afero.Fs
	logger interfaces.Logger
}

// NewTextService creates a new text service
func NewTextService(fs afero.Fs, logger interfaces.Logger) *TextService {
	return &TextService{
		fs:     fs,
		logger: logger,
	}
}

// Load loads texts from a file, splitting by "-" delimiter
func (s *TextService) Load(ctx context.Context, path string) ([]string, error) {
	file, err := s.fs.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var texts []string
	var current strings.Builder
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "-" {
			if current.Len() > 0 {
				texts = append(texts, strings.TrimSpace(current.String()))
				current.Reset()
			}
		} else {
			if current.Len() > 0 {
				current.WriteString("\n")
			}
			current.WriteString(line)
		}
	}

	if current.Len() > 0 {
		texts = append(texts, strings.TrimSpace(current.String()))
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return texts, nil
}

// Save saves texts to a file with "-" delimiter
func (s *TextService) Save(ctx context.Context, path string, texts []string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := s.fs.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := s.fs.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	for i, text := range texts {
		if _, err := file.WriteString(text); err != nil {
			return fmt.Errorf("failed to write text: %w", err)
		}
		if i < len(texts)-1 {
			if _, err := file.WriteString("\n-\n"); err != nil {
				return fmt.Errorf("failed to write delimiter: %w", err)
			}
		}
	}

	return nil
}

// Hash computes a SHA256 hash of text
func (s *TextService) Hash(text string) string {
	hash := sha256.Sum256([]byte(text))
	return hex.EncodeToString(hash[:])
}

// SaveHashes saves hashes to a file
func (s *TextService) SaveHashes(ctx context.Context, path string, hashes []string) error {
	dir := filepath.Dir(path)
	if err := s.fs.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := s.fs.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create hash file: %w", err)
	}
	defer file.Close()

	for _, hash := range hashes {
		if _, err := file.WriteString(hash + "\n"); err != nil {
			return fmt.Errorf("failed to write hash: %w", err)
		}
	}

	return nil
}

// LoadHashes loads hashes from a file
func (s *TextService) LoadHashes(ctx context.Context, path string) ([]string, error) {
	exists, err := afero.Exists(s.fs, path)
	if err != nil {
		return nil, fmt.Errorf("failed to check if file exists: %w", err)
	}
	if !exists {
		return []string{}, nil
	}

	file, err := s.fs.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open hash file: %w", err)
	}
	defer file.Close()

	var hashes []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		hashes = append(hashes, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading hash file: %w", err)
	}

	return hashes, nil
}
