package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/spf13/afero"
)

// Benchmark sidecar text loading
func BenchmarkTextService_Load(b *testing.B) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewTextService(fs, logger)

	// Create test file
	testPath := "/test/slide1.txt"
	testContent := "Narration for slide 1"
	_ = fs.MkdirAll("/test", 0755)
	_ = afero.WriteFile(fs, testPath, []byte(testContent), 0644)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.Load(ctx, testPath)
	}
}

// Benchmark sidecar text saving
func BenchmarkTextService_Save(b *testing.B) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewTextService(fs, logger)

	texts := []string{
		"Narration for slide 1",
	}
	_ = fs.MkdirAll("/test", 0755)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testPath := fmt.Sprintf("/test/slide_%d.txt", i)
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

// Benchmark hash of loaded sidecar text
func BenchmarkTextService_LoadAndHash(b *testing.B) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewTextService(fs, logger)

	// Create test file with a single text entry
	testPath := "/test/slide1.txt"
	testContent := "This is test content for slide sidecar hashing benchmark"
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
