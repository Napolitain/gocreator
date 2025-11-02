package main

import (
	"flag"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gocreator/internal/audio"
	"gocreator/internal/text"
	"gocreator/internal/utils"
	"gocreator/internal/video"

	"github.com/openai/openai-go/v3"
)

func main() {
	// Setup structured logging
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	lang := flag.String("lang", "en", "Language of the text input (default: English, options are language codes and 'AI')")
	langsOut := flag.String("langs-out", "en", "Language of the text output (default: English, options can be multiple selections)")
	audioModelFlag := flag.String("audio-model", "openai", "Audio model (options: openai, google)")
	flag.Parse()

	audioModel := audio.Model(*audioModelFlag)
	rootDir, err := os.Getwd()
	if err != nil {
		logger.Error("Failed to get working directory", "error", err)
		os.Exit(1)
	}

	// Create OpenAI client
	openaiClient := openai.NewClient()

	dataDir := filepath.Join(rootDir, "data")

	// Verify data directory exists and has content
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		logger.Error("Data directory does not exist")
		os.Exit(1)
	}

	textsFile := filepath.Join(dataDir, "texts.txt")
	if _, err := os.Stat(textsFile); err != nil {
		logger.Error("Text file does not exist in data directory")
		os.Exit(1)
	}

	langsOutList := strings.Split(*langsOut, ",")
	if !utils.Contains(langsOutList, *lang) {
		langsOutList = append([]string{*lang}, langsOutList...)
	} else {
		langsOutList = utils.Remove(langsOutList, *lang)
		langsOutList = append([]string{*lang}, langsOutList...)
	}

	texts := text.Texts{
		LangIn:   *lang,
		LangsOut: langsOutList,
		DataDir:  dataDir,
		RootDir:  rootDir,
	}
	texts.GenerateTexts(openaiClient, logger)
	audios := audio.GenerateAudios(&texts, openaiClient, audioModel, logger)

	slides := []string{}
	slidesDir := filepath.Join(dataDir, "slides")
	if _, err := os.Stat(slidesDir); os.IsNotExist(err) {
		os.Mkdir(slidesDir, os.ModePerm)
	}
	files, err := os.ReadDir(slidesDir)
	if err != nil {
		logger.Error("Failed to read slides directory", "error", err)
		os.Exit(1)
	}
	for _, file := range files {
		if strings.ToLower(filepath.Ext(file.Name())) == ".png" ||
			strings.ToLower(filepath.Ext(file.Name())) == ".jpg" ||
			strings.ToLower(filepath.Ext(file.Name())) == ".jpeg" {
			slides = append(slides, filepath.Join(slidesDir, file.Name()))
		}
	}

	video.GenerateVideos(slides, audios, dataDir, logger)
}
