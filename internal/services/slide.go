package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gocreator/internal/interfaces"

	"github.com/spf13/afero"
)

var (
	naturalTokenPattern   = regexp.MustCompile(`\d+|\D+`)
	artifactNameSanitizer = regexp.MustCompile(`[^A-Za-z0-9._-]+`)
)

var supportedSlideExtensions = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".pdf":  true,
	".mp4":  true,
	".mov":  true,
	".avi":  true,
	".mkv":  true,
	".webm": true,
}

type commandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

type pdfCacheManifest struct {
	SourceHash string `json:"source_hash"`
	PageCount  int    `json:"page_count"`
}

// SlideService handles slide loading
type SlideService struct {
	fs         afero.Fs
	logger     interfaces.Logger
	runCommand commandRunner
}

// NewSlideService creates a new slide service
func NewSlideService(fs afero.Fs, logger interfaces.Logger) *SlideService {
	return &SlideService{
		fs:         fs,
		logger:     logger,
		runCommand: defaultCommandRunner,
	}
}

func defaultCommandRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}

// LoadSlides loads slide images, videos, and PDFs from a directory.
func (s *SlideService) LoadSlides(ctx context.Context, dir string) ([]string, error) {
	exists, err := afero.DirExists(s.fs, dir)
	if err != nil {
		return nil, fmt.Errorf("failed to check directory: %w", err)
	}
	if !exists {
		if err := s.fs.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}
		return []string{}, nil
	}

	files, err := afero.ReadDir(s.fs, dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	sort.Slice(files, func(i, j int) bool {
		return compareNatural(strings.ToLower(files[i].Name()), strings.ToLower(files[j].Name())) < 0
	})

	var slides []string
	pdfCacheRoot := filepath.Join(filepath.Dir(dir), "cache", "pdf")

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(file.Name()))
		if !supportedSlideExtensions[ext] {
			continue
		}

		slidePath := filepath.Join(dir, file.Name())
		if ext != ".pdf" {
			slides = append(slides, slidePath)
			continue
		}

		expandedSlides, err := s.expandPDF(ctx, slidePath, pdfCacheRoot)
		if err != nil {
			return nil, err
		}
		slides = append(slides, expandedSlides...)
	}

	return slides, nil
}

func (s *SlideService) expandPDF(ctx context.Context, slidePath, cacheRoot string) ([]string, error) {
	sourceHash, err := s.hashFile(slidePath)
	if err != nil {
		return nil, fmt.Errorf("failed to hash PDF %s: %w", slidePath, err)
	}

	baseName := sanitizeArtifactName(strings.TrimSuffix(filepath.Base(slidePath), filepath.Ext(slidePath)))
	artifactDir := filepath.Join(cacheRoot, fmt.Sprintf("%s-%s", baseName, sourceHash[:12]))

	cachedSlides, err := s.loadCachedPDFSlides(artifactDir, baseName, sourceHash)
	if err != nil {
		s.logger.Warn("Ignoring invalid PDF cache", "path", slidePath, "error", err)
	} else if cachedSlides != nil {
		s.logger.Info("Using cached PDF slides", "path", slidePath, "pages", len(cachedSlides))
		return cachedSlides, nil
	}

	if err := s.fs.RemoveAll(artifactDir); err != nil {
		return nil, fmt.Errorf("failed to reset PDF cache for %s: %w", slidePath, err)
	}
	if err := s.fs.MkdirAll(artifactDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create PDF cache directory for %s: %w", slidePath, err)
	}

	pageCount, err := s.getPDFPageCount(ctx, slidePath)
	if err != nil {
		return nil, err
	}
	if pageCount < 1 {
		return nil, fmt.Errorf("PDF %s does not contain any pages", slidePath)
	}

	output, err := s.runCommand(ctx, "pdfseparate", slidePath, filepath.Join(artifactDir, fmt.Sprintf("%s-page-%%04d.pdf", baseName)))
	if err != nil {
		return nil, wrapPDFCommandError("pdfseparate", slidePath, err, output)
	}

	renderedSlides := make([]string, 0, pageCount)
	for page := 1; page <= pageCount; page++ {
		splitPDF := filepath.Join(artifactDir, fmt.Sprintf("%s-page-%04d.pdf", baseName, page))
		renderBase := filepath.Join(artifactDir, fmt.Sprintf("%s-page-%04d", baseName, page))

		output, err := s.runCommand(ctx, "pdftocairo", "-png", "-singlefile", splitPDF, renderBase)
		if err != nil {
			return nil, wrapPDFCommandError("pdftocairo", slidePath, err, output)
		}

		renderedSlide := renderBase + ".png"
		exists, err := afero.Exists(s.fs, renderedSlide)
		if err != nil {
			return nil, fmt.Errorf("failed to check rendered PDF page for %s: %w", slidePath, err)
		}
		if !exists {
			return nil, fmt.Errorf("rendered PDF page missing for %s: %s", slidePath, renderedSlide)
		}

		renderedSlides = append(renderedSlides, renderedSlide)
	}

	if err := s.savePDFCacheManifest(artifactDir, sourceHash, pageCount); err != nil {
		s.logger.Warn("Failed to save PDF cache manifest", "path", slidePath, "error", err)
	}

	return renderedSlides, nil
}

func (s *SlideService) loadCachedPDFSlides(artifactDir, baseName, sourceHash string) ([]string, error) {
	manifestPath := filepath.Join(artifactDir, "manifest.json")
	exists, err := afero.Exists(s.fs, manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check PDF cache manifest: %w", err)
	}
	if !exists {
		return nil, nil
	}

	data, err := afero.ReadFile(s.fs, manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF cache manifest: %w", err)
	}

	var manifest pdfCacheManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse PDF cache manifest: %w", err)
	}

	if manifest.SourceHash != sourceHash || manifest.PageCount < 1 {
		return nil, nil
	}

	renderedSlides := make([]string, 0, manifest.PageCount)
	for page := 1; page <= manifest.PageCount; page++ {
		splitPDF := filepath.Join(artifactDir, fmt.Sprintf("%s-page-%04d.pdf", baseName, page))
		renderedSlide := filepath.Join(artifactDir, fmt.Sprintf("%s-page-%04d.png", baseName, page))

		for _, requiredPath := range []string{splitPDF, renderedSlide} {
			exists, err := afero.Exists(s.fs, requiredPath)
			if err != nil {
				return nil, fmt.Errorf("failed to check cached PDF artifact %s: %w", requiredPath, err)
			}
			if !exists {
				return nil, nil
			}
		}

		renderedSlides = append(renderedSlides, renderedSlide)
	}

	return renderedSlides, nil
}

func (s *SlideService) savePDFCacheManifest(artifactDir, sourceHash string, pageCount int) error {
	manifestPath := filepath.Join(artifactDir, "manifest.json")
	manifest := pdfCacheManifest{
		SourceHash: sourceHash,
		PageCount:  pageCount,
	}

	data, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal PDF cache manifest: %w", err)
	}

	if err := afero.WriteFile(s.fs, manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write PDF cache manifest: %w", err)
	}

	return nil
}

func (s *SlideService) getPDFPageCount(ctx context.Context, slidePath string) (int, error) {
	output, err := s.runCommand(ctx, "pdfinfo", slidePath)
	if err != nil {
		return 0, wrapPDFCommandError("pdfinfo", slidePath, err, output)
	}

	pageCount, encrypted, err := parsePDFInfoOutput(output)
	if err != nil {
		return 0, fmt.Errorf("failed to parse PDF metadata for %s: %w", slidePath, err)
	}
	if encrypted {
		return 0, fmt.Errorf("encrypted PDFs are not supported: %s", slidePath)
	}

	return pageCount, nil
}

func parsePDFInfoOutput(output []byte) (int, bool, error) {
	var pageCount int
	var encrypted bool

	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "Pages:") {
			value := strings.TrimSpace(strings.TrimPrefix(line, "Pages:"))
			parsedPageCount, err := strconv.Atoi(value)
			if err != nil {
				return 0, false, fmt.Errorf("invalid page count %q", value)
			}
			pageCount = parsedPageCount
			continue
		}

		if strings.HasPrefix(line, "Encrypted:") {
			value := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(line, "Encrypted:")))
			encrypted = !strings.HasPrefix(value, "no")
		}
	}

	if pageCount == 0 {
		return 0, encrypted, fmt.Errorf("page count not found")
	}

	return pageCount, encrypted, nil
}

func wrapPDFCommandError(tool, slidePath string, err error, output []byte) error {
	if errors.Is(err, exec.ErrNotFound) {
		return fmt.Errorf("PDF input requires %s in PATH to process %s: %w", tool, slidePath, err)
	}

	outputText := strings.TrimSpace(string(output))
	lowerOutput := strings.ToLower(outputText)
	if strings.Contains(lowerOutput, "encrypted") || strings.Contains(lowerOutput, "password") {
		return fmt.Errorf("encrypted PDFs are not supported: %s", slidePath)
	}
	if outputText == "" {
		return fmt.Errorf("%s failed for %s: %w", tool, slidePath, err)
	}

	return fmt.Errorf("%s failed for %s: %w: %s", tool, slidePath, err, outputText)
}

func (s *SlideService) hashFile(path string) (string, error) {
	file, err := s.fs.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to hash file: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func sanitizeArtifactName(name string) string {
	sanitized := artifactNameSanitizer.ReplaceAllString(name, "-")
	sanitized = strings.Trim(sanitized, "-.")
	if sanitized == "" {
		return "slide"
	}
	return sanitized
}

func compareNatural(a, b string) int {
	aTokens := naturalTokenPattern.FindAllString(a, -1)
	bTokens := naturalTokenPattern.FindAllString(b, -1)
	maxTokens := len(aTokens)
	if len(bTokens) < maxTokens {
		maxTokens = len(bTokens)
	}

	for i := 0; i < maxTokens; i++ {
		aToken := aTokens[i]
		bToken := bTokens[i]

		aIsNumeric := isNumericToken(aToken)
		bIsNumeric := isNumericToken(bToken)

		switch {
		case aIsNumeric && bIsNumeric:
			if cmp := compareNumericTokens(aToken, bToken); cmp != 0 {
				return cmp
			}
		default:
			if cmp := strings.Compare(aToken, bToken); cmp != 0 {
				return cmp
			}
		}
	}

	switch {
	case len(aTokens) < len(bTokens):
		return -1
	case len(aTokens) > len(bTokens):
		return 1
	default:
		return strings.Compare(a, b)
	}
}

func isNumericToken(token string) bool {
	if token == "" {
		return false
	}

	for _, r := range token {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}

func compareNumericTokens(a, b string) int {
	trimmedA := strings.TrimLeft(a, "0")
	if trimmedA == "" {
		trimmedA = "0"
	}

	trimmedB := strings.TrimLeft(b, "0")
	if trimmedB == "" {
		trimmedB = "0"
	}

	switch {
	case len(trimmedA) < len(trimmedB):
		return -1
	case len(trimmedA) > len(trimmedB):
		return 1
	}

	if cmp := strings.Compare(trimmedA, trimmedB); cmp != 0 {
		return cmp
	}

	return strings.Compare(a, b)
}
