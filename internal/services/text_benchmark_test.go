package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/spf13/afero"
)

// Benchmark text loading
func BenchmarkTextService_Load(b *testing.B) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewTextService(fs, logger)

	// Create test file
	testPath := "/test/texts.txt"
	testContent := "First text\n-\nSecond text\n-\nThird text\n-\nFourth text\n-\nFifth text"
	_ = fs.MkdirAll("/test", 0755)
	_ = afero.WriteFile(fs, testPath, []byte(testContent), 0644)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.Load(ctx, testPath)
	}
}

// Benchmark text saving
func BenchmarkTextService_Save(b *testing.B) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewTextService(fs, logger)

	texts := []string{
		"First text for saving",
		"Second text for saving",
		"Third text for saving",
		"Fourth text for saving",
		"Fifth text for saving",
	}
	_ = fs.MkdirAll("/test", 0755)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testPath := fmt.Sprintf("/test/texts_%d.txt", i)
		_ = service.Save(ctx, testPath, texts)
	}
}

// Benchmark text hashing
func BenchmarkTextService_Hash(b *testing.B) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewTextService(fs, logger)

	text := "This is a test text for hashing benchmark. It contains some content to make the hash computation more realistic."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.Hash(text)
	}
}

// Benchmark loading hashes
func BenchmarkTextService_LoadHashes(b *testing.B) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewTextService(fs, logger)

	// Create test hashes file
	hashFile := "/test/hashes"
	hashes := []string{
		"hash1hash1hash1hash1hash1hash1",
		"hash2hash2hash2hash2hash2hash2",
		"hash3hash3hash3hash3hash3hash3",
		"hash4hash4hash4hash4hash4hash4",
		"hash5hash5hash5hash5hash5hash5",
	}
	_ = fs.MkdirAll("/test", 0755)
	_ = service.SaveHashes(context.Background(), hashFile, hashes)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.LoadHashes(ctx, hashFile)
	}
}

// Benchmark saving hashes
func BenchmarkTextService_SaveHashes(b *testing.B) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewTextService(fs, logger)

	hashes := []string{
		"hash1hash1hash1hash1hash1hash1",
		"hash2hash2hash2hash2hash2hash2",
		"hash3hash3hash3hash3hash3hash3",
		"hash4hash4hash4hash4hash4hash4",
		"hash5hash5hash5hash5hash5hash5",
	}
	_ = fs.MkdirAll("/test", 0755)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hashFile := fmt.Sprintf("/test/hashes_%d", i)
		_ = service.SaveHashes(ctx, hashFile, hashes)
	}
}

// Benchmark hash of loaded text (single text hashing workflow)
func BenchmarkTextService_LoadAndHash(b *testing.B) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewTextService(fs, logger)

	// Create test file with a single text entry
	testPath := "/test/file.txt"
	testContent := "This is test content for file hashing benchmark"
	_ = fs.MkdirAll("/test", 0755)
	_ = afero.WriteFile(fs, testPath, []byte(testContent), 0644)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Load and hash workflow - tests the common pattern of loading then hashing
		texts, _ := service.Load(ctx, testPath)
		if len(texts) > 0 {
			_ = service.Hash(texts[0])
		}
	}
}
