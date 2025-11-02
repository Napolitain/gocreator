package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"gocreator/internal/adapters"
	"gocreator/internal/interfaces"
	"gocreator/internal/services"

	"github.com/openai/openai-go/v3"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// NewCreateCommand creates the create command
func NewCreateCommand() *cobra.Command {
	var inputLang string
	var outputLangs string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create videos with translations",
		Long:  `Create videos by processing text files, generating translations, audio, and combining with slides.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(inputLang, outputLangs)
		},
	}

	cmd.Flags().StringVarP(&inputLang, "lang", "l", "en", "Language of the text input")
	cmd.Flags().StringVarP(&outputLangs, "langs-out", "o", "en", "Comma-separated list of output languages")

	return cmd
}

func runCreate(inputLang, outputLangs string) error {
	// Setup structured logging
	slogger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	logger := &interfaces.SlogLogger{Logger: slogger}

	// Get working directory
	rootDir, err := os.Getwd()
	if err != nil {
		logger.Error("Failed to get working directory", "error", err)
		return err
	}

	// Initialize dependencies
	fs := afero.NewOsFs()
	openaiClient := openai.NewClient()
	openaiAdapter := adapters.NewOpenAIAdapter(openaiClient)

	// Create services with dependency injection
	textService := services.NewTextService(fs, logger)
	translationService := services.NewTranslationService(openaiAdapter, logger)
	audioService := services.NewAudioService(fs, openaiAdapter, textService, logger)
	videoService := services.NewVideoService(fs, logger)
	slideService := services.NewSlideService(fs, logger)

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

	// Parse output languages and ensure input language is included
	langsOut := parseLanguages(outputLangs, inputLang)

	// Create configuration
	cfg := services.VideoCreatorConfig{
		RootDir:     rootDir,
		InputLang:   inputLang,
		OutputLangs: langsOut,
	}

	// Run video creation
	ctx := context.Background()
	if err := creator.Create(ctx, cfg); err != nil {
		logger.Error("Video creation failed", "error", err)
		return err
	}

	fmt.Println("All videos created successfully!")
	return nil
}

// parseLanguages ensures the input language is first in the output list
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
