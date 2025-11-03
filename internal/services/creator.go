package services

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"gocreator/internal/interfaces"

	"github.com/spf13/afero"
)

// VideoCreatorConfig holds configuration for video creation
type VideoCreatorConfig struct {
	RootDir        string
	InputLang      string
	OutputLangs    []string
	GoogleSlidesID string // Google Slides presentation ID (found in the URL). When empty, uses local slides; when provided, fetches from Google Slides API
}

// VideoCreator orchestrates the video creation process
type VideoCreator struct {
	fs                 afero.Fs
	textService        interfaces.TextProcessor
	translationService interfaces.Translator
	audioService       interfaces.AudioGenerator
	videoService       interfaces.VideoGenerator
	slideService       interfaces.SlideLoader
	logger             interfaces.Logger
}

// NewVideoCreator creates a new video creator
func NewVideoCreator(
	fs afero.Fs,
	textService interfaces.TextProcessor,
	translationService interfaces.Translator,
	audioService interfaces.AudioGenerator,
	videoService interfaces.VideoGenerator,
	slideService interfaces.SlideLoader,
	logger interfaces.Logger,
) *VideoCreator {
	return &VideoCreator{
		fs:                 fs,
		textService:        textService,
		translationService: translationService,
		audioService:       audioService,
		videoService:       videoService,
		slideService:       slideService,
		logger:             logger,
	}
}

// Create creates videos for all specified languages
func (vc *VideoCreator) Create(ctx context.Context, cfg VideoCreatorConfig) error {
	dataDir := filepath.Join(cfg.RootDir, "data")

	var inputTexts []string
	var slides []string
	var err error

	// Check if using Google Slides
	if cfg.GoogleSlidesID != "" {
		// Fetch slides and notes from Google Slides
		slidesDir := filepath.Join(dataDir, "slides")
		slides, inputTexts, err = vc.slideService.LoadFromGoogleSlides(ctx, cfg.GoogleSlidesID, slidesDir)
		if err != nil {
			return fmt.Errorf("failed to load Google Slides: %w", err)
		}
		vc.logger.Info("Loaded from Google Slides", "slideCount", len(slides), "noteCount", len(inputTexts))

		// Save the fetched notes as input texts for caching
		textsPath := filepath.Join(dataDir, "texts.txt")
		if err := vc.textService.Save(ctx, textsPath, inputTexts); err != nil {
			// Saving fetched notes is only for caching purposes. It's acceptable to continue without saving,
			// for example, if running on a read-only filesystem or if caching is not critical for correctness.
			vc.logger.Warn("Failed to save fetched notes", "error", err)
		}
	} else {
		// Load input texts from file
		textsPath := filepath.Join(dataDir, "texts.txt")
		inputTexts, err = vc.textService.Load(ctx, textsPath)
		if err != nil {
			return fmt.Errorf("failed to load input texts: %w", err)
		}

		vc.logger.Info("Loaded texts", "count", len(inputTexts))

		// Load slides from directory
		slidesDir := filepath.Join(dataDir, "slides")
		slides, err = vc.slideService.LoadSlides(ctx, slidesDir)
		if err != nil {
			return fmt.Errorf("failed to load slides: %w", err)
		}

		vc.logger.Info("Loaded slides", "count", len(slides))
	}

	if len(slides) != len(inputTexts) {
		return fmt.Errorf("slide and text count mismatch: %d slides, %d texts", len(slides), len(inputTexts))
	}

	// Process each language in parallel
	var wg sync.WaitGroup
	errors := make([]error, len(cfg.OutputLangs))
	
	for i, lang := range cfg.OutputLangs {
		wg.Add(1)
		go func(idx int, l string) {
			defer wg.Done()
			if err := vc.processLanguage(ctx, cfg, l, inputTexts, slides, dataDir); err != nil {
				errors[idx] = fmt.Errorf("failed to process language %s: %w", l, err)
			}
		}(i, lang)
	}
	
	wg.Wait()
	
	// Check for any errors
	for _, err := range errors {
		if err != nil {
			return err
		}
	}

	return nil
}

func (vc *VideoCreator) processLanguage(
	ctx context.Context,
	cfg VideoCreatorConfig,
	lang string,
	inputTexts []string,
	slides []string,
	dataDir string,
) error {
	logger := vc.logger.With("lang", lang)
	logger.Info("Processing language")

	cacheDir := filepath.Join(dataDir, "cache", lang)
	textDir := filepath.Join(cacheDir, "text")
	audioDir := filepath.Join(cacheDir, "audio")

	var texts []string
	var err error

	// Translate if needed
	if lang == cfg.InputLang {
		texts = inputTexts
	} else {
		textsPath := filepath.Join(textDir, "texts.txt")

		// Check if translation exists
		exists, err := afero.Exists(vc.fs, textsPath)
		if err != nil {
			return fmt.Errorf("failed to check translation cache: %w", err)
		}

		if exists {
			logger.Info("Loading cached translation")
			texts, err = vc.textService.Load(ctx, textsPath)
			if err != nil {
				return fmt.Errorf("failed to load cached translation: %w", err)
			}
		} else {
			logger.Info("Translating texts")
			texts, err = vc.translationService.TranslateBatch(ctx, inputTexts, lang)
			if err != nil {
				return fmt.Errorf("translation failed: %w", err)
			}

			// Save translated texts
			if err := vc.textService.Save(ctx, textsPath, texts); err != nil {
				return fmt.Errorf("failed to save translation: %w", err)
			}
		}
	}

	// Generate audio
	logger.Info("Generating audio")
	audioPaths, err := vc.audioService.GenerateBatch(ctx, texts, audioDir)
	if err != nil {
		return fmt.Errorf("audio generation failed: %w", err)
	}

	// Generate video
	logger.Info("Generating video")
	outputDir := filepath.Join(dataDir, "out")
	outputPath := filepath.Join(outputDir, fmt.Sprintf("output-%s.mp4", lang))

	if err := vc.videoService.GenerateFromSlides(ctx, slides, audioPaths, outputPath); err != nil {
		return fmt.Errorf("video generation failed: %w", err)
	}

	logger.Info("Video created successfully", "path", outputPath)
	return nil
}
