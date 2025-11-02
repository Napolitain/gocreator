package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"gocreator/internal/adapters"
	"gocreator/internal/interfaces"
	"gocreator/internal/services"

	"github.com/openai/openai-go/v3"
	"github.com/spf13/afero"
)

func main() {
	// Setup structured logging
	slogger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	logger := &interfaces.SlogLogger{Logger: slogger}

	// Parse flags
	inputLang := flag.String("lang", "en", "Language of the text input")
	outputLangs := flag.String("langs-out", "en", "Comma-separated list of output languages")
	flag.Parse()

	// Get working directory
	rootDir, err := os.Getwd()
	if err != nil {
		logger.Error("Failed to get working directory", "error", err)
		os.Exit(1)
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
	langsOut := parseLanguages(*outputLangs, *inputLang)

	// Create configuration
	cfg := services.VideoCreatorConfig{
		RootDir:     rootDir,
		InputLang:   *inputLang,
		OutputLangs: langsOut,
	}

	// Run video creation
	ctx := context.Background()
	if err := creator.Create(ctx, cfg); err != nil {
		logger.Error("Video creation failed", "error", err)
		os.Exit(1)
	}

	fmt.Println("All videos created successfully!")
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
