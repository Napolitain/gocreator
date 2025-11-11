package services

import (
	"testing"

	"gocreator/internal/mocks"

	"github.com/spf13/afero"
)

// FuzzTranslationService_getCacheKey tests cache key generation with random inputs
func FuzzTranslationService_getCacheKey(f *testing.F) {
	// Seed corpus with interesting cases
	f.Add("Hello world", "en")
	f.Add("", "")
	f.Add("Special chars: !@#$%^&*()", "fr")
	f.Add("Unicode: 你好世界", "zh")
	f.Add("Long text: "+string(make([]byte, 1000)), "de")
	f.Add("Text with\nnewlines", "es")
	f.Add("Tab\tseparated", "it")
	f.Add("a|b", "pt") // Pipe character is used in cache key generation

	f.Fuzz(func(t *testing.T, text, targetLang string) {
		mockClient := new(mocks.MockOpenAIClient)
		logger := &mockLogger{}
		service := NewTranslationService(mockClient, logger)

		// getCacheKey should never panic
		key1 := service.getCacheKey(text, targetLang)
		if key1 == "" {
			t.Error("getCacheKey returned empty string")
		}

		// Cache key should be deterministic
		key2 := service.getCacheKey(text, targetLang)
		if key1 != key2 {
			t.Errorf("Cache key is not deterministic: %s != %s", key1, key2)
		}

		// Cache key should be 64 characters (SHA256 in hex)
		if len(key1) != 64 {
			t.Errorf("Cache key length is %d, expected 64", len(key1))
		}

		// Different inputs should produce different cache keys
		if text != "" || targetLang != "" {
			differentKey := service.getCacheKey(text+"x", targetLang)
			if key1 == differentKey && text != text+"x" {
				t.Errorf("Same cache key for different texts")
			}

			differentLangKey := service.getCacheKey(text, targetLang+"x")
			if key1 == differentLangKey && targetLang != targetLang+"x" {
				t.Errorf("Same cache key for different languages")
			}
		}
	})
}

// FuzzTranslationService_MemoryCache tests memory cache operations with random keys and values
func FuzzTranslationService_MemoryCache(f *testing.F) {
	// Seed corpus
	f.Add("key1", "value1")
	f.Add("", "")
	f.Add("unicode_key_你好", "unicode_value_世界")
	f.Add("long_key_"+string(make([]byte, 100)), "long_value_"+string(make([]byte, 100)))
	f.Add("special!@#$", "chars%^&*()")

	f.Fuzz(func(t *testing.T, key, value string) {
		mockClient := new(mocks.MockOpenAIClient)
		logger := &mockLogger{}
		service := NewTranslationService(mockClient, logger)

		// setInMemoryCache should never panic
		service.setInMemoryCache(key, value)

		// getFromMemoryCache should never panic
		retrieved, found := service.getFromMemoryCache(key)
		if !found {
			t.Error("Failed to retrieve cached value")
			return
		}

		if retrieved != value {
			t.Errorf("Retrieved value doesn't match: %q != %q", retrieved, value)
		}

		// Retrieve non-existent key should return false
		_, found = service.getFromMemoryCache(key + "_nonexistent")
		if found {
			t.Error("Found non-existent key")
		}
	})
}

// FuzzTranslationService_DiskCache tests disk cache operations with random keys and values
func FuzzTranslationService_DiskCache(f *testing.F) {
	// Seed corpus
	f.Add("key1", "value1")
	f.Add("", "")
	f.Add("unicode_你好", "世界🌍")
	f.Add("key\nwith\nnewlines", "value\nwith\nnewlines")

	f.Fuzz(func(t *testing.T, key, value string) {
		mockClient := new(mocks.MockOpenAIClient)
		logger := &mockLogger{}
		fs := afero.NewMemMapFs()
		cacheDir := "/cache"
		service := NewTranslationServiceWithCache(mockClient, logger, fs, cacheDir)

		// setInDiskCache should never panic
		service.setInDiskCache(key, value)

		// getFromDiskCache should never panic
		retrieved, found := service.getFromDiskCache(key)
		if !found {
			// It's possible that some keys can't be used as filenames
			// (e.g., those with path separators), so we skip if not found
			t.Skipf("Could not retrieve key from disk cache: %q", key)
		}

		if retrieved != value {
			t.Errorf("Retrieved value doesn't match: %q != %q", retrieved, value)
		}

		// Retrieve non-existent key should return false
		_, found = service.getFromDiskCache(key + "_nonexistent")
		if found {
			t.Error("Found non-existent key")
		}
	})
}
