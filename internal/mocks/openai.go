package mocks

import (
	"context"
	"io"

	"github.com/openai/openai-go/v3"
	"github.com/stretchr/testify/mock"
)

// MockOpenAIClient is a mock implementation of the OpenAIClient interface
type MockOpenAIClient struct {
	mock.Mock
}

func (m *MockOpenAIClient) ChatCompletion(ctx context.Context, messages []openai.ChatCompletionMessageParamUnion) (string, error) {
	args := m.Called(ctx, messages)
	return args.String(0), args.Error(1)
}

func (m *MockOpenAIClient) GenerateSpeech(ctx context.Context, text string) (io.ReadCloser, error) {
	args := m.Called(ctx, text)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}
