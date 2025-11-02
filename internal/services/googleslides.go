package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"gocreator/internal/interfaces"

	"github.com/spf13/afero"
	"google.golang.org/api/option"
	"google.golang.org/api/slides/v1"
)

// GoogleSlidesService handles fetching slides and notes from Google Slides
type GoogleSlidesService struct {
	fs     afero.Fs
	logger interfaces.Logger
}

// NewGoogleSlidesService creates a new Google Slides service
func NewGoogleSlidesService(fs afero.Fs, logger interfaces.Logger) *GoogleSlidesService {
	return &GoogleSlidesService{
		fs:     fs,
		logger: logger,
	}
}

// LoadFromGoogleSlides fetches slides as images and their speaker notes from a Google Slides presentation
func (s *GoogleSlidesService) LoadFromGoogleSlides(ctx context.Context, presentationID, outputDir string) ([]string, []string, error) {
	// Create Google Slides service
	slidesService, err := s.createSlidesService(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create slides service: %w", err)
	}

	// Get presentation
	presentation, err := slidesService.Presentations.Get(presentationID).Do()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get presentation: %w", err)
	}

	s.logger.Info("Fetched presentation", "title", presentation.Title, "slideCount", len(presentation.Slides))

	// Create output directory
	if err := s.fs.MkdirAll(outputDir, 0755); err != nil {
		return nil, nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	var slidePaths []string
	var notes []string

	// Process each slide
	for i, slide := range presentation.Slides {
		s.logger.Debug("Processing slide", "index", i, "objectId", slide.ObjectId)

		// Get slide thumbnail using the API
		thumbnail, err := slidesService.Presentations.Pages.GetThumbnail(presentationID, slide.ObjectId).Do()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get thumbnail for slide %d: %w", i, err)
		}

		// Download slide image from thumbnail URL
		slidePath := filepath.Join(outputDir, fmt.Sprintf("slide_%d.png", i))
		if err := s.downloadImage(ctx, thumbnail.ContentUrl, slidePath); err != nil {
			return nil, nil, fmt.Errorf("failed to download slide %d: %w", i, err)
		}

		slidePaths = append(slidePaths, slidePath)

		// Extract speaker notes
		note := s.extractSpeakerNotes(slide)
		notes = append(notes, note)

		s.logger.Debug("Processed slide", "index", i, "path", slidePath, "noteLength", len(note))
	}

	return slidePaths, notes, nil
}

// createSlidesService creates a Google Slides API service with credentials
func (s *GoogleSlidesService) createSlidesService(ctx context.Context) (*slides.Service, error) {
	// Check for credentials file
	credentialsPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credentialsPath == "" {
		return nil, fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS environment variable not set. Please set it to the path of your service account credentials file. See GOOGLE_SLIDES_GUIDE.md for setup instructions")
	}

	// Create service with credentials
	service, err := slides.NewService(ctx, option.WithCredentialsFile(credentialsPath))
	if err != nil {
		return nil, fmt.Errorf("failed to create slides service: %w", err)
	}

	return service, nil
}

// downloadImage downloads an image from a URL and saves it to the filesystem
func (s *GoogleSlidesService) downloadImage(ctx context.Context, url, outputPath string) error {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Make HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download image: status %d", resp.StatusCode)
	}

	// Create output file
	file, err := s.fs.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy image data to file
	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("failed to write image: %w", err)
	}

	return nil
}

// extractSpeakerNotes extracts speaker notes from a slide
func (s *GoogleSlidesService) extractSpeakerNotes(slide *slides.Page) string {
	if slide.SlideProperties == nil || slide.SlideProperties.NotesPage == nil {
		return ""
	}

	notesPage := slide.SlideProperties.NotesPage
	var noteText string

	// Iterate through page elements to find text in notes
	for _, pageElement := range notesPage.PageElements {
		if pageElement.Shape != nil && pageElement.Shape.Text != nil {
			for _, textElement := range pageElement.Shape.Text.TextElements {
				if textElement.TextRun != nil {
					noteText += textElement.TextRun.Content
				}
			}
		}
	}

	return noteText
}

// LoadSlides is not implemented for Google Slides service
func (s *GoogleSlidesService) LoadSlides(ctx context.Context, dir string) ([]string, error) {
	return nil, fmt.Errorf("LoadSlides not implemented for GoogleSlidesService, use LoadFromGoogleSlides instead")
}
