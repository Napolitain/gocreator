//go:build windows

package com

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gocreator/internal/adapters"
	"gocreator/internal/interfaces"
	"gocreator/internal/services"

	"github.com/go-ole/go-ole"
	"github.com/openai/openai-go/v3"
	"github.com/spf13/afero"
)

// CLSID for GoCreator COM object
// {8B9C5A3E-1234-5678-9ABC-DEF012345678}
var CLSID_GoCreator = ole.NewGUID("8B9C5A3E-1234-5678-9ABC-DEF012345678")

// IID for IGoCreator interface
// {9C8D6B4F-2345-6789-ABCD-EF0123456789}
var IID_IGoCreator = ole.NewGUID("9C8D6B4F-2345-6789-ABCD-EF0123456789")

// GoCreatorCOM is the COM server implementation
type GoCreatorCOM struct {
	logger  interfaces.Logger
	mu      sync.Mutex
	rootDir string
}

// NewGoCreatorCOM creates a new COM server instance
func NewGoCreatorCOM() *GoCreatorCOM {
	slogger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	logger := &interfaces.SlogLogger{Logger: slogger}

	rootDir, err := os.Getwd()
	if err != nil {
		rootDir = "."
	}

	return &GoCreatorCOM{
		logger:  logger,
		rootDir: rootDir,
	}
}

// SetRootDirectory sets the working directory for video creation
func (gc *GoCreatorCOM) SetRootDirectory(path string) error {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	// Validate the path exists
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("invalid root directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", path)
	}

	gc.rootDir = path
	gc.logger.Info("Root directory set", "path", path)
	return nil
}

// CreateVideo creates a video with the specified configuration
// Parameters:
//   - inputLang: Language of the input text (e.g., "en")
//   - outputLangs: Comma-separated list of output languages (e.g., "en,fr,es")
//   - googleSlidesID: Google Slides presentation ID (empty string for local slides)
func (gc *GoCreatorCOM) CreateVideo(inputLang, outputLangs, googleSlidesID string) error {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	gc.logger.Info("CreateVideo called",
		"inputLang", inputLang,
		"outputLangs", outputLangs,
		"googleSlidesID", googleSlidesID,
		"rootDir", gc.rootDir)

	// Initialize dependencies
	fs := afero.NewOsFs()
	openaiClient := openai.NewClient()
	openaiAdapter := adapters.NewOpenAIAdapter(openaiClient)

	// Create services with dependency injection
	textService := services.NewTextService(fs, gc.logger)
	translationService := services.NewTranslationService(openaiAdapter, gc.logger)
	audioService := services.NewAudioService(fs, openaiAdapter, textService, gc.logger)
	videoService := services.NewVideoService(fs, gc.logger)

	// Choose slide service based on whether Google Slides ID is provided
	var slideService interfaces.SlideLoader
	if googleSlidesID != "" {
		slideService = services.NewGoogleSlidesService(fs, gc.logger)
		gc.logger.Info("Using Google Slides", "presentationID", googleSlidesID)
	} else {
		slideService = services.NewSlideService(fs, gc.logger)
		gc.logger.Info("Using local slides")
	}

	// Create video creator
	creator := services.NewVideoCreator(
		fs,
		textService,
		translationService,
		audioService,
		videoService,
		slideService,
		gc.logger,
	)

	// Parse output languages
	langsOut := parseLanguages(outputLangs, inputLang)

	// Create configuration
	cfg := services.VideoCreatorConfig{
		RootDir:        gc.rootDir,
		InputLang:      inputLang,
		OutputLangs:    langsOut,
		GoogleSlidesID: googleSlidesID,
	}

	// Run video creation
	ctx := context.Background()
	if err := creator.Create(ctx, cfg); err != nil {
		gc.logger.Error("Video creation failed", "error", err)
		return err
	}

	gc.logger.Info("Video created successfully")
	return nil
}

// GetVersion returns the version of the COM server
func (gc *GoCreatorCOM) GetVersion() string {
	return "1.0.0"
}

// GetOutputPath returns the expected output path for a given language
func (gc *GoCreatorCOM) GetOutputPath(lang string) string {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	return filepath.Join(gc.rootDir, "data", "out", fmt.Sprintf("output-%s.mp4", lang))
}

// parseLanguages ensures the input language is first in the output list
func parseLanguages(outputLangs, inputLang string) []string {
	if outputLangs == "" {
		return []string{inputLang}
	}

	var langs []string
	for _, lang := range splitCommaSeparated(outputLangs) {
		if lang != "" && lang != inputLang {
			langs = append(langs, lang)
		}
	}

	// Prepend input language
	return append([]string{inputLang}, langs...)
}

// splitCommaSeparated splits a comma-separated string and trims whitespace
func splitCommaSeparated(s string) []string {
	if s == "" {
		return nil
	}

	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// RegisterCOMServer registers the COM server in the Windows registry
func RegisterCOMServer(exePath string) error {
	// Initialize COM
	err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED)
	if err != nil {
		return fmt.Errorf("failed to initialize COM: %w", err)
	}
	defer ole.CoUninitialize()

	// This is a simplified registration - in production, you would need to:
	// 1. Create registry keys under HKEY_CLASSES_ROOT\CLSID\{CLSID}
	// 2. Set InprocServer32 or LocalServer32 to the executable path
	// 3. Set threading model, etc.
	//
	// For now, we just log the intended registration
	fmt.Printf("COM Server registration (manual step required):\n")
	fmt.Printf("  CLSID: %s\n", CLSID_GoCreator.String())
	fmt.Printf("  IID: %s\n", IID_IGoCreator.String())
	fmt.Printf("  Executable: %s\n", exePath)
	fmt.Println("\nTo register manually, create registry keys under:")
	fmt.Printf("  HKEY_CLASSES_ROOT\\CLSID\\%s\n", CLSID_GoCreator.String())
	fmt.Printf("  Set LocalServer32 = %s\n", exePath)

	return nil
}

// UnregisterCOMServer unregisters the COM server from the Windows registry
func UnregisterCOMServer() error {
	// Initialize COM
	err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED)
	if err != nil {
		return fmt.Errorf("failed to initialize COM: %w", err)
	}
	defer ole.CoUninitialize()

	// This is a simplified unregistration - in production, you would:
	// 1. Delete registry keys under HKEY_CLASSES_ROOT\CLSID\{CLSID}
	//
	// For now, we just log the intended unregistration
	fmt.Printf("COM Server unregistration (manual step required):\n")
	fmt.Printf("  CLSID: %s\n", CLSID_GoCreator.String())
	fmt.Println("\nTo unregister manually, delete registry keys under:")
	fmt.Printf("  HKEY_CLASSES_ROOT\\CLSID\\%s\n", CLSID_GoCreator.String())

	return nil
}

// IsCOMAvailable returns true if COM support is available on this platform
func IsCOMAvailable() bool {
	return true
}
