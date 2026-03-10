package services

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"gocreator/internal/config"
	"gocreator/internal/interfaces"

	"github.com/spf13/afero"
)

// VideoCreatorConfig holds configuration for video creation
type VideoCreatorConfig struct {
	RootDir          string
	InputLang        string
	OutputLangs      []string
	ProgressCallback interfaces.ProgressCallback
	Transition       TransitionConfig        // Transition configuration for slide transitions
	Timing           config.TimingConfig     // Timing and alignment configuration
	Effects          []config.EffectConfig   // Per-slide visual effects
	MultiView        *config.MultiViewConfig // Multi-view configuration for split-screen layouts
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

	// Use no-op callback if none provided
	progress := cfg.ProgressCallback
	if progress == nil {
		progress = &interfaces.NoOpProgressCallback{}
	}

	// Configure video service with transitions and multi-view if available
	if videoService, ok := vc.videoService.(*VideoService); ok {
		if err := videoService.SetMediaAlignment(cfg.Timing.MediaAlignment); err != nil {
			return fmt.Errorf("invalid media alignment: %w", err)
		}

		if err := cfg.Transition.Validate(); err == nil && cfg.Transition.IsEnabled() {
			videoService.SetTransition(cfg.Transition)
			vc.logger.Info("Transitions enabled", "type", cfg.Transition.Type, "duration", cfg.Transition.Duration)
		} else {
			if err != nil {
				vc.logger.Warn("Transitions not enabled due to validation failure", "type", cfg.Transition.Type, "duration", cfg.Transition.Duration, "error", err)
			} else if !cfg.Transition.IsEnabled() {
				vc.logger.Debug("Transitions are disabled", "type", cfg.Transition.Type)
			}
		}

		videoService.SetEffects(cfg.Effects)
		if len(cfg.Effects) > 0 {
			vc.logger.Info("Effects enabled", "count", len(cfg.Effects))
		}

		// Configure multi-view if enabled
		if cfg.MultiView != nil && cfg.MultiView.Enabled {
			videoService.SetMultiView(cfg.MultiView)
			vc.logger.Info("Multi-view enabled", "layouts", len(cfg.MultiView.Layouts))
		}
	}

	var slides []string
	var err error

	// Loading stage
	progress.OnStageStart("Loading")

	progress.OnStageProgress("Loading", 60, "Loading slides")

	// Load slides from directory
	slidesDir := filepath.Join(dataDir, "slides")
	slides, err = vc.slideService.LoadSlides(ctx, slidesDir)
	if err != nil {
		progress.OnStageComplete("Loading", false, fmt.Sprintf("Failed: %v", err))
		return fmt.Errorf("failed to load slides: %w", err)
	}

	vc.logger.Info("Loaded slides", "count", len(slides))

	if len(slides) == 0 {
		progress.OnStageComplete("Loading", false, "No slides found")
		return fmt.Errorf("no slides found in %s", slidesDir)
	}

	progress.OnStageComplete("Loading", true, fmt.Sprintf("Loaded %d slides", len(slides)))

	// Process each language in parallel
	var wg sync.WaitGroup
	errors := make([]error, len(cfg.OutputLangs))

	for i, lang := range cfg.OutputLangs {
		wg.Add(1)
		go func(idx int, l string) {
			defer wg.Done()
			if err := vc.processLanguage(ctx, cfg, l, slides, slidesDir, dataDir, progress); err != nil {
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
	slides []string,
	slidesDir string,
	dataDir string,
	progress interfaces.ProgressCallback,
) error {
	logger := vc.logger.With("lang", lang)
	logger.Info("Processing language")

	cacheDir := filepath.Join(dataDir, "cache", lang)
	audioDir := filepath.Join(cacheDir, "audio")

	// Translation stage
	progress.OnItemStart("Translation", lang)
	progress.OnItemProgress("Translation", lang, 40, "Resolving slide sidecars...")
	texts, translatedCount, err := vc.resolveTextsForLanguage(ctx, cfg.InputLang, lang, slidesDir, slides)
	if err != nil {
		progress.OnItemComplete("Translation", lang, false, fmt.Sprintf("Error: %v", err))
		return fmt.Errorf("failed to resolve texts: %w", err)
	}

	switch {
	case translatedCount > 0:
		progress.OnItemComplete("Translation", lang, true, fmt.Sprintf("Translated %d slide texts", translatedCount))
	default:
		progress.OnItemComplete("Translation", lang, true, "Using local sidecars")
	}

	// Audio generation stage
	progress.OnItemStart("Audio Generation", lang)
	progress.OnItemProgress("Audio Generation", lang, 30, "Resolving narration...")
	audioPaths, prerecordedCount, generatedCount, err := vc.resolveAudioForLanguage(ctx, cfg.InputLang, lang, slidesDir, slides, texts, audioDir)
	if err != nil {
		progress.OnItemComplete("Audio Generation", lang, false, fmt.Sprintf("Error: %v", err))
		return fmt.Errorf("audio generation failed: %w", err)
	}
	progress.OnItemComplete("Audio Generation", lang, true, fmt.Sprintf("Using %d prerecorded and %d generated tracks", prerecordedCount, generatedCount))

	// Video assembly stage
	progress.OnItemStart("Video Assembly", lang)
	logger.Info("Generating video")
	progress.OnItemProgress("Video Assembly", lang, 30, "Assembling video...")

	outputDir := filepath.Join(dataDir, "out")
	outputPath := filepath.Join(outputDir, fmt.Sprintf("output-%s.mp4", lang))

	if err := vc.videoService.GenerateFromSlides(ctx, slides, audioPaths, outputPath); err != nil {
		progress.OnItemComplete("Video Assembly", lang, false, fmt.Sprintf("Error: %v", err))
		return fmt.Errorf("video generation failed: %w", err)
	}

	logger.Info("Video created successfully", "path", outputPath)
	progress.OnItemComplete("Video Assembly", lang, true, "Video complete")
	return nil
}
