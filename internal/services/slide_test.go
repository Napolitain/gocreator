package services

import (
	"context"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestNewSlideService(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewSlideService(fs, logger)
	
	assert.NotNil(t, service)
	assert.Equal(t, fs, service.fs)
	assert.Equal(t, logger, service.logger)
}

func TestSlideService_LoadSlides(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(afero.Fs)
		dir            string
		expectedCount  int
		expectedError  bool
	}{
		{
			name: "load slides from directory with mixed files",
			setupFunc: func(fs afero.Fs) {
				_ = afero.WriteFile(fs, "/slides/slide1.png", []byte("png data"), 0644)
				_ = afero.WriteFile(fs, "/slides/slide2.jpg", []byte("jpg data"), 0644)
				_ = afero.WriteFile(fs, "/slides/slide3.jpeg", []byte("jpeg data"), 0644)
				_ = afero.WriteFile(fs, "/slides/video.mp4", []byte("mp4 data"), 0644)
				_ = afero.WriteFile(fs, "/slides/ignored.txt", []byte("text data"), 0644)
			},
			dir:           "/slides",
			expectedCount: 4,
			expectedError: false,
		},
		{
			name: "empty directory returns empty slice",
			setupFunc: func(fs afero.Fs) {
				_ = fs.MkdirAll("/empty", 0755)
			},
			dir:           "/empty",
			expectedCount: 0,
			expectedError: false,
		},
		{
			name: "non-existent directory creates it and returns empty slice",
			setupFunc: func(fs afero.Fs) {
				// Do nothing, directory doesn't exist
			},
			dir:           "/nonexistent",
			expectedCount: 0,
			expectedError: false,
		},
		{
			name: "directory with subdirectories ignores them",
			setupFunc: func(fs afero.Fs) {
				_ = afero.WriteFile(fs, "/slides/slide1.png", []byte("png data"), 0644)
				_ = fs.MkdirAll("/slides/subdir", 0755)
				_ = afero.WriteFile(fs, "/slides/subdir/slide2.png", []byte("png data"), 0644)
			},
			dir:           "/slides",
			expectedCount: 1,
			expectedError: false,
		},
		{
			name: "supports video formats",
			setupFunc: func(fs afero.Fs) {
				_ = afero.WriteFile(fs, "/videos/clip1.mp4", []byte("mp4"), 0644)
				_ = afero.WriteFile(fs, "/videos/clip2.mov", []byte("mov"), 0644)
				_ = afero.WriteFile(fs, "/videos/clip3.avi", []byte("avi"), 0644)
				_ = afero.WriteFile(fs, "/videos/clip4.mkv", []byte("mkv"), 0644)
				_ = afero.WriteFile(fs, "/videos/clip5.webm", []byte("webm"), 0644)
			},
			dir:           "/videos",
			expectedCount: 5,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			logger := &mockLogger{}
			service := NewSlideService(fs, logger)
			
			tt.setupFunc(fs)
			
			ctx := context.Background()
			slides, err := service.LoadSlides(ctx, tt.dir)
			
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, slides, tt.expectedCount)
			}
		})
	}
}

func TestSlideService_LoadFromGoogleSlides(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewSlideService(fs, logger)
	
	ctx := context.Background()
	slides, notes, err := service.LoadFromGoogleSlides(ctx, "presentation-id", "/output")
	
	assert.Error(t, err)
	assert.Nil(t, slides)
	assert.Nil(t, notes)
	assert.Contains(t, err.Error(), "not implemented")
}
