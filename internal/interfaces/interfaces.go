package interfaces

import (
	"context"
	"io"
	"log/slog"

	"github.com/openai/openai-go/v3"
	"github.com/spf13/afero"
)

// FileSystem abstracts filesystem operations
type FileSystem interface {
	afero.Fs
}

// Translator handles text translation
type Translator interface {
	Translate(ctx context.Context, text, targetLang string) (string, error)
	TranslateBatch(ctx context.Context, texts []string, targetLang string) ([]string, error)
}

// AudioGenerator generates audio from text
type AudioGenerator interface {
	Generate(ctx context.Context, text, outputPath string) error
	GenerateBatch(ctx context.Context, texts []string, outputDir string) ([]string, error)
}

// VideoGenerator generates videos from slides and audio
type VideoGenerator interface {
	GenerateFromSlides(ctx context.Context, slides, audioPaths []string, outputPath string) error
}

// TextProcessor handles text loading and processing
type TextProcessor interface {
	Load(ctx context.Context, path string) ([]string, error)
	Save(ctx context.Context, path string, texts []string) error
	Hash(text string) string
}

// SlideLoader loads slide assets from a local directory
type SlideLoader interface {
	LoadSlides(ctx context.Context, dir string) ([]string, error)
}

// CacheService manages caching operations
type CacheService interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{})
	Delete(key string)
	Clear()
}

// Logger abstracts logging operations
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
}

// CommandResult contains the captured output from an external command.
type CommandResult struct {
	Stdout []byte
	Stderr []byte
}

// CombinedOutput returns stdout followed by stderr, matching the common parsing pattern
// used by ffmpeg/ffprobe helpers throughout the services layer.
func (r CommandResult) CombinedOutput() []byte {
	output := make([]byte, 0, len(r.Stdout)+len(r.Stderr))
	output = append(output, r.Stdout...)
	output = append(output, r.Stderr...)
	return output
}

// CommandExecutor runs external commands and returns captured output.
type CommandExecutor interface {
	Run(ctx context.Context, name string, args ...string) (CommandResult, error)
}

// SpeechOptions describes optional TTS settings for synthesized narration.
type SpeechOptions struct {
	Model string
	Voice string
	Speed float64
}

// SpeechSynthesisClient optionally supports per-request speech options.
type SpeechSynthesisClient interface {
	GenerateSpeechWithOptions(ctx context.Context, text string, options SpeechOptions) (io.ReadCloser, error)
}

// OpenAIClient wraps OpenAI client operations
type OpenAIClient interface {
	ChatCompletion(ctx context.Context, messages []openai.ChatCompletionMessageParamUnion) (string, error)
	GenerateSpeech(ctx context.Context, text string) (io.ReadCloser, error)
}

// SlogLogger adapts slog.Logger to our Logger interface
type SlogLogger struct {
	*slog.Logger
}

// With returns a new logger with additional context
func (l *SlogLogger) With(args ...any) Logger {
	return &SlogLogger{Logger: l.Logger.With(args...)}
}

// ProgressCallback receives progress updates during video creation
type ProgressCallback interface {
	OnStageStart(stage string)
	OnStageProgress(stage string, progress int, message string)
	OnStageComplete(stage string, success bool, message string)
	OnItemStart(stage string, item string)
	OnItemProgress(stage string, item string, progress int, message string)
	OnItemComplete(stage string, item string, success bool, message string)
}

// NoOpProgressCallback is a progress callback that does nothing
type NoOpProgressCallback struct{}

func (n *NoOpProgressCallback) OnStageStart(stage string)                                  {}
func (n *NoOpProgressCallback) OnStageProgress(stage string, progress int, message string) {}
func (n *NoOpProgressCallback) OnStageComplete(stage string, success bool, message string) {}
func (n *NoOpProgressCallback) OnItemStart(stage string, item string)                      {}
func (n *NoOpProgressCallback) OnItemProgress(stage string, item string, progress int, message string) {
}
func (n *NoOpProgressCallback) OnItemComplete(stage string, item string, success bool, message string) {
}
