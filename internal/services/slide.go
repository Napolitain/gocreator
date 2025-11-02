package services

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"gocreator/internal/interfaces"

	"github.com/spf13/afero"
)

// SlideService handles slide loading
type SlideService struct {
	fs     afero.Fs
	logger interfaces.Logger
}

// NewSlideService creates a new slide service
func NewSlideService(fs afero.Fs, logger interfaces.Logger) *SlideService {
	return &SlideService{
		fs:     fs,
		logger: logger,
	}
}

// LoadSlides loads slide images from a directory
func (s *SlideService) LoadSlides(ctx context.Context, dir string) ([]string, error) {
	exists, err := afero.DirExists(s.fs, dir)
	if err != nil {
		return nil, fmt.Errorf("failed to check directory: %w", err)
	}
	if !exists {
		if err := s.fs.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}
		return []string{}, nil
	}

	files, err := afero.ReadDir(s.fs, dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var slides []string
	validExts := map[string]bool{
		".png":  true,
		".jpg":  true,
		".jpeg": true,
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(file.Name()))
		if validExts[ext] {
			slides = append(slides, filepath.Join(dir, file.Name()))
		}
	}

	return slides, nil
}
