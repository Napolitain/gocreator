package services

import (
	"context"
	"strings"
	"testing"

	"github.com/spf13/afero"
)

// FuzzTextService_Hash tests the Hash function with random inputs
func FuzzTextService_Hash(f *testing.F) {
	// Seed corpus with interesting cases
	f.Add("Hello world")
	f.Add("")
	f.Add("Special chars: !@#$%^&*()")
	f.Add("Unicode: 你好世界 🌍")
	f.Add(strings.Repeat("a", 1000))
	f.Add("\n\r\t")
	f.Add("a\x00b") // null byte

	f.Fuzz(func(t *testing.T, input string) {
		fs := afero.NewMemMapFs()
		logger := &mockLogger{}
		service := NewTextService(fs, logger)

		// Hash should never panic and always return a non-empty string
		hash := service.Hash(input)
		if hash == "" {
			t.Error("Hash returned empty string")
		}

		// Hash should be deterministic
		hash2 := service.Hash(input)
		if hash != hash2 {
			t.Errorf("Hash is not deterministic: %s != %s", hash, hash2)
		}

		// Hash should be 64 characters (SHA256 in hex)
		if len(hash) != 64 {
			t.Errorf("Hash length is %d, expected 64", len(hash))
		}

		// Different inputs should produce different hashes (with very high probability)
		if input != "" {
			differentInput := input + "x"
			differentHash := service.Hash(differentInput)
			if hash == differentHash {
				t.Errorf("Same hash for different inputs: %s and %s", input, differentInput)
			}
		}
	})
}

// FuzzTextService_LoadAndSave tests Load and Save with random text content
func FuzzTextService_LoadAndSave(f *testing.F) {
	// Seed corpus with interesting cases
	f.Add("Single text")
	f.Add("First\n-\nSecond")
	f.Add("Text with\nmultiple\nlines\n-\nAnother text")
	f.Add("")
	f.Add("-")
	f.Add("-\n-\n-")
	f.Add("Unicode: 你好\n-\nEmoji: 🌍")
	f.Add(strings.Repeat("Long text ", 100))

	f.Fuzz(func(t *testing.T, fileContent string) {
		fs := afero.NewMemMapFs()
		logger := &mockLogger{}
		service := NewTextService(fs, logger)

		// Write the fuzzed content to a file
		testPath := "/fuzz_test.txt"
		err := afero.WriteFile(fs, testPath, []byte(fileContent), 0644)
		if err != nil {
			t.Skip("Failed to write test file")
		}

		ctx := context.Background()

		// Load should never panic
		texts, err := service.Load(ctx, testPath)
		if err != nil {
			// Load might fail for some inputs, but shouldn't panic
			return
		}

		// Helper to check if text contains a delimiter line
		containsDelimiterLine := func(text string) bool {
			lines := strings.Split(text, "\n")
			for _, line := range lines {
				if line == "-" {
					return true
				}
			}
			return false
		}

		// Skip if any text contains a delimiter line or is exactly "-"
		// These cannot be properly round-tripped due to delimiter ambiguity
		for _, text := range texts {
			if text == "-" || containsDelimiterLine(text) {
				t.Skip("Input contains delimiter pattern that cannot be round-tripped")
			}
		}

		// Save should never panic
		savePath := "/fuzz_save.txt"
		err = service.Save(ctx, savePath, texts)
		if err != nil {
			t.Errorf("Save failed: %v", err)
			return
		}

		// Reload and verify consistency
		reloaded, err := service.Load(ctx, savePath)
		if err != nil {
			t.Errorf("Reload failed: %v", err)
			return
		}

		// Filter out empty strings from comparison
		// The Load function trims whitespace, so whitespace-only texts become empty
		nonEmptyTexts := []string{}
		for _, text := range texts {
			if text != "" {
				nonEmptyTexts = append(nonEmptyTexts, text)
			}
		}

		nonEmptyReloaded := []string{}
		for _, text := range reloaded {
			if text != "" {
				nonEmptyReloaded = append(nonEmptyReloaded, text)
			}
		}

		// The reloaded texts should match the saved texts
		if len(nonEmptyTexts) != len(nonEmptyReloaded) {
			t.Errorf("Length mismatch after save/load: %d != %d", len(nonEmptyTexts), len(nonEmptyReloaded))
			return
		}

		for i := range nonEmptyTexts {
			// Normalize line endings for comparison since bufio.Scanner
			// may handle \r\n differently than standalone \r
			original := strings.ReplaceAll(nonEmptyTexts[i], "\r\n", "\n")
			original = strings.ReplaceAll(original, "\r", "\n")
			reloaded := strings.ReplaceAll(nonEmptyReloaded[i], "\r\n", "\n")
			reloaded = strings.ReplaceAll(reloaded, "\r", "\n")
			
			if original != reloaded {
				t.Errorf("Text mismatch at index %d:\nOriginal: %q\nReloaded: %q", i, nonEmptyTexts[i], nonEmptyReloaded[i])
			}
		}
	})
}

// FuzzTextService_SaveHashes tests SaveHashes and LoadHashes with random hash values
func FuzzTextService_SaveHashes(f *testing.F) {
	// Seed corpus
	f.Add("abc123", "def456")
	f.Add("", "")
	f.Add("onlyone", "")
	f.Add(strings.Repeat("a", 64), strings.Repeat("b", 64))

	f.Fuzz(func(t *testing.T, hash1, hash2 string) {
		fs := afero.NewMemMapFs()
		logger := &mockLogger{}
		service := NewTextService(fs, logger)

		ctx := context.Background()
		testPath := "/fuzz_hashes.txt"

		// Create a slice of hashes, filtering out those with newlines
		// (the hash file format uses newlines as delimiters)
		hashes := []string{}
		if hash1 != "" && !strings.Contains(hash1, "\n") && !strings.Contains(hash1, "\r") {
			hashes = append(hashes, hash1)
		}
		if hash2 != "" && !strings.Contains(hash2, "\n") && !strings.Contains(hash2, "\r") {
			hashes = append(hashes, hash2)
		}

		// SaveHashes should never panic
		err := service.SaveHashes(ctx, testPath, hashes)
		if err != nil {
			t.Errorf("SaveHashes failed: %v", err)
			return
		}

		// LoadHashes should never panic
		loaded, err := service.LoadHashes(ctx, testPath)
		if err != nil {
			t.Errorf("LoadHashes failed: %v", err)
			return
		}

		// Verify the loaded hashes match
		if len(hashes) != len(loaded) {
			t.Errorf("Length mismatch: %d != %d", len(hashes), len(loaded))
			return
		}

		for i := range hashes {
			if hashes[i] != loaded[i] {
				t.Errorf("Hash mismatch at index %d: %q != %q", i, hashes[i], loaded[i])
			}
		}
	})
}
