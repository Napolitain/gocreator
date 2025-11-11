package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gocreator/internal/adapters"
	"gocreator/internal/config"
	"gocreator/internal/interfaces"
	"gocreator/internal/services"
	"gocreator/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/openai/openai-go/v3"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// NewCreateCommand creates the create command
func NewCreateCommand() *cobra.Command {
	var inputLang string
	var outputLangs string
	var googleSlidesID string
	var configFile string
	var noProgress bool

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create videos with translations",
		Long:  `Create videos by processing text files, generating translations, audio, and combining with slides.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(inputLang, outputLangs, googleSlidesID, configFile, noProgress)
		},
	}

	cmd.Flags().StringVarP(&inputLang, "lang", "l", "", "Language of the text input (overrides config file)")
	cmd.Flags().StringVarP(&outputLangs, "langs-out", "o", "", "Comma-separated list of output languages (overrides config file)")
	cmd.Flags().StringVar(&googleSlidesID, "google-slides", "", "Google Slides presentation ID (overrides config file)")
	cmd.Flags().StringVarP(&configFile, "config", "c", "", "Config file path (default: looks for gocreator.yaml in current and parent directories)")
	cmd.Flags().BoolVar(&noProgress, "no-progress", false, "Disable progress UI")

	return cmd
}

func runCreate(inputLang, outputLangs, googleSlidesID, configFile string, noProgress bool) error {
	// Get working directory
	rootDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Initialize filesystem
	fs := afero.NewOsFs()

	// Load configuration
	var cfg *config.Config
	if configFile != "" {
		// Use specified config file
		cfg, err = config.LoadConfig(fs, configFile)
		if err != nil {
			return fmt.Errorf("failed to load config file %s: %w", configFile, err)
		}
		fmt.Printf("✓ Loaded config from %s\n", configFile)
	} else {
		// Try to find config file
		foundPath, err := config.FindConfigFile(fs)
		if err != nil {
			return fmt.Errorf("error searching for config file: %w", err)
		}
		
		if foundPath != "" {
			cfg, err = config.LoadConfig(fs, foundPath)
			if err != nil {
				return fmt.Errorf("failed to load config file %s: %w", foundPath, err)
			}
			fmt.Printf("✓ Loaded config from %s\n", foundPath)
		} else {
			// Use default config
			cfg = config.DefaultConfig()
			fmt.Println("ℹ Using default configuration (no config file found)")
		}
	}

	// Override config with command-line flags
	if inputLang != "" {
		cfg.Input.Lang = inputLang
	}
	if outputLangs != "" {
		cfg.Output.Languages = parseLanguages(outputLangs, cfg.Input.Lang)
	}
	if googleSlidesID != "" {
		cfg.Input.Source = "google-slides"
		cfg.Input.PresentationID = googleSlidesID
	}

	// Ensure input language is in output languages
	if len(cfg.Output.Languages) == 0 {
		cfg.Output.Languages = []string{cfg.Input.Lang}
	}
	cfg.Output.Languages = ensureInputLanguageFirst(cfg.Output.Languages, cfg.Input.Lang)

	// Setup logging
	slogger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	logger := &interfaces.SlogLogger{Logger: slogger}

	// Initialize progress UI if enabled
	var prog *tea.Program
	var progressAdapter *ui.ProgressAdapter
	if !noProgress {
		progressModel := ui.NewProgressModel()
		prog = tea.NewProgram(progressModel)
		progressAdapter = ui.NewProgressAdapter(prog)
		
		// Run progress UI in background
		go func() {
			if _, err := prog.Run(); err != nil {
				logger.Error("Progress UI error", "error", err)
			}
		}()
	}

	// Initialize OpenAI client
	openaiClient := openai.NewClient()
	openaiAdapter := adapters.NewOpenAIAdapter(openaiClient)

	// Create services with dependency injection
	textService := services.NewTextService(fs, logger)
	
	// Create translation service with disk cache
	translationCacheDir := filepath.Join(rootDir, cfg.Cache.Directory, "translations")
	translationService := services.NewTranslationServiceWithCache(openaiAdapter, logger, fs, translationCacheDir)
	
	audioService := services.NewAudioService(fs, openaiAdapter, textService, logger)
	videoService := services.NewVideoService(fs, logger)
	
	// Choose slide service based on source
	var slideService interfaces.SlideLoader
	if cfg.Input.Source == "google-slides" && cfg.Input.PresentationID != "" {
		slideService = services.NewGoogleSlidesService(fs, logger)
		logger.Info("Using Google Slides", "presentationID", cfg.Input.PresentationID)
	} else {
		slideService = services.NewSlideService(fs, logger)
		logger.Info("Using local slides")
	}

	// Create video creator
	creator := services.NewVideoCreator(
		fs,
		textService,
		translationService,
		audioService,
		videoService,
		slideService,
		logger,
	)

	// Create video creator configuration with progress callback
	var progressCallback interfaces.ProgressCallback
	if progressAdapter != nil {
		progressCallback = progressAdapter
	} else {
		progressCallback = &interfaces.NoOpProgressCallback{}
	}

	// Convert config transition to services transition
	transition := services.TransitionConfig{
		Type:     services.TransitionType(cfg.Transition.Type),
		Duration: cfg.Transition.Duration,
	}

	// If transition is not valid, use default (none)
	if err := transition.Validate(); err != nil {
		logger.Warn("Invalid transition configuration, using default (none)", "error", err)
		transition = services.TransitionConfig{Type: services.TransitionNone, Duration: 0.0}
	}

	creatorCfg := services.VideoCreatorConfig{
		RootDir:          rootDir,
		InputLang:        cfg.Input.Lang,
		OutputLangs:      cfg.Output.Languages,
		GoogleSlidesID:   cfg.Input.PresentationID,
		ProgressCallback: progressCallback,
		Transition:       transition,
	}

	// Run video creation
	ctx := context.Background()
	if err := creator.Create(ctx, creatorCfg); err != nil {
		if prog != nil {
			prog.Send(ui.CompleteMsg{})
			prog.Wait()
		}
		return fmt.Errorf("video creation failed: %w", err)
	}

	// Complete progress
	if prog != nil {
		prog.Send(ui.CompleteMsg{})
		prog.Wait()
	}

	if noProgress {
		fmt.Println("✓ All videos created successfully!")
	}
	return nil
}

// parseLanguages parses comma-separated languages
func parseLanguages(outputLangs, inputLang string) []string {
	langs := strings.Split(outputLangs, ",")

	// Remove input language if it exists elsewhere
	filtered := make([]string, 0, len(langs))
	for _, lang := range langs {
		lang = strings.TrimSpace(lang)
		if lang != "" && lang != inputLang {
			filtered = append(filtered, lang)
		}
	}

	// Prepend input language
	return append([]string{inputLang}, filtered...)
}

// ensureInputLanguageFirst ensures input language is first in the list
func ensureInputLanguageFirst(languages []string, inputLang string) []string {
	// Remove input language if it exists elsewhere
	filtered := make([]string, 0, len(languages))
	for _, lang := range languages {
		if lang != inputLang {
			filtered = append(filtered, lang)
		}
	}

	// Prepend input language
	return append([]string{inputLang}, filtered...)
}
