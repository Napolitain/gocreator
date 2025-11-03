package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sync"

	"gocreator/internal/interfaces"

	"github.com/openai/openai-go/v3"
	"github.com/spf13/afero"
)

// TranslationService handles text translation with caching
type TranslationService struct {
	client       interfaces.OpenAIClient
	logger       interfaces.Logger
	fs           afero.Fs
	memoryCache  map[string]string
	cacheMutex   sync.RWMutex
	cacheDir     string
}

// NewTranslationService creates a new translation service
func NewTranslationService(client interfaces.OpenAIClient, logger interfaces.Logger) *TranslationService {
	return &TranslationService{
		client:      client,
		logger:      logger,
		memoryCache: make(map[string]string),
	}
}

// NewTranslationServiceWithCache creates a new translation service with disk cache support
func NewTranslationServiceWithCache(client interfaces.OpenAIClient, logger interfaces.Logger, fs afero.Fs, cacheDir string) *TranslationService {
	return &TranslationService{
		client:      client,
		logger:      logger,
		fs:          fs,
		memoryCache: make(map[string]string),
		cacheDir:    cacheDir,
	}
}

// getCacheKey generates a cache key from text and target language
func (s *TranslationService) getCacheKey(text, targetLang string) string {
	data := fmt.Sprintf("%s|%s", text, targetLang)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// getFromMemoryCache retrieves a cached translation from memory
func (s *TranslationService) getFromMemoryCache(key string) (string, bool) {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()
	val, ok := s.memoryCache[key]
	return val, ok
}

// setInMemoryCache stores a translation in memory cache
func (s *TranslationService) setInMemoryCache(key, value string) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()
	s.memoryCache[key] = value
}

// getFromDiskCache retrieves a cached translation from disk
func (s *TranslationService) getFromDiskCache(key string) (string, bool) {
	if s.fs == nil || s.cacheDir == "" {
		return "", false
	}

	cachePath := filepath.Join(s.cacheDir, key+".txt")
	data, err := afero.ReadFile(s.fs, cachePath)
	if err != nil {
		return "", false
	}

	s.logger.Info("Translation cache hit (disk)", "key", key)
	return string(data), true
}

// setInDiskCache stores a translation on disk
func (s *TranslationService) setInDiskCache(key, value string) {
	if s.fs == nil || s.cacheDir == "" {
		return
	}

	if err := s.fs.MkdirAll(s.cacheDir, 0755); err != nil {
		s.logger.Warn("Failed to create cache directory", "error", err)
		return
	}

	cachePath := filepath.Join(s.cacheDir, key+".txt")
	if err := afero.WriteFile(s.fs, cachePath, []byte(value), 0644); err != nil {
		s.logger.Warn("Failed to write to disk cache", "error", err)
	}
}

// Translate translates text to target language with caching
func (s *TranslationService) Translate(ctx context.Context, text, targetLang string) (string, error) {
	// Check memory cache first
	cacheKey := s.getCacheKey(text, targetLang)
	if cached, ok := s.getFromMemoryCache(cacheKey); ok {
		s.logger.Info("Translation cache hit (memory)", "key", cacheKey)
		return cached, nil
	}

	// Check disk cache
	if cached, ok := s.getFromDiskCache(cacheKey); ok {
		// Store in memory for faster future access
		s.setInMemoryCache(cacheKey, cached)
		return cached, nil
	}

	// No cache, call API
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.UserMessage(fmt.Sprintf("Translate '%s' to %s and don't return anything else than the translation.", text, targetLang)),
	}

	translated, err := s.client.ChatCompletion(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("translation failed: %w", err)
	}

	// Cache the result
	s.setInMemoryCache(cacheKey, translated)
	s.setInDiskCache(cacheKey, translated)

	return translated, nil
}

// TranslateBatch translates multiple texts in parallel
func (s *TranslationService) TranslateBatch(ctx context.Context, texts []string, targetLang string) ([]string, error) {
	results := make([]string, len(texts))
	errors := make([]error, len(texts))
	var wg sync.WaitGroup

	for i, text := range texts {
		wg.Add(1)
		go func(idx int, txt string) {
			defer wg.Done()
			translated, err := s.Translate(ctx, txt, targetLang)
			if err != nil {
				errors[idx] = err
				return
			}
			results[idx] = translated
		}(i, text)
	}

	wg.Wait()

	// Check for any errors
	for i, err := range errors {
		if err != nil {
			return nil, fmt.Errorf("translation failed for text %d: %w", i, err)
		}
	}

	return results, nil
}
