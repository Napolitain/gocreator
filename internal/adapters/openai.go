package adapters

import (
	"context"
	"io"

	"gocreator/internal/interfaces"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"
)

// OpenAIAdapter wraps the OpenAI client
type OpenAIAdapter struct {
	client openai.Client
}

// NewOpenAIAdapter creates a new OpenAI adapter
func NewOpenAIAdapter(client openai.Client) *OpenAIAdapter {
	return &OpenAIAdapter{client: client}
}

// ChatCompletion sends a chat completion request
func (a *OpenAIAdapter) ChatCompletion(ctx context.Context, messages []openai.ChatCompletionMessageParamUnion) (string, error) {
	resp, err := a.client.Chat.Completions.New(
		ctx,
		openai.ChatCompletionNewParams{
			Model:    openai.ChatModelGPT4oMini,
			Messages: messages,
		},
	)
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}

// GenerateSpeech generates speech from text
func (a *OpenAIAdapter) GenerateSpeech(ctx context.Context, text string) (io.ReadCloser, error) {
	return a.GenerateSpeechWithOptions(ctx, text, interfaces.SpeechOptions{})
}

// GenerateSpeechWithOptions generates speech from text using per-request voice settings.
func (a *OpenAIAdapter) GenerateSpeechWithOptions(ctx context.Context, text string, options interfaces.SpeechOptions) (io.ReadCloser, error) {
	model := openai.SpeechModelTTS1HD
	if options.Model != "" {
		model = options.Model
	}

	voice := openai.AudioSpeechNewParamsVoiceAlloy
	if options.Voice != "" {
		voice = openai.AudioSpeechNewParamsVoice(options.Voice)
	}

	params := openai.AudioSpeechNewParams{
		Model:          model,
		Input:          text,
		Voice:          voice,
		ResponseFormat: openai.AudioSpeechNewParamsResponseFormatMP3,
	}
	if options.Speed > 0 {
		params.Speed = param.NewOpt(options.Speed)
	}

	response, err := a.client.Audio.Speech.New(
		ctx,
		params,
	)
	if err != nil {
		return nil, err
	}
	return response.Body, nil
}
