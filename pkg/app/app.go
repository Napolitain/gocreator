package app

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
)

var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

// isSymlink checks if a file is a symlink
func isSymlink(path string) (bool, error) {
	fileInfo, err := os.Lstat(path)
	if err != nil {
		return false, err
	}
	return fileInfo.Mode()&os.ModeSymlink != 0, nil
}

// copyFile copies a file from src to dst
// src and dst are file paths
func copyFile(src string, dst string, followSymlinks bool) error {
	// Is it symlink
	isSymlink, err := isSymlink(src)
	if err != nil {
		return err
	}
	if isSymlink && !followSymlinks {
		return nil
	}

	// Copy file
	// Open the source file
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = sourceFile.Close() }()

	// Create the destination file
	destinationFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = destinationFile.Close() }()

	// Copy the contents from source to destination
	if _, err = io.Copy(destinationFile, sourceFile); err != nil {
		return err
	}

	// Flush the destination file to ensure all data is written
	if err = destinationFile.Sync(); err != nil {
		return err
	}

	return nil
}

func copyTree(src string, dst string, followSymlinks bool) error {
	stack := [][2]string{{src, dst}}
	for len(stack) > 0 {
		// Pop the stack
		current := stack[0][0]
		currentDst := stack[0][1]
		stack = stack[1:]

		// Is it symlink
		isSymlink, err := isSymlink(current)
		if err != nil {
			return err
		}
		if isSymlink && !followSymlinks {
			continue
		}

		// Is it a file
		fileInfo, err := os.Stat(current)
		if err != nil {
			return err
		}
		if !fileInfo.IsDir() {
			err := copyFile(current, currentDst, followSymlinks)
			// If file already exists, skip
			if os.IsExist(err) {
				continue
			}
			if err != nil {
				return err
			}
		} else { // Else it is a directory
			// Mkdir the dir
			err := os.MkdirAll(currentDst, os.ModePerm)
			if err != nil && !os.IsExist(err) {
				return err
			}
			// Add its children path to the stack
			children, err := os.ReadDir(current)
			if err != nil {
				return err
			}
			for _, child := range children {
				// Path is current join child
				childPath := filepath.Join(current, child.Name())
				childDst := filepath.Join(currentDst, child.Name())
				stack = append(stack, [2]string{childPath, childDst})
			}
		}
	}
	return nil
}

// AudioGenerationModel represents the model to use for audio generation
type AudioGenerationModel string

const (
	// Audio generation models
	OPENAI_TTS AudioGenerationModel = "openai"
	GOOGLE_TTS AudioGenerationModel = "google"
)

// SlideText is the struct for the slide text content
type SlideText struct {
	Text string
}

func newSlideText(text string) *SlideText {
	return &SlideText{Text: text}
}

func (s *SlideText) hash() string {
	hash := sha256.New()
	hash.Write([]byte(s.Text))
	return hex.EncodeToString(hash.Sum(nil))
}

// Text is an array of SlideText, corresponding basically to a full video.
type Text struct {
	Lang       string
	RootDir    string
	DataDir    string
	CacheDir   string
	LangDir    string
	AudioDir   string
	TextDir    string
	OpenAI     openai.Client
	SlidesText []*SlideText
	Hashes     []string
}

// newText creates a new Text struct. It corresponds to an entire video text (single language).
func newText(rootDir, lang string, client openai.Client, logger *slog.Logger) (*Text, error) {
	dataDir := filepath.Join(rootDir, "data")
	cacheDir := filepath.Join(dataDir, "cache")
	langDir := filepath.Join(cacheDir, lang)
	audioDir := filepath.Join(langDir, "audio")
	textDir := filepath.Join(langDir, "text")

	dirs := []string{cacheDir, langDir, audioDir, textDir}
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			err := os.MkdirAll(dir, os.ModePerm)
			if err != nil {
				logger.Error("Failed to create directory", "dir", dir, "error", err)
				return nil, err
			}
		}
	}

	return &Text{
		Lang:       lang,
		RootDir:    rootDir,
		DataDir:    dataDir,
		CacheDir:   cacheDir,
		LangDir:    langDir,
		AudioDir:   audioDir,
		TextDir:    textDir,
		OpenAI:     client,
		SlidesText: []*SlideText{},
		Hashes:     []string{},
	}, nil
}

// generateText loads the text from the text file.
func (t *Text) generateText(inputText *Text, logger *slog.Logger) error {
	var textFilePath string
	if inputText == nil {
		textFilePath = filepath.Join(t.DataDir, "texts.txt")
	} else {
		textFilePath = filepath.Join(t.TextDir, "texts.txt")
	}

	textFile, err := os.Open(textFilePath)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := textFile.Close(); cerr != nil {
			logger.Error("Failed to close text file", "error", cerr)
		}
	}()

	if inputText != nil {
		if err := t.translateText(inputText, logger); err != nil {
			return err
		}
	} else {
		if err := t.loadTextInput(textFile); err != nil {
			return err
		}
	}

	return nil
}

func (t *Text) translateText(inputText *Text, logger *slog.Logger) error {
	// Simplified: perform translations sequentially so errors can be returned easily
	t.SlidesText = make([]*SlideText, len(inputText.SlidesText))
	t.Hashes = make([]string, len(inputText.SlidesText))
	for i, slideText := range inputText.SlidesText {
		resp, err := t.OpenAI.Chat.Completions.New(
			context.Background(),
			openai.ChatCompletionNewParams{
				Model: openai.ChatModelGPT4oMini,
				Messages: []openai.ChatCompletionMessageParamUnion{
					openai.UserMessage(fmt.Sprintf("Translate '%s' to %s and don't return anything else than the translation.", slideText.Text, t.Lang)),
				},
			},
		)
		if err != nil {
			logger.Error("Translation error", "error", err)
			return err
		}

		translatedText := resp.Choices[0].Message.Content
		t.SlidesText[i] = newSlideText(translatedText)
		t.Hashes[i] = inputText.Hashes[i]
	}

	if err := t.saveTextFile(t.SlidesText); err != nil {
		return err
	}
	return nil
}

func (t *Text) loadTextInput(textFile *os.File) error {
	scanner := bufio.NewScanner(textFile)
	slideText := ""
	t.Hashes = []string{}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "-" {
			s := newSlideText(slideText)
			t.SlidesText = append(t.SlidesText, s)
			t.Hashes = append(t.Hashes, s.hash())
			slideText = ""
		} else {
			slideText += line + "\n"
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	// Append last slide if any
	if strings.TrimSpace(slideText) != "" {
		s := newSlideText(slideText)
		t.SlidesText = append(t.SlidesText, s)
		t.Hashes = append(t.Hashes, s.hash())
	}
	return nil
}

func (t *Text) saveTextFile(slidesText []*SlideText) error {
	textFile := filepath.Join(t.TextDir, "texts.txt")
	hashFile := filepath.Join(t.TextDir, "hashes")

	textF, err := os.Create(textFile)
	if err != nil {
		return err
	}
	defer func() { _ = textF.Close() }()

	hashF, err := os.Create(hashFile)
	if err != nil {
		return err
	}
	defer func() { _ = hashF.Close() }()

	for i, slideText := range slidesText {
		if i == len(slidesText)-1 {
			if _, err := textF.WriteString(slideText.Text); err != nil {
				return err
			}
			if _, err := hashF.WriteString(t.Hashes[i]); err != nil {
				return err
			}
		} else {
			if _, err := textF.WriteString(slideText.Text + "\n-\n"); err != nil {
				return err
			}
			if _, err := hashF.WriteString(t.Hashes[i] + "\n"); err != nil {
				return err
			}
		}
	}
	return nil
}

func (t *Text) generateCacheHashes(directory string) []string {
	hashFile := filepath.Join(directory, "hashes")
	if _, err := os.Stat(hashFile); os.IsNotExist(err) {
		return make([]string, len(t.SlidesText))
	}

	file, err := os.Open(hashFile)
	if err != nil {
		return make([]string, len(t.SlidesText))
	}
	defer func() { _ = file.Close() }()

	var hashes []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		hashes = append(hashes, scanner.Text())
	}
	return hashes
}

// Texts struct is a set of texts for different languages
type Texts struct {
	LangIn   string
	LangsOut []string
	DataDir  string
	Texts    []*Text
	RootDir  string
}

func (t *Texts) generateTexts(client openai.Client, logger *slog.Logger) error {
	t.Texts = make([]*Text, len(t.LangsOut))

	// Find the index of the input language and generate its Text first
	var baseIdx = -1
	for i, langOut := range t.LangsOut {
		if t.LangIn == langOut {
			text, err := newText(t.RootDir, langOut, client, logger)
			if err != nil {
				logger.Error("Failed to create text", "error", err)
				return err
			}
			if err := text.generateText(nil, logger); err != nil {
				logger.Error("Failed to generate text", "error", err)
				return err
			}
			t.Texts[i] = text
			baseIdx = i
			break
		}
	}

	if baseIdx == -1 {
		return fmt.Errorf("input language %s not found in output languages", t.LangIn)
	}

	// Generate translations for the other languages
	for i, langOut := range t.LangsOut {
		if t.LangIn != langOut {
			text, err := newText(t.RootDir, langOut, client, logger)
			if err != nil {
				logger.Error("Failed to create text", "error", err)
				return err
			}
			if err := text.generateText(t.Texts[baseIdx], logger); err != nil {
				logger.Error("Failed to generate text", "error", err)
				return err
			}
			t.Texts[i] = text
		}
	}
	return nil
}

// Audio generation types and functions
func generateAudioLocal(text, audioPath, title string, seed int, logger *slog.Logger) string {
	logger.Info("Generating audio (local)", "text", text)
	outputFilename := fmt.Sprintf("%s.wav", strings.TrimSuffix(title, filepath.Ext(title)))

	if _, err := os.Stat(outputFilename); err == nil {
		logger.Info("Audio already exists", "filename", outputFilename)
		return outputFilename
	}

	requestBody := AudioRequest{
		ModelChoice:       "",
		Text:              text,
		Language:          "en-us",
		SpeakerAudio:      audioPath,
		PrefixAudio:       "assets/silence_100ms.wav",
		E1:                1,
		E2:                0.05,
		E3:                0.05,
		E4:                0.05,
		E5:                0.05,
		E6:                0.05,
		E7:                0.1,
		E8:                0.2,
		VQSingle:          0.78,
		Fmax:              24000,
		PitchStd:          45,
		SpeakingRate:      15,
		DnsmosOvrl:        4,
		SpeakerNoised:     false,
		CfgScale:          2,
		TopP:              0,
		TopK:              0,
		MinP:              0,
		Linear:            0.5,
		Confidence:        2,
		Quadratic:         0,
		Seed:              seed,
		RandomizeSeed:     true,
		UnconditionalKeys: []string{"string"},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		logger.Error("Failed to marshal request body", "error", err)
		return ""
	}

	// Use default local endpoint; make configurable later
	resp, err := http.Post("http://localhost:7860/generate_audio", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Error("Request failed", "error", err)
		return ""
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		logger.Error("HTTP error", "status", resp.Status)
		return ""
	}

	var response AudioResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		logger.Error("JSON decode failed", "error", err)
		return ""
	}
	audioData, err := base64.StdEncoding.DecodeString(response.AudioBase64)
	if err != nil {
		logger.Error("Base64 decode failed", "error", err)
		return ""
	}
	if err := os.WriteFile(outputFilename, audioData, 0644); err != nil {
		logger.Error("File write failed", "error", err)
		return ""
	}

	logger.Info("Audio saved", "filename", outputFilename)
	return outputFilename
}

func splitText(text string) []string {
	// naive sentence split; will be refined in later stages
	sentences := strings.Split(text, ".")
	var parts []string
	for _, sentence := range sentences {
		trimmed := strings.TrimSpace(sentence)
		if len(trimmed) > 0 {
			parts = append(parts, trimmed+".")
		}
	}
	return parts
}

func createSilenceAudio(outputFilename string, duration int, logger *slog.Logger) error {
	cmd := exec.Command("ffmpeg", "-y", "-f", "lavfi", "-i", fmt.Sprintf("anullsrc=channel_layout=stereo:sample_rate=44100:duration=%d", duration), outputFilename)
	if err := cmd.Run(); err != nil {
		logger.Error("Failed to create silence audio", "error", err)
		return err
	}
	return nil
}

func mergeAudio(audioFiles []string, outputFilename string, logger *slog.Logger) error {
	// create cache area for silence file
	rootDir, err := os.Getwd()
	if err != nil {
		return err
	}
	cacheDir := filepath.Join(rootDir, "cache")
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		if err := os.MkdirAll(cacheDir, os.ModePerm); err != nil {
			return err
		}
	}
	audioDir := filepath.Join(cacheDir, "en", "audio")
	if _, err := os.Stat(audioDir); os.IsNotExist(err) {
		if err := os.MkdirAll(audioDir, os.ModePerm); err != nil {
			return err
		}
	}

	silenceFile := filepath.Join(audioDir, "silence_100ms.wav")
	if _, err := os.Stat(silenceFile); os.IsNotExist(err) {
		if err := createSilenceAudio(silenceFile, 1, logger); err != nil {
			return err
		}
	}

	// build concat file list (platform safe would be better later)
	fileList := filepath.Join(cacheDir, "files.txt")
	f, err := os.Create(fileList)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	for _, file := range audioFiles {
		if _, err := f.WriteString(fmt.Sprintf("file '%s'\nfile '%s'\n", file, silenceFile)); err != nil {
			return err
		}
	}

	cmd := exec.Command("ffmpeg", "-y", "-f", "concat", "-safe", "0", "-i", fileList, "-c", "copy", outputFilename)
	if out, err := cmd.CombinedOutput(); err != nil {
		logger.Error("ffmpeg merge audio failed", "output", string(out), "error", err)
		return err
	}
	return nil
}

func wavToMp3(wavFile string, logger *slog.Logger) error {
	mp3File := strings.TrimSuffix(wavFile, filepath.Ext(wavFile)) + ".mp3"
	cmd := exec.Command("ffmpeg", "-y", "-i", wavFile, mp3File)
	if out, err := cmd.CombinedOutput(); err != nil {
		logger.Error("ffmpeg wav->mp3 failed", "output", string(out), "error", err)
		return err
	}
	return nil
}

func generateAudioOpenAI(client openai.Client, text, speechFilePath, cachedHash string) error {
	sum := sha256.Sum256([]byte(text))
	newHash := hex.EncodeToString(sum[:])
	if newHash == cachedHash {
		return nil
	}

	response, err := client.Audio.Speech.New(
		context.Background(),
		openai.AudioSpeechNewParams{
			Model:          openai.SpeechModelTTS1HD,
			Input:          text,
			Voice:          openai.AudioSpeechNewParamsVoice("onyx"),
			ResponseFormat: openai.AudioSpeechNewParamsResponseFormatMP3,
		},
	)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()

	f, err := os.Create(speechFilePath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	if _, err := io.Copy(f, response.Body); err != nil {
		return err
	}
	return nil
}

func generateAudioLocalTTS(text, speechFilePath, audioModel string, logger *slog.Logger) error {
	seed := rnd.Intn(1<<32 - 1)
	audioFiles := []string{}
	for _, textPart := range splitText(text) {
		audioFiles = append(audioFiles, generateAudioLocal(textPart, audioModel, speechFilePath, seed, logger))
	}
	if err := mergeAudio(audioFiles, speechFilePath, logger); err != nil {
		return err
	}
	if err := wavToMp3(speechFilePath, logger); err != nil {
		return err
	}
	return nil
}

func (t *Texts) generateAudios(client openai.Client, audioModel AudioGenerationModel, logger *slog.Logger) (map[string][]string, error) {
	audiosLangToPath := make(map[string][]string)

	for _, text := range t.Texts {
		audioDir := filepath.Join(text.DataDir, "cache", text.Lang, "audio")
		if _, err := os.Stat(audioDir); os.IsNotExist(err) {
			if err := os.MkdirAll(audioDir, os.ModePerm); err != nil {
				return nil, err
			}
		}
		cachedHashes := text.generateCacheHashes(audioDir)
		currentHashes := text.Hashes
		hashFile, err := os.Create(filepath.Join(audioDir, "hashes"))
		if err != nil {
			return nil, err
		}
		defer func() { _ = hashFile.Close() }()
		writer := bufio.NewWriter(hashFile)

		results := make([]string, len(currentHashes))
		for j, currentHash := range currentHashes {
			var cachedHash string
			if j < len(cachedHashes) {
				cachedHash = cachedHashes[j]
			} else {
				cachedHash = ""
			}
			if _, err := writer.WriteString(currentHash + "\n"); err != nil {
				return nil, err
			}

			audioPath := filepath.Join(audioDir, fmt.Sprintf("%d.mp3", j))
			if currentHash == cachedHash {
				results[j] = audioPath
				continue
			}

			if audioModel == OPENAI_TTS {
				if err := generateAudioOpenAI(client, text.SlidesText[j].Text, audioPath, cachedHash); err != nil {
					return nil, err
				}
			} else if audioModel == GOOGLE_TTS {
				// not implemented
				continue
			} else {
				if err := generateAudioLocalTTS(text.SlidesText[j].Text, audioPath, string(audioModel), logger); err != nil {
					return nil, err
				}
			}
			results[j] = audioPath
		}

		if err := writer.Flush(); err != nil {
			return nil, err
		}
		audiosLangToPath[text.Lang] = results
	}
	return audiosLangToPath, nil
}

func getImageDimensions(imagePath string) (int, int, error) {
	cmd := exec.Command("ffmpeg", "-i", imagePath, "-vf", "scale", "-vframes", "1", "-f", "null", "-")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, 0, fmt.Errorf("error running ffmpeg command: %v", err)
	}

	outputStr := string(output)
	re := regexp.MustCompile(`(\d+)x(\d+)`)
	matches := re.FindStringSubmatch(outputStr)
	if len(matches) < 3 {
		return 0, 0, fmt.Errorf("error parsing dimensions from ffmpeg output")
	}

	var width, height int
	n, err := fmt.Sscanf(matches[0], "%dx%d", &width, &height)
	if err != nil || n != 2 {
		return 0, 0, fmt.Errorf("error scanning dimensions: %v", err)
	}

	return width, height, nil
}

func buildFFmpegConcatCommand(videoFiles []string, finalOutput string) *exec.Cmd {
	args := []string{"-y"}

	for _, video := range videoFiles {
		args = append(args, "-i", video)
	}

	filterComplex := strings.Builder{}
	for i := range videoFiles {
		filterComplex.WriteString(fmt.Sprintf("[%d:v][%d:a]", i, i))
	}
	filterComplex.WriteString(fmt.Sprintf("concat=n=%d:v=1:a=1[outv][outa]", len(videoFiles)))

	args = append(args, "-filter_complex", filterComplex.String())
	args = append(args, "-map", "[outv]", "-map", "[outa]", finalOutput)

	return exec.Command("ffmpeg", args...)
}

// AudioRequest represents the request body for audio generation
type AudioRequest struct {
	ModelChoice       string   `json:"model_choice"`
	Text              string   `json:"text"`
	Language          string   `json:"language"`
	SpeakerAudio      string   `json:"speaker_audio"`
	PrefixAudio       string   `json:"prefix_audio"`
	E1                float64  `json:"e1"`
	E2                float64  `json:"e2"`
	E3                float64  `json:"e3"`
	E4                float64  `json:"e4"`
	E5                float64  `json:"e5"`
	E6                float64  `json:"e6"`
	E7                float64  `json:"e7"`
	E8                float64  `json:"e8"`
	VQSingle          float64  `json:"vq_single"`
	Fmax              int      `json:"fmax"`
	PitchStd          int      `json:"pitch_std"`
	SpeakingRate      int      `json:"speaking_rate"`
	DnsmosOvrl        int      `json:"dnsmos_ovrl"`
	SpeakerNoised     bool     `json:"speaker_noised"`
	CfgScale          int      `json:"cfg_scale"`
	TopP              int      `json:"top_p"`
	TopK              int      `json:"top_k"`
	MinP              int      `json:"min_p"`
	Linear            float64  `json:"linear"`
	Confidence        int      `json:"confidence"`
	Quadratic         int      `json:"quadratic"`
	Seed              int      `json:"seed"`
	RandomizeSeed     bool     `json:"randomize_seed"`
	UnconditionalKeys []string `json:"unconditional_keys"`
}

// AudioResponse represents the response body for audio generation
type AudioResponse struct {
	AudioBase64 string `json:"audio_base64"`
	SampleRate  int    `json:"sample_rate"`
	Seed        int    `json:"seed"`
}

// Helper functions
func isDirEmpty(dir string) (bool, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	return len(files) == 0, nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func remove(slice []string, item string) []string {
	for i, s := range slice {
		if s == item {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

// Run is the entrypoint for the application. It returns an error instead of exiting.
func Run() error {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	lang := flag.String("lang", "en", "Language of the text input (default: English, options are language codes and 'AI')")
	langsOut := flag.String("langs-out", "en", "Language of the text output (default: English, options can be multiple selections)")
	audioModelFlag := flag.String("audio-model", "openai", "Audio model to use: openai or local/http-host")
	flag.Parse()

	audioModel := AudioGenerationModel(*audioModelFlag)
	rootDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	openaiClient := openai.NewClient()

	dataDockerDir := filepath.Join(rootDir, "data_docker")
	dataDir := filepath.Join(rootDir, "data")
	if _, err := os.Stat(dataDockerDir); err != nil {
		empty, err := isDirEmpty(dataDir)
		if empty || err != nil {
			return fmt.Errorf("data directory does not exist or is empty")
		}
	}

	dataDirExists := false
	if _, err := os.Stat(dataDir); err == nil {
		dataDirExists = true
	}
	empty, err := isDirEmpty(dataDir)
	if err != nil {
		return fmt.Errorf("failed to check data directory: %w", err)
	}
	if dataDirExists && empty {
		items, err := os.ReadDir(dataDockerDir)
		if err != nil {
			return fmt.Errorf("failed to read data_docker: %w", err)
		}
		for _, item := range items {
			sourcePath := filepath.Join(dataDockerDir, item.Name())
			destPath := filepath.Join(dataDir, item.Name())

			if item.IsDir() {
				if _, err := os.Stat(destPath); os.IsNotExist(err) {
					if err := copyTree(sourcePath, destPath, false); err != nil {
						return fmt.Errorf("failed to copy directory: %w", err)
					}
				}
			} else {
				if err := copyFile(sourcePath, destPath, false); err != nil {
					return fmt.Errorf("failed to copy file: %w", err)
				}
			}
		}
	}

	textsFile := filepath.Join(dataDir, "texts.txt")
	if _, err := os.Stat(textsFile); err != nil {
		return fmt.Errorf("text file does not exist in data directory")
	}

	langsOutList := strings.Split(*langsOut, ",")
	if !contains(langsOutList, *lang) {
		langsOutList = append([]string{*lang}, langsOutList...)
	} else {
		langsOutList = remove(langsOutList, *lang)
		langsOutList = append([]string{*lang}, langsOutList...)
	}

	texts := Texts{
		LangIn:   *lang,
		LangsOut: langsOutList,
		DataDir:  dataDir,
		RootDir:  rootDir,
	}
	if err := texts.generateTexts(openaiClient, logger); err != nil {
		return err
	}
	audios, err := texts.generateAudios(openaiClient, audioModel, logger)
	if err != nil {
		return err
	}

	slides := []string{}
	slidesDir := filepath.Join(dataDir, "slides")
	if _, err := os.Stat(slidesDir); os.IsNotExist(err) {
		if err := os.MkdirAll(slidesDir, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create slides dir: %w", err)
		}
	}
	files, err := os.ReadDir(slidesDir)
	if err != nil {
		return fmt.Errorf("failed to read slides dir: %w", err)
	}
	for _, file := range files {
		if strings.ToLower(filepath.Ext(file.Name())) == ".png" ||
			strings.ToLower(filepath.Ext(file.Name())) == ".jpg" ||
			strings.ToLower(filepath.Ext(file.Name())) == ".jpeg" {
			slides = append(slides, filepath.Join(slidesDir, file.Name()))
		}
	}

	if len(slides) == 0 {
		return fmt.Errorf("no slides found in slides directory")
	}

	firstSlide := slides[0]
	width, height, err := getImageDimensions(firstSlide)
	if err != nil {
		return fmt.Errorf("failed to get image dimensions: %w", err)
	}
	if width%2 != 0 {
		width--
	}
	if height%2 != 0 {
		height--
	}
	cacheDir := filepath.Join(dataDir, "cache")
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		if err := os.MkdirAll(cacheDir, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create cache dir: %w", err)
		}
	}

	for audioLang, audioList := range audios {
		langDir := filepath.Join(cacheDir, audioLang)
		if _, err := os.Stat(langDir); os.IsNotExist(err) {
			if err := os.MkdirAll(langDir, os.ModePerm); err != nil {
				return fmt.Errorf("failed to create lang dir: %w", err)
			}
		}
		videoDir := filepath.Join(langDir, "videos")
		if _, err := os.Stat(videoDir); os.IsNotExist(err) {
			if err := os.MkdirAll(videoDir, os.ModePerm); err != nil {
				return fmt.Errorf("failed to create video dir: %w", err)
			}
		}

		if len(slides) != len(audioList) {
			return fmt.Errorf("slide and audio count mismatch for %s", audioLang)
		}

		var videoFiles []string

		for i := 0; i < len(slides); i++ {
			slide := slides[i]
			audio := audioList[i]
			outputVideo := fmt.Sprintf("slide%d_video_%s.mp4", i+1, audioLang)
			outputVideoPath := filepath.Join(videoDir, outputVideo)

			iw, ih, err := getImageDimensions(slide)
			if err != nil {
				return fmt.Errorf("failed to get dimensions for slide %s: %w", slide, err)
			}

			scaleFilter := fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease", width, height)
			padFilter := fmt.Sprintf("pad=%d:%d:(ow-iw)/2:(oh-ih)/2,setsar=1", width, height)
			filterComplex := fmt.Sprintf("%s,%s", scaleFilter, padFilter)

			var cmd *exec.Cmd
			if width != iw || height != ih {
				cmd = exec.Command("ffmpeg", "-loop", "1", "-i", slide, "-i", audio, "-vf", filterComplex, "-c:v", "libx264", "-tune", "stillimage", "-c:a", "mp3", "-b:a", "192k", "-pix_fmt", "yuv420p", "-shortest", outputVideoPath)
			} else {
				cmd = exec.Command("ffmpeg", "-loop", "1", "-i", slide, "-i", audio, "-c:v", "libx264", "-tune", "stillimage", "-c:a", "mp3", "-b:a", "192k", "-pix_fmt", "yuv420p", "-shortest", outputVideoPath)
			}
			var stderr bytes.Buffer
			cmd.Stderr = &stderr

			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to generate video for slide %s: %v: %s", slide, err, stderr.String())
			}

			videoFiles = append(videoFiles, outputVideoPath)
		}

		outputDir := filepath.Join(dataDir, "out")
		if _, err := os.Stat(outputDir); os.IsNotExist(err) {
			if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
				return fmt.Errorf("failed to create out dir: %w", err)
			}
		}
		finalOutput := filepath.Join(outputDir, fmt.Sprintf("output-%s.mp4", audioLang))

		concatCmd := buildFFmpegConcatCommand(videoFiles, finalOutput)
		if out, err := concatCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to concatenate videos: %v: %s", err, string(out))
		}
	}

	return nil
}
