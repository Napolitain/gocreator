package services

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/spf13/afero"
)

var (
	pdfPageNarrationPattern           = regexp.MustCompile(`^(.*)-page-(\d+)$`)
	supportedNarrationAudioExtensions = []string{".aac", ".flac", ".m4a", ".mp3", ".ogg", ".opus", ".wav"}
)

func (vc *VideoCreator) resolveTextsForLanguage(
	ctx context.Context,
	inputLang string,
	lang string,
	slidesDir string,
	slides []string,
) ([]string, int, error) {
	texts := make([]string, len(slides))
	sourceTexts := make([]string, 0, len(slides))
	sourceIndexes := make([]int, 0, len(slides))

	for idx, slidePath := range slides {
		audioPath, found, err := vc.lookupAudioForLanguage(slidesDir, slidePath, inputLang, lang)
		if err != nil {
			return nil, 0, err
		}
		if found && audioPath != "" {
			continue
		}

		text, found, err := vc.lookupTextForLanguage(slidesDir, slidePath, inputLang, lang)
		if err != nil {
			return nil, 0, err
		}
		if found {
			texts[idx] = text
			continue
		}

		if lang == inputLang {
			continue
		}

		sourceText, found, err := vc.lookupSourceText(slidesDir, slidePath, inputLang)
		if err != nil {
			return nil, 0, err
		}
		if !found {
			continue
		}

		sourceTexts = append(sourceTexts, sourceText)
		sourceIndexes = append(sourceIndexes, idx)
	}

	if len(sourceTexts) == 0 {
		return texts, 0, nil
	}

	translatedTexts, err := vc.translationService.TranslateBatch(ctx, sourceTexts, lang)
	if err != nil {
		return nil, 0, err
	}

	for i, slideIndex := range sourceIndexes {
		texts[slideIndex] = translatedTexts[i]
	}

	return texts, len(sourceIndexes), nil
}

func (vc *VideoCreator) resolveAudioForLanguage(
	ctx context.Context,
	inputLang string,
	lang string,
	slidesDir string,
	slides []string,
	texts []string,
	audioDir string,
) ([]string, int, int, error) {
	audioPaths := make([]string, len(slides))
	type ttsJob struct {
		index int
		text  string
		path  string
	}

	ttsJobs := make([]ttsJob, 0, len(slides))
	prerecordedCount := 0

	for idx, slidePath := range slides {
		audioPath, found, err := vc.lookupAudioForLanguage(slidesDir, slidePath, inputLang, lang)
		if err != nil {
			return nil, 0, 0, err
		}
		if found {
			audioPaths[idx] = audioPath
			prerecordedCount++
			continue
		}

		if strings.TrimSpace(texts[idx]) == "" {
			return nil, 0, 0, fmt.Errorf("slide %s has no matching text or audio sidecar for language %s", slideNarrationLabel(slidePath), lang)
		}

		ttsJobs = append(ttsJobs, ttsJob{
			index: idx,
			text:  texts[idx],
			path:  filepath.Join(audioDir, fmt.Sprintf("%d.mp3", idx)),
		})
	}

	if len(ttsJobs) == 0 {
		return audioPaths, prerecordedCount, 0, nil
	}

	if err := vc.fs.MkdirAll(audioDir, 0755); err != nil {
		return nil, 0, 0, fmt.Errorf("failed to create audio cache directory: %w", err)
	}

	errors := make([]error, len(ttsJobs))
	var wg sync.WaitGroup

	for i, job := range ttsJobs {
		wg.Add(1)
		go func(jobIndex int, current ttsJob) {
			defer wg.Done()
			if err := vc.audioService.Generate(ctx, current.text, current.path); err != nil {
				errors[jobIndex] = fmt.Errorf("failed to generate narration for slide %s: %w", slideNarrationLabel(slides[current.index]), err)
				return
			}
			audioPaths[current.index] = current.path
		}(i, job)
	}

	wg.Wait()

	for _, err := range errors {
		if err != nil {
			return nil, 0, 0, err
		}
	}

	return audioPaths, prerecordedCount, len(ttsJobs), nil
}

func (vc *VideoCreator) lookupTextForLanguage(slidesDir, slidePath, inputLang, lang string) (string, bool, error) {
	baseNames := slideNarrationBaseCandidates(slidePath)
	if lang == inputLang {
		return vc.readPreferredTextSidecar(slidesDir, [][]string{
			buildTextCandidatePaths(slidesDir, baseNames, lang),
			buildGenericTextCandidatePaths(slidesDir, baseNames),
		})
	}

	return vc.readPreferredTextSidecar(slidesDir, [][]string{
		buildTextCandidatePaths(slidesDir, baseNames, lang),
	})
}

func (vc *VideoCreator) lookupSourceText(slidesDir, slidePath, inputLang string) (string, bool, error) {
	baseNames := slideNarrationBaseCandidates(slidePath)
	return vc.readPreferredTextSidecar(slidesDir, [][]string{
		buildTextCandidatePaths(slidesDir, baseNames, inputLang),
		buildGenericTextCandidatePaths(slidesDir, baseNames),
	})
}

func (vc *VideoCreator) lookupAudioForLanguage(slidesDir, slidePath, inputLang, lang string) (string, bool, error) {
	baseNames := slideNarrationBaseCandidates(slidePath)
	groups := [][]string{
		buildAudioCandidatePaths(slidesDir, baseNames, lang),
	}
	if lang == inputLang {
		groups = append(groups, buildGenericAudioCandidatePaths(slidesDir, baseNames))
	}
	return vc.findPreferredAudioSidecar(groups)
}

func (vc *VideoCreator) readPreferredTextSidecar(slidesDir string, groups [][]string) (string, bool, error) {
	for _, group := range groups {
		matches, err := existingPaths(vc.fs, group)
		if err != nil {
			return "", false, err
		}
		if len(matches) == 0 {
			continue
		}
		if len(matches) > 1 {
			return "", false, fmt.Errorf("multiple matching text sidecars found in %s: %s", slidesDir, strings.Join(matches, ", "))
		}

		data, err := afero.ReadFile(vc.fs, matches[0])
		if err != nil {
			return "", false, fmt.Errorf("failed to read text sidecar %s: %w", matches[0], err)
		}
		return strings.TrimSpace(string(data)), true, nil
	}

	return "", false, nil
}

func (vc *VideoCreator) findPreferredAudioSidecar(groups [][]string) (string, bool, error) {
	for _, group := range groups {
		matches, err := existingPaths(vc.fs, group)
		if err != nil {
			return "", false, err
		}
		if len(matches) == 0 {
			continue
		}
		if len(matches) > 1 {
			return "", false, fmt.Errorf("multiple matching audio sidecars found: %s", strings.Join(matches, ", "))
		}
		return matches[0], true, nil
	}

	return "", false, nil
}

func buildTextCandidatePaths(slidesDir string, baseNames []string, lang string) []string {
	paths := make([]string, 0, len(baseNames))
	for _, baseName := range baseNames {
		paths = append(paths, filepath.Join(slidesDir, fmt.Sprintf("%s.%s.txt", baseName, lang)))
	}
	return paths
}

func buildGenericTextCandidatePaths(slidesDir string, baseNames []string) []string {
	paths := make([]string, 0, len(baseNames))
	for _, baseName := range baseNames {
		paths = append(paths, filepath.Join(slidesDir, baseName+".txt"))
	}
	return paths
}

func buildAudioCandidatePaths(slidesDir string, baseNames []string, lang string) []string {
	paths := make([]string, 0, len(baseNames)*len(supportedNarrationAudioExtensions))
	for _, baseName := range baseNames {
		for _, ext := range supportedNarrationAudioExtensions {
			paths = append(paths, filepath.Join(slidesDir, fmt.Sprintf("%s.%s%s", baseName, lang, ext)))
		}
	}
	return paths
}

func buildGenericAudioCandidatePaths(slidesDir string, baseNames []string) []string {
	paths := make([]string, 0, len(baseNames)*len(supportedNarrationAudioExtensions))
	for _, baseName := range baseNames {
		for _, ext := range supportedNarrationAudioExtensions {
			paths = append(paths, filepath.Join(slidesDir, baseName+ext))
		}
	}
	return paths
}

func existingPaths(fs afero.Fs, candidates []string) ([]string, error) {
	matches := make([]string, 0, 1)
	for _, candidate := range candidates {
		exists, err := afero.Exists(fs, candidate)
		if err != nil {
			return nil, fmt.Errorf("failed to inspect sidecar %s: %w", candidate, err)
		}
		if exists {
			matches = append(matches, candidate)
		}
	}
	return matches, nil
}

func slideNarrationBaseCandidates(slidePath string) []string {
	baseName := strings.TrimSuffix(filepath.Base(slidePath), filepath.Ext(slidePath))
	candidates := []string{baseName}

	matches := pdfPageNarrationPattern.FindStringSubmatch(baseName)
	if len(matches) != 3 {
		return candidates
	}

	pageNumber, err := strconv.Atoi(matches[2])
	if err != nil {
		return candidates
	}

	candidates = append(candidates,
		fmt.Sprintf("%s-p%03d", matches[1], pageNumber),
		fmt.Sprintf("%s-p%d", matches[1], pageNumber),
	)
	return candidates
}

func slideNarrationLabel(slidePath string) string {
	return strings.Join(slideNarrationBaseCandidates(slidePath), " / ")
}
