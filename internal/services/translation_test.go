package services

import (
	"context"
	"errors"
	"testing"

	"gocreator/internal/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTranslationService_Translate(t *testing.T) {
	tests := []struct {
		name           string
		inputText      string
		targetLang     string
		mockResponse   string
		mockError      error
		expectedResult string
		expectError    bool
	}{
		{
			name:           "successful translation",
			inputText:      "Hello world",
			targetLang:     "es",
			mockResponse:   "Hola mundo",
			mockError:      nil,
			expectedResult: "Hola mundo",
			expectError:    false,
		},
		{
			name:           "api error",
			inputText:      "Hello",
			targetLang:     "fr",
			mockResponse:   "",
			mockError:      errors.New("API error"),
			expectedResult: "",
			expectError:    true,
		},
		{
			name:           "empty text",
			inputText:      "",
			targetLang:     "de",
			mockResponse:   "",
			mockError:      nil,
			expectedResult: "",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(mocks.MockOpenAIClient)
			logger := &mockLogger{}
			service := NewTranslationService(mockClient, logger)

			// Setup mock expectations
			mockClient.On("ChatCompletion", mock.Anything, mock.AnythingOfType("[]openai.ChatCompletionMessageParamUnion")).
				Return(tt.mockResponse, tt.mockError)

			ctx := context.Background()
			result, err := service.Translate(ctx, tt.inputText, tt.targetLang)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestTranslationService_TranslateBatch(t *testing.T) {
	t.Run("successful batch translation", func(t *testing.T) {
		mockClient := new(mocks.MockOpenAIClient)
		logger := &mockLogger{}
		service := NewTranslationService(mockClient, logger)

		inputTexts := []string{"Hello", "Goodbye", "Thank you"}

		// Setup mock to return a translation for any input (3 times)
		mockClient.On("ChatCompletion", mock.Anything, mock.AnythingOfType("[]openai.ChatCompletionMessageParamUnion")).
			Return("translated", nil).Times(3)

		ctx := context.Background()
		results, err := service.TranslateBatch(ctx, inputTexts, "es")

		assert.NoError(t, err)
		assert.Len(t, results, 3)
		// Verify all results are not empty
		for _, result := range results {
			assert.NotEmpty(t, result)
		}
		mockClient.AssertExpectations(t)
	})

	t.Run("empty input", func(t *testing.T) {
		mockClient := new(mocks.MockOpenAIClient)
		logger := &mockLogger{}
		service := NewTranslationService(mockClient, logger)

		ctx := context.Background()
		results, err := service.TranslateBatch(ctx, []string{}, "fr")

		assert.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("single item batch", func(t *testing.T) {
		mockClient := new(mocks.MockOpenAIClient)
		logger := &mockLogger{}
		service := NewTranslationService(mockClient, logger)

		mockClient.On("ChatCompletion", mock.Anything, mock.AnythingOfType("[]openai.ChatCompletionMessageParamUnion")).
			Return("Hallo", nil).Once()

		ctx := context.Background()
		results, err := service.TranslateBatch(ctx, []string{"Hello"}, "de")

		assert.NoError(t, err)
		assert.Equal(t, []string{"Hallo"}, results)
		mockClient.AssertExpectations(t)
	})
}

func TestTranslationService_TranslateBatch_WithError(t *testing.T) {
	mockClient := new(mocks.MockOpenAIClient)
	logger := &mockLogger{}
	service := NewTranslationService(mockClient, logger)

	inputTexts := []string{"Hello", "Goodbye"}
	
	// First translation succeeds, second fails
	mockClient.On("ChatCompletion", mock.Anything, mock.AnythingOfType("[]openai.ChatCompletionMessageParamUnion")).
		Return("Hola", nil).Once()
	mockClient.On("ChatCompletion", mock.Anything, mock.AnythingOfType("[]openai.ChatCompletionMessageParamUnion")).
		Return("", errors.New("API error")).Once()

	ctx := context.Background()
	_, err := service.TranslateBatch(ctx, inputTexts, "es")

	assert.Error(t, err)
	mockClient.AssertExpectations(t)
}
