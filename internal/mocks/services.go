package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockTextProcessor is a mock implementation of the TextProcessor interface
type MockTextProcessor struct {
	mock.Mock
}

func (m *MockTextProcessor) Load(ctx context.Context, path string) ([]string, error) {
	args := m.Called(ctx, path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockTextProcessor) Save(ctx context.Context, path string, texts []string) error {
	args := m.Called(ctx, path, texts)
	return args.Error(0)
}

func (m *MockTextProcessor) Hash(text string) string {
	args := m.Called(text)
	return args.String(0)
}

// MockTranslator is a mock implementation of the Translator interface
type MockTranslator struct {
	mock.Mock
}

func (m *MockTranslator) Translate(ctx context.Context, text, targetLang string) (string, error) {
	args := m.Called(ctx, text, targetLang)
	return args.String(0), args.Error(1)
}

func (m *MockTranslator) TranslateBatch(ctx context.Context, texts []string, targetLang string) ([]string, error) {
	args := m.Called(ctx, texts, targetLang)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

// MockAudioGenerator is a mock implementation of the AudioGenerator interface
type MockAudioGenerator struct {
	mock.Mock
}

func (m *MockAudioGenerator) Generate(ctx context.Context, text, outputPath string) error {
	args := m.Called(ctx, text, outputPath)
	return args.Error(0)
}

func (m *MockAudioGenerator) GenerateBatch(ctx context.Context, texts []string, outputDir string) ([]string, error) {
	args := m.Called(ctx, texts, outputDir)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

// MockVideoGenerator is a mock implementation of the VideoGenerator interface
type MockVideoGenerator struct {
	mock.Mock
}

func (m *MockVideoGenerator) GenerateFromSlides(ctx context.Context, slides, audioPaths []string, outputPath string) error {
	args := m.Called(ctx, slides, audioPaths, outputPath)
	return args.Error(0)
}

// MockSlideLoader is a mock implementation of the SlideLoader interface
type MockSlideLoader struct {
	mock.Mock
}

func (m *MockSlideLoader) LoadSlides(ctx context.Context, dir string) ([]string, error) {
	args := m.Called(ctx, dir)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockSlideLoader) LoadFromGoogleSlides(ctx context.Context, presentationID, outputDir string) ([]string, []string, error) {
	args := m.Called(ctx, presentationID, outputDir)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	if args.Get(1) == nil {
		return args.Get(0).([]string), nil, args.Error(2)
	}
	return args.Get(0).([]string), args.Get(1).([]string), args.Error(2)
}
