package services

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSlideService(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewSlideService(fs, logger)

	assert.NotNil(t, service)
	assert.Equal(t, fs, service.fs)
	assert.Equal(t, logger, service.logger)
	assert.NotNil(t, service.runCommand)
}

func TestSlideService_LoadSlides(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(afero.Fs)
		dir           string
		expectedCount int
		expectedError bool
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
			name:          "non-existent directory creates it and returns empty slice",
			setupFunc:     func(fs afero.Fs) {},
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

			slides, err := service.LoadSlides(context.Background(), tt.dir)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Len(t, slides, tt.expectedCount)
		})
	}
}

func TestSlideService_LoadSlides_UsesNaturalOrdering(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewSlideService(fs, logger)

	require.NoError(t, afero.WriteFile(fs, "/slides/slide10.png", []byte("png"), 0644))
	require.NoError(t, afero.WriteFile(fs, "/slides/slide2.png", []byte("png"), 0644))
	require.NoError(t, afero.WriteFile(fs, "/slides/slide1.png", []byte("png"), 0644))

	slides, err := service.LoadSlides(context.Background(), "/slides")
	require.NoError(t, err)

	assert.Equal(t, []string{
		filepath.Join("/slides", "slide1.png"),
		filepath.Join("/slides", "slide2.png"),
		filepath.Join("/slides", "slide10.png"),
	}, slides)
}

func TestSlideService_LoadSlides_ExpandsPDFsAndReusesCache(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewSlideService(fs, logger)

	require.NoError(t, afero.WriteFile(fs, "/data/slides/01-intro.png", []byte("png"), 0644))
	require.NoError(t, afero.WriteFile(fs, "/data/slides/02-handout.pdf", []byte("pdf contents"), 0644))
	require.NoError(t, afero.WriteFile(fs, "/data/slides/03-outro.mp4", []byte("video"), 0644))
	sourceHash, err := service.hashFile("/data/slides/02-handout.pdf")
	require.NoError(t, err)
	cacheDir := filepath.Join("/data/cache/pdf", fmt.Sprintf("02-handout-%s", sourceHash[:12]))

	callCount := 0
	service.runCommand = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		callCount++

		switch name {
		case "pdfinfo":
			return []byte("Pages: 2\nEncrypted: no\n"), nil
		case "pdfseparate":
			require.Len(t, args, 2)
			for page := 1; page <= 2; page++ {
				outputPath := fmt.Sprintf(args[1], page)
				require.NoError(t, afero.WriteFile(fs, outputPath, []byte(fmt.Sprintf("page %d", page)), 0644))
			}
			return []byte(""), nil
		case "pdftocairo":
			require.Len(t, args, 4)
			renderedPath := args[3] + ".png"
			require.NoError(t, afero.WriteFile(fs, renderedPath, []byte("png"), 0644))
			return []byte(""), nil
		default:
			return nil, fmt.Errorf("unexpected command: %s", name)
		}
	}

	slides, err := service.LoadSlides(context.Background(), "/data/slides")
	require.NoError(t, err)
	assert.Equal(t, []string{
		filepath.Join("/data/slides", "01-intro.png"),
		filepath.Join(cacheDir, "02-handout-page-0001.png"),
		filepath.Join(cacheDir, "02-handout-page-0002.png"),
		filepath.Join("/data/slides", "03-outro.mp4"),
	}, slides)
	assert.Equal(t, 4, callCount)

	callCount = 0
	slides, err = service.LoadSlides(context.Background(), "/data/slides")
	require.NoError(t, err)
	assert.Equal(t, []string{
		filepath.Join("/data/slides", "01-intro.png"),
		filepath.Join(cacheDir, "02-handout-page-0001.png"),
		filepath.Join(cacheDir, "02-handout-page-0002.png"),
		filepath.Join("/data/slides", "03-outro.mp4"),
	}, slides)
	assert.Equal(t, 0, callCount, "cached PDF artifacts should be reused without rerunning commands")

	manifestPath := filepath.Join(cacheDir, "manifest.json")
	exists, err := afero.Exists(fs, manifestPath)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestSlideService_LoadSlides_FailsWhenPDFToolsAreMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewSlideService(fs, logger)

	require.NoError(t, afero.WriteFile(fs, "/data/slides/secret.pdf", []byte("encrypted"), 0644))
	service.runCommand = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return nil, &exec.Error{Name: name, Err: exec.ErrNotFound}
	}

	_, err := service.LoadSlides(context.Background(), "/data/slides")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PDF input requires pdfinfo")
}

func TestSlideService_LoadSlides_FailsForProtectedPDFMetadata(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewSlideService(fs, logger)

	require.NoError(t, afero.WriteFile(fs, "/data/slides/secret.pdf", []byte("encrypted"), 0644))
	service.runCommand = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if name == "pdfinfo" {
			return []byte("Pages: 3\nEncrypted: yes (print:no copy:no change:no)\n"), nil
		}
		return nil, fmt.Errorf("unexpected command: %s", name)
	}

	_, err := service.LoadSlides(context.Background(), "/data/slides")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "encrypted PDFs are not supported")
}

func TestParsePDFInfoOutput(t *testing.T) {
	pageCount, encrypted, err := parsePDFInfoOutput([]byte("Title: demo\nPages: 12\nEncrypted: no\n"))
	require.NoError(t, err)
	assert.Equal(t, 12, pageCount)
	assert.False(t, encrypted)
}
