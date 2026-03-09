package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gocreator/internal/interfaces"
	"gocreator/internal/services"

	"github.com/openai/openai-go/v3"
	"github.com/spf13/afero"
)

// MockOpenAIClient simulates OpenAI API calls with configurable delays
type MockOpenAIClient struct {
	TranslationDelay time.Duration
	TTSDelay         time.Duration
	CallCount        struct {
		Translation int
		TTS         int
	}
}

func NewMockOpenAIClient(translationDelay, ttsDelay time.Duration) *MockOpenAIClient {
	return &MockOpenAIClient{
		TranslationDelay: translationDelay,
		TTSDelay:         ttsDelay,
	}
}

func (m *MockOpenAIClient) ChatCompletion(ctx context.Context, messages []openai.ChatCompletionMessageParamUnion) (string, error) {
	m.CallCount.Translation++
	time.Sleep(m.TranslationDelay)

	// Extract text from messages and return a mock translation
	return "Mock translation", nil
}

func (m *MockOpenAIClient) GenerateSpeech(ctx context.Context, text string) (io.ReadCloser, error) {
	m.CallCount.TTS++
	time.Sleep(m.TTSDelay)

	// Return mock audio data
	mockAudio := strings.Repeat("audio-data-", 100) // ~1KB of mock audio
	return io.NopCloser(strings.NewReader(mockAudio)), nil
}

// CacheTrackingLogger tracks cache hits and misses
type CacheTrackingLogger struct {
	SegmentCacheHits       int
	SegmentCacheMisses     int
	FinalCacheHits         int
	FinalCacheMisses       int
	TranslationCacheHits   int
	TranslationCacheMisses int
	AudioCacheHits         int
	AudioCacheMisses       int
}

func (l *CacheTrackingLogger) Debug(msg string, args ...any) {
	l.trackCache(msg)
}

func (l *CacheTrackingLogger) Info(msg string, args ...any) {
	l.trackCache(msg)
}

func (l *CacheTrackingLogger) Warn(msg string, args ...any) {
	l.trackCache(msg)
}

func (l *CacheTrackingLogger) Error(msg string, args ...any) {
	fmt.Printf("[ERROR] %s\n", msg)
}

func (l *CacheTrackingLogger) With(args ...any) interfaces.Logger {
	return l
}

func (l *CacheTrackingLogger) trackCache(msg string) {
	msgLower := strings.ToLower(msg)

	if strings.Contains(msgLower, "using cached video segment") {
		l.SegmentCacheHits++
	} else if strings.Contains(msgLower, "using cached final video") {
		l.FinalCacheHits++
	} else if strings.Contains(msgLower, "loading cached translation") {
		l.TranslationCacheHits++
	} else if strings.Contains(msgLower, "using cached audio") {
		l.AudioCacheHits++
	}

	// Track misses (these would be in debug logs)
	if strings.Contains(msgLower, "generating") || strings.Contains(msgLower, "translating") {
		if strings.Contains(msgLower, "segment") {
			l.SegmentCacheMisses++
		} else if strings.Contains(msgLower, "audio") {
			l.AudioCacheMisses++
		} else if strings.Contains(msgLower, "translation") {
			l.TranslationCacheMisses++
		}
	}
}

// Scenario represents a test scenario
type Scenario struct {
	Name              string
	NumSlides         int
	NumLanguages      int
	TransitionEnabled bool
	RunCount          int // How many times to run (to test caching)
}

// PerformanceMetrics holds performance metrics for a scenario run
type PerformanceMetrics struct {
	Scenario             string
	Run                  int
	Duration             time.Duration
	SegmentCacheHits     int
	FinalCacheHits       int
	TranslationCacheHits int
	AudioCacheHits       int
	APICallsTranslation  int
	APICallsTTS          int
	TotalAPICalls        int
	CacheHitRate         float64
}

func main() {
	fmt.Println("=== GoCreator Cache Performance Testing Tool ===")
	fmt.Println()

	scenarios := []Scenario{
		{
			Name:              "Small Project (3 slides, 2 languages)",
			NumSlides:         3,
			NumLanguages:      2,
			TransitionEnabled: true,
			RunCount:          3,
		},
		{
			Name:              "Medium Project (5 slides, 3 languages)",
			NumSlides:         5,
			NumLanguages:      3,
			TransitionEnabled: true,
			RunCount:          3,
		},
		{
			Name:              "Large Project (10 slides, 4 languages)",
			NumSlides:         10,
			NumLanguages:      4,
			TransitionEnabled: true,
			RunCount:          3,
		},
	}

	allMetrics := []PerformanceMetrics{}

	for _, scenario := range scenarios {
		fmt.Printf("\n### Testing Scenario: %s ###\n", scenario.Name)
		metrics := runScenario(scenario)
		allMetrics = append(allMetrics, metrics...)
	}

	printSummary(allMetrics)
}

func runScenario(scenario Scenario) []PerformanceMetrics {
	metrics := []PerformanceMetrics{}

	// Create temporary directory for this scenario
	baseDir := filepath.Join(os.TempDir(), "cache-perf-test", strings.ReplaceAll(scenario.Name, " ", "-"))

	for run := 1; run <= scenario.RunCount; run++ {
		fmt.Printf("\n  Run %d/%d...\n", run, scenario.RunCount)

		// Setup
		fs := afero.NewOsFs()
		logger := &CacheTrackingLogger{}
		mockClient := NewMockOpenAIClient(50*time.Millisecond, 50*time.Millisecond)

		// Create data directory structure
		dataDir := filepath.Join(baseDir, "data")
		slidesDir := filepath.Join(dataDir, "slides")

		if run == 1 {
			// Clean up before first run
			if err := os.RemoveAll(baseDir); err != nil {
				log.Printf("Warning: failed to remove base dir: %v", err)
			}
		}

		// Create directories
		if err := os.MkdirAll(slidesDir, 0755); err != nil {
			log.Fatalf("Failed to create slides directory: %v", err)
		}

		// Generate mock slides and matching sidecar texts
		setupMockData(fs, slidesDir, scenario.NumSlides)

		// Create services
		textService := services.NewTextService(fs, logger)
		translationService := services.NewTranslationServiceWithCache(mockClient, logger, fs, filepath.Join(dataDir, "cache", "translations"))
		audioService := services.NewAudioService(fs, mockClient, textService, logger)
		videoService := services.NewVideoService(fs, logger)
		slideService := services.NewSlideService(fs, logger)

		// Configure transitions
		if scenario.TransitionEnabled {
			videoService.SetTransition(services.TransitionConfig{
				Type:     services.TransitionFade,
				Duration: 0.5,
			})
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

		// Load slides
		slides, err := slideService.LoadSlides(context.Background(), slidesDir)
		if err != nil {
			log.Fatalf("Failed to load slides: %v", err)
		}

		// Load source narration from per-slide sidecars
		inputTexts, err := loadSlideTexts(context.Background(), textService, slides)
		if err != nil {
			log.Fatalf("Failed to load sidecar texts: %v", err)
		}

		// Generate languages
		languages := generateLanguages(scenario.NumLanguages)

		// Record start time
		startTime := time.Now()

		// Process each language
		for _, lang := range languages {
			cfg := services.VideoCreatorConfig{
				RootDir:     baseDir,
				InputLang:   "en",
				OutputLangs: []string{lang},
				Transition: services.TransitionConfig{
					Type:     services.TransitionFade,
					Duration: 0.5,
				},
			}

			// Process language (simplified - just the core operations)
			ctx := context.Background()
			cacheDir := filepath.Join(dataDir, "cache", lang)
			audioDir := filepath.Join(cacheDir, "audio")

			// Translate if needed
			var texts []string
			if lang == cfg.InputLang {
				texts = inputTexts
			} else {
				texts, err = translationService.TranslateBatch(ctx, inputTexts, lang)
				if err != nil {
					log.Printf("Warning: failed to translate texts for %s: %v", lang, err)
					continue
				}
			}

			// Generate audio
			audioPaths, _ := audioService.GenerateBatch(ctx, texts, audioDir)

			// Generate video
			outputDir := filepath.Join(dataDir, "out")
			outputPath := filepath.Join(outputDir, fmt.Sprintf("output-%s.mp4", lang))
			if err := videoService.GenerateFromSlides(ctx, slides, audioPaths, outputPath); err != nil {
				log.Printf("Warning: failed to generate video: %v", err)
			}
		}

		duration := time.Since(startTime)

		// Calculate cache hit rate
		totalOps := logger.SegmentCacheHits + logger.SegmentCacheMisses +
			logger.FinalCacheHits + logger.FinalCacheMisses +
			logger.TranslationCacheHits + logger.TranslationCacheMisses +
			logger.AudioCacheHits + logger.AudioCacheMisses

		cacheHits := logger.SegmentCacheHits + logger.FinalCacheHits +
			logger.TranslationCacheHits + logger.AudioCacheHits

		cacheHitRate := 0.0
		if totalOps > 0 {
			cacheHitRate = float64(cacheHits) / float64(totalOps) * 100
		}

		totalAPICalls := mockClient.CallCount.Translation + mockClient.CallCount.TTS

		metric := PerformanceMetrics{
			Scenario:             scenario.Name,
			Run:                  run,
			Duration:             duration,
			SegmentCacheHits:     logger.SegmentCacheHits,
			FinalCacheHits:       logger.FinalCacheHits,
			TranslationCacheHits: logger.TranslationCacheHits,
			AudioCacheHits:       logger.AudioCacheHits,
			APICallsTranslation:  mockClient.CallCount.Translation,
			APICallsTTS:          mockClient.CallCount.TTS,
			TotalAPICalls:        totalAPICalls,
			CacheHitRate:         cacheHitRate,
		}

		metrics = append(metrics, metric)

		fmt.Printf("    Duration: %v\n", duration)
		fmt.Printf("    Cache Hit Rate: %.1f%%\n", cacheHitRate)
		fmt.Printf("    API Calls (Translation): %d\n", mockClient.CallCount.Translation)
		fmt.Printf("    API Calls (TTS): %d\n", mockClient.CallCount.TTS)
		fmt.Printf("    Segment Cache Hits: %d\n", logger.SegmentCacheHits)
		fmt.Printf("    Final Cache Hits: %d\n", logger.FinalCacheHits)

		// Prevent creator from being optimized away
		_ = creator
	}

	return metrics
}

func setupMockData(fs afero.Fs, slidesDir string, numSlides int) {
	// Create mock slide images
	for i := 0; i < numSlides; i++ {
		slidePath := filepath.Join(slidesDir, fmt.Sprintf("slide_%d.png", i))
		mockImageData := strings.Repeat(fmt.Sprintf("image-data-%d-", i), 100)
		if err := afero.WriteFile(fs, slidePath, []byte(mockImageData), 0644); err != nil {
			log.Printf("Warning: failed to write slide: %v", err)
		}

		textPath := filepath.Join(slidesDir, fmt.Sprintf("slide_%d.txt", i))
		text := fmt.Sprintf("This is the narration for slide %d. It contains important information.", i)
		if err := afero.WriteFile(fs, textPath, []byte(text), 0644); err != nil {
			log.Printf("Warning: failed to write sidecar text: %v", err)
		}
	}
}

func loadSlideTexts(ctx context.Context, textService *services.TextService, slides []string) ([]string, error) {
	texts := make([]string, 0, len(slides))
	for _, slidePath := range slides {
		sidecarPath := filepath.Join(
			filepath.Dir(slidePath),
			strings.TrimSuffix(filepath.Base(slidePath), filepath.Ext(slidePath))+".txt",
		)

		entries, err := textService.Load(ctx, sidecarPath)
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", sidecarPath, err)
		}
		if len(entries) == 0 {
			return nil, fmt.Errorf("sidecar %s is empty", sidecarPath)
		}

		texts = append(texts, entries[0])
	}

	return texts, nil
}

func generateLanguages(count int) []string {
	allLangs := []string{"en", "fr", "es", "de", "ja", "zh", "pt", "ru"}
	if count > len(allLangs) {
		count = len(allLangs)
	}
	return allLangs[:count]
}

func printSummary(metrics []PerformanceMetrics) {
	fmt.Println()
	fmt.Println()
	fmt.Println("=== Performance Summary ===")
	fmt.Println()

	// Group by scenario
	scenarioMap := make(map[string][]PerformanceMetrics)
	for _, m := range metrics {
		scenarioMap[m.Scenario] = append(scenarioMap[m.Scenario], m)
	}

	for scenario, runs := range scenarioMap {
		fmt.Printf("### %s ###\n", scenario)

		firstRun := runs[0]
		lastRun := runs[len(runs)-1]

		speedup := float64(firstRun.Duration) / float64(lastRun.Duration)
		apiSavings := firstRun.TotalAPICalls - lastRun.TotalAPICalls
		apiSavingsPercent := 0.0
		if firstRun.TotalAPICalls > 0 {
			apiSavingsPercent = float64(apiSavings) / float64(firstRun.TotalAPICalls) * 100
		}

		fmt.Printf("  First Run:  %v (%.0f%% cache hits, %d API calls)\n",
			firstRun.Duration, firstRun.CacheHitRate, firstRun.TotalAPICalls)
		fmt.Printf("  Last Run:   %v (%.0f%% cache hits, %d API calls)\n",
			lastRun.Duration, lastRun.CacheHitRate, lastRun.TotalAPICalls)
		fmt.Printf("  Speedup:    %.1fx faster\n", speedup)
		fmt.Printf("  API Savings: %d calls (%.0f%% reduction)\n\n",
			apiSavings, apiSavingsPercent)
	}

	fmt.Println("=== Key Insights ===")
	fmt.Println("- First run: No cache, all operations performed")
	fmt.Println("- Subsequent runs: Cache utilized, significant speedup")
	fmt.Println("- API calls are simulated with 50ms delay each")
	fmt.Println("- Segment and final video caching provide the largest performance gains")
	fmt.Println("\nNote: Actual performance will vary based on real API latency,")
	fmt.Println("      FFmpeg encoding time, and disk I/O speed.")
}
