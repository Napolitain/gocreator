package adapters

import (
	"context"
	"io"

	"github.com/openai/openai-go/v3"
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
	response, err := a.client.Audio.Speech.New(
		ctx,
		openai.AudioSpeechNewParams{
			Model:          openai.SpeechModelTTS1HD,
			Input:          text,
			Voice:          openai.AudioSpeechNewParamsVoice("onyx"),
			ResponseFormat: openai.AudioSpeechNewParamsResponseFormatMP3,
		},
	)
	if err != nil {
		return nil, err
	}
	return response.Body, nil
}
