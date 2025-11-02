package services

import (
	"context"
	"fmt"
	"sync"

	"gocreator/internal/interfaces"

	"github.com/openai/openai-go/v3"
)

// TranslationService handles text translation
type TranslationService struct {
	client interfaces.OpenAIClient
	logger interfaces.Logger
}

// NewTranslationService creates a new translation service
func NewTranslationService(client interfaces.OpenAIClient, logger interfaces.Logger) *TranslationService {
	return &TranslationService{
		client: client,
		logger: logger,
	}
}

// Translate translates text to target language
func (s *TranslationService) Translate(ctx context.Context, text, targetLang string) (string, error) {
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.UserMessage(fmt.Sprintf("Translate '%s' to %s and don't return anything else than the translation.", text, targetLang)),
	}

	translated, err := s.client.ChatCompletion(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("translation failed: %w", err)
	}

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
