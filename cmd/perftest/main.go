package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gocreator/internal/adapters"
	"gocreator/internal/interfaces"
	"gocreator/internal/services"

	"github.com/joho/godotenv"
	"github.com/openai/openai-go/v3"
	"github.com/spf13/afero"
)

// PerfTestResult holds performance metrics for a single operation
type PerfTestResult struct {
	Operation   string
	CacheStatus string
	Duration    time.Duration
	Iterations  int
	AvgDuration time.Duration
}

// PerfTestResults holds all performance test results
type PerfTestResults struct {
	Results     []PerfTestResult
	E2EDuration time.Duration
	E2ECacheDur time.Duration
	CacheHits   int
	TotalOps    int
}

// Logger for performance testing
type perfLogger struct {
	cacheHits int
	prefix    string
}

func newPerfLogger(prefix string) *perfLogger {
	return &perfLogger{prefix: prefix}
}

func (l *perfLogger) Info(msg string, args ...any) {
	if strings.Contains(msg, "cached") || strings.Contains(msg, "Using cached") || strings.Contains(msg, "Loading cached") {
		l.cacheHits++
		fmt.Printf("[%s] CACHE HIT: %s", l.prefix, msg)
		for i := 0; i < len(args); i += 2 {
			if i+1 < len(args) {
				fmt.Printf(" %v=%v", args[i], args[i+1])
			}
		}
		fmt.Println()
	}
}

func (l *perfLogger) Warn(msg string, args ...any) {}
func (l *perfLogger) Error(msg string, args ...any) {
	fmt.Printf("[%s] ERROR: %s", l.prefix, msg)
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			fmt.Printf(" %v=%v", args[i], args[i+1])
		}
	}
	fmt.Println()
}
func (l *perfLogger) Debug(msg string, args ...any)      {}
func (l *perfLogger) With(args ...any) interfaces.Logger { return l }

func main() {
	results := &PerfTestResults{
		Results: make([]PerfTestResult, 0),
	}

	fmt.Println("GoCreator Performance Testing Tool")
	fmt.Println("===================================")
	fmt.Println()

	if err := godotenv.Load(); err != nil {
		if !os.IsNotExist(err) {
			_, _ = fmt.Fprintf(os.Stderr, "Warning: failed to load .env: %v\n", err)
		}
	}

	// Check for OpenAI API key
	apiKey := os.Getenv("OPENAI_API_KEY")
	useRealAPI := apiKey != ""

	if !useRealAPI {
		fmt.Println("⚠️  OPENAI_API_KEY not set - running in simulation mode")
		fmt.Println("   Set OPENAI_API_KEY environment variable to test with real API calls")
		fmt.Println()
	}

	// Setup test environment
	tempDir, err := os.MkdirTemp("", "gocreator-perftest-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create temp directory: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	fs := afero.NewOsFs()

	// Create test data
	dataDir := filepath.Join(tempDir, "data")
	slidesDir := filepath.Join(dataDir, "slides")
	if err := fs.MkdirAll(slidesDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create directories: %v\n", err)
		os.Exit(1)
	}

	// Create dummy slides (we'll use text files as placeholders since we can't create real images easily)
	testTexts := []string{
		"Welcome to GoCreator performance testing",
		"This is the second slide with some content",
		"Third slide demonstrates caching capabilities",
	}

	// Create test slides (simple text files as placeholders)
	for i := 0; i < len(testTexts); i++ {
		slidePath := filepath.Join(slidesDir, fmt.Sprintf("%d.txt", i))
		if err := afero.WriteFile(fs, slidePath, []byte(fmt.Sprintf("Slide %d", i)), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create test slide: %v\n", err)
			os.Exit(1)
		}
	}

	// Create texts file
	textsPath := filepath.Join(dataDir, "texts.txt")
	textsContent := strings.Join(testTexts, "\n-\n")
	if err := afero.WriteFile(fs, textsPath, []byte(textsContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create texts file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Test Setup:")
	fmt.Printf("  - Slides: %d\n", len(testTexts))
	fmt.Printf("  - Temp Directory: %s\n", tempDir)
	fmt.Println()

	// Run tests
	ctx := context.Background()

	if useRealAPI {
		runRealAPITests(ctx, fs, dataDir, results)
	} else {
		runSimulatedTests(ctx, fs, dataDir, results)
	}

	// Print results as markdown table
	printResultsTable(results)
}

func runSimulatedTests(ctx context.Context, fs afero.Fs, dataDir string, results *PerfTestResults) {
	fmt.Println("Running simulated performance tests...")
	fmt.Println()

	// Test 1: Text operations
	testTextOperations(ctx, fs, dataDir, results)

	// Test 2: Cache operations
	testCacheOperations(results)

	fmt.Println()
	fmt.Println("✓ Simulated tests completed")
	fmt.Println("  Note: To test translation and audio with real API, set OPENAI_API_KEY")
}

func runRealAPITests(ctx context.Context, fs afero.Fs, dataDir string, results *PerfTestResults) {
	fmt.Println("Running performance tests with real OpenAI API...")
	fmt.Println()

	// Create services
	logger := newPerfLogger("test")
	textService := services.NewTextService(fs, logger)

	openaiClient := openai.NewClient()
	client := adapters.NewOpenAIAdapter(openaiClient)

	audioService := services.NewAudioService(fs, client, textService, logger)
	
	// Create translation service with disk cache
	translationCacheDir := filepath.Join(dataDir, "cache", "translations")
	translationService := services.NewTranslationServiceWithCache(client, logger, fs, translationCacheDir)
	
	videoService := services.NewVideoService(fs, logger)

	// Test without cache
	fmt.Println("Test 1: Operations WITHOUT cache")
	fmt.Println("----------------------------------")

	testTexts := []string{
		"Welcome to GoCreator performance testing",
		"This is the second slide with some content",
		"Third slide demonstrates caching capabilities",
	}

	// Measure translation
	start := time.Now()
	logger.cacheHits = 0
	translatedTexts, err := translationService.TranslateBatch(ctx, testTexts, "Spanish")
	translationDur := time.Since(start)
	if err != nil {
		fmt.Printf("  Translation error: %v\n", err)
	} else {
		results.Results = append(results.Results, PerfTestResult{
			Operation:   "Translation API (3 texts)",
			CacheStatus: "No Cache",
			Duration:    translationDur,
			Iterations:  len(testTexts),
			AvgDuration: translationDur / time.Duration(len(testTexts)),
		})
		fmt.Printf("  ✓ Translation: %v (avg: %v per text)\n", translationDur, translationDur/time.Duration(len(testTexts)))
	}

	// Measure audio generation
	start = time.Now()
	audioDir := filepath.Join(dataDir, "cache", "es", "audio")
	audioPaths, err := audioService.GenerateBatch(ctx, translatedTexts, audioDir)
	audioDur := time.Since(start)
	if err != nil {
		fmt.Printf("  Audio generation error: %v\n", err)
	} else {
		results.Results = append(results.Results, PerfTestResult{
			Operation:   "Audio Generation (3 files)",
			CacheStatus: "No Cache",
			Duration:    audioDur,
			Iterations:  len(audioPaths),
			AvgDuration: audioDur / time.Duration(len(audioPaths)),
		})
		fmt.Printf("  ✓ Audio Generation: %v (avg: %v per file)\n", audioDur, audioDur/time.Duration(len(audioPaths)))
	}

	results.E2EDuration = translationDur + audioDur

	fmt.Println()
	fmt.Println("Test 2: Operations WITH cache")
	fmt.Println("------------------------------")

	// Save translation to cache for next run
	_ = textService.Save(ctx, filepath.Join(dataDir, "cache", "es", "text", "texts.txt"), translatedTexts)

	// Measure translation with cache
	start = time.Now()
	logger.cacheHits = 0
	_, err = translationService.TranslateBatch(ctx, testTexts, "Spanish")
	cachedTranslationDur := time.Since(start)
	if err != nil {
		fmt.Printf("  Translation error: %v\n", err)
	} else {
		results.Results = append(results.Results, PerfTestResult{
			Operation:   "Translation API (3 texts)",
			CacheStatus: "With Cache",
			Duration:    cachedTranslationDur,
			Iterations:  len(testTexts),
			AvgDuration: cachedTranslationDur / time.Duration(len(testTexts)),
		})
		fmt.Printf("  ✓ Translation (no API call): %v\n", cachedTranslationDur)
	}

	// Measure audio generation with cache
	start = time.Now()
	cachedAudioPaths, err := audioService.GenerateBatch(ctx, translatedTexts, audioDir)
	cachedAudioDur := time.Since(start)
	if err != nil {
		fmt.Printf("  Audio generation error: %v\n", err)
	} else {
		results.Results = append(results.Results, PerfTestResult{
			Operation:   "Audio Generation (3 files)",
			CacheStatus: "With Cache",
			Duration:    cachedAudioDur,
			Iterations:  len(cachedAudioPaths),
			AvgDuration: cachedAudioDur / time.Duration(len(cachedAudioPaths)),
		})
		fmt.Printf("  ✓ Audio Generation (cache hit): %v\n", cachedAudioDur)
		fmt.Printf("  ✓ Cache hits: %d\n", logger.cacheHits)
	}

	results.E2ECacheDur = cachedTranslationDur + cachedAudioDur
	results.CacheHits = logger.cacheHits
	results.TotalOps = len(testTexts) * 2

	// Test video concatenation (FFmpeg combine audio + slides)
	fmt.Println()
	fmt.Println("Test 3: Video Concatenation")
	fmt.Println("----------------------------")
	
	// Create test slides (simple text files as placeholders)
	slidesDir := filepath.Join(dataDir, "slides")
	_ = fs.MkdirAll(slidesDir, 0755)
	testSlides := make([]string, len(testTexts))
	for i := range testTexts {
		slidePath := filepath.Join(slidesDir, fmt.Sprintf("slide_%d.txt", i))
		_ = afero.WriteFile(fs, slidePath, []byte(fmt.Sprintf("Slide %d content", i)), 0644)
		testSlides[i] = slidePath
	}
	
	// Measure video generation from slides + audio
	outputPath := filepath.Join(dataDir, "out", "test_video.mp4")
	start = time.Now()
	err = videoService.GenerateFromSlides(ctx, testSlides, audioPaths, outputPath)
	videoConcatDur := time.Since(start)
	if err != nil {
		fmt.Printf("  Video concatenation error: %v (FFmpeg may not be available in test environment)\n", err)
	} else {
		results.Results = append(results.Results, PerfTestResult{
			Operation:   "Video Concatenation (3 segments)",
			CacheStatus: "N/A",
			Duration:    videoConcatDur,
			Iterations:  len(testSlides),
			AvgDuration: videoConcatDur / time.Duration(len(testSlides)),
		})
		fmt.Printf("  ✓ Video Concatenation: %v (avg: %v per segment)\n", videoConcatDur, videoConcatDur/time.Duration(len(testSlides)))
	}

	fmt.Println()
	fmt.Println("✓ Real API tests completed")
}

func testTextOperations(ctx context.Context, fs afero.Fs, dataDir string, results *PerfTestResults) {
	logger := newPerfLogger("text")
	textService := services.NewTextService(fs, logger)

	textsPath := filepath.Join(dataDir, "texts.txt")

	// Test load
	start := time.Now()
	iterations := 1000
	for i := 0; i < iterations; i++ {
		_, _ = textService.Load(ctx, textsPath)
	}
	loadDur := time.Since(start)
	results.Results = append(results.Results, PerfTestResult{
		Operation:   "Text Load",
		CacheStatus: "N/A",
		Duration:    loadDur,
		Iterations:  iterations,
		AvgDuration: loadDur / time.Duration(iterations),
	})
	fmt.Printf("  ✓ Text Load: %v (%d iterations, avg: %v)\n", loadDur, iterations, loadDur/time.Duration(iterations))

	// Test hash
	start = time.Now()
	text := "This is a test text for hashing performance measurement"
	for i := 0; i < iterations; i++ {
		_ = textService.Hash(text)
	}
	hashDur := time.Since(start)
	results.Results = append(results.Results, PerfTestResult{
		Operation:   "Text Hash",
		CacheStatus: "N/A",
		Duration:    hashDur,
		Iterations:  iterations,
		AvgDuration: hashDur / time.Duration(iterations),
	})
	fmt.Printf("  ✓ Text Hash: %v (%d iterations, avg: %v)\n", hashDur, iterations, hashDur/time.Duration(iterations))
}

func testCacheOperations(results *PerfTestResults) {
	cacheService := services.NewCacheService(5*time.Minute, 10*time.Minute)

	iterations := 10000

	// Test set
	start := time.Now()
	for i := 0; i < iterations; i++ {
		cacheService.Set(fmt.Sprintf("key_%d", i), "value")
	}
	setDur := time.Since(start)
	results.Results = append(results.Results, PerfTestResult{
		Operation:   "Cache Set",
		CacheStatus: "N/A",
		Duration:    setDur,
		Iterations:  iterations,
		AvgDuration: setDur / time.Duration(iterations),
	})
	fmt.Printf("  ✓ Cache Set: %v (%d iterations, avg: %v)\n", setDur, iterations, setDur/time.Duration(iterations))

	// Test get (hit) - access first 1000 keys repeatedly (90% hit rate pattern)
	// This simulates a realistic cache usage where some keys are accessed more frequently
	start = time.Now()
	for i := 0; i < iterations; i++ {
		_, _ = cacheService.Get(fmt.Sprintf("key_%d", i%1000))
	}
	getDur := time.Since(start)
	results.Results = append(results.Results, PerfTestResult{
		Operation:   "Cache Get (90% hit rate)",
		CacheStatus: "Hit",
		Duration:    getDur,
		Iterations:  iterations,
		AvgDuration: getDur / time.Duration(iterations),
	})
	fmt.Printf("  ✓ Cache Get (90%% hit rate): %v (%d iterations, avg: %v)\n", getDur, iterations, getDur/time.Duration(iterations))

	// Test get (miss)
	start = time.Now()
	for i := 0; i < iterations; i++ {
		_, _ = cacheService.Get(fmt.Sprintf("nonexistent_%d", i))
	}
	getMissDur := time.Since(start)
	results.Results = append(results.Results, PerfTestResult{
		Operation:   "Cache Get (miss)",
		CacheStatus: "Miss",
		Duration:    getMissDur,
		Iterations:  iterations,
		AvgDuration: getMissDur / time.Duration(iterations),
	})
	fmt.Printf("  ✓ Cache Get (miss): %v (%d iterations, avg: %v)\n", getMissDur, iterations, getMissDur/time.Duration(iterations))
}

func printResultsTable(results *PerfTestResults) {
	fmt.Println()
	fmt.Println("Performance Test Results")
	fmt.Println("========================")
	fmt.Println()
	fmt.Println("| Operation | Cache Status | Total Duration | Iterations | Avg Duration |")
	fmt.Println("|-----------|--------------|----------------|------------|--------------|")

	for _, result := range results.Results {
		fmt.Printf("| %s | %s | %v | %d | %v |\n",
			padRight(result.Operation, 30),
			padRight(result.CacheStatus, 12),
			padRight(result.Duration.String(), 14),
			result.Iterations,
			result.AvgDuration)
	}

	if results.E2EDuration > 0 {
		fmt.Println()
		fmt.Println("End-to-End Performance:")
		fmt.Printf("| E2E Without Cache | %v |\n", results.E2EDuration)
		fmt.Printf("| E2E With Cache    | %v |\n", results.E2ECacheDur)
		if results.E2EDuration > 0 && results.E2ECacheDur > 0 {
			speedup := float64(results.E2EDuration) / float64(results.E2ECacheDur)
			fmt.Printf("| Speedup Factor    | %.2fx |\n", speedup)
		}
		fmt.Printf("| Cache Hit Count   | %d / %d operations |\n", results.CacheHits, results.TotalOps)
	}

	fmt.Println()
}

func padRight(s string, length int) string {
	if len(s) >= length {
		return s
	}
	return s + strings.Repeat(" ", length-len(s))
}
