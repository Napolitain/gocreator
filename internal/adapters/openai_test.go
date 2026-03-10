package adapters

import (
	"testing"

	"github.com/openai/openai-go/v3"
	"github.com/stretchr/testify/assert"
)

func TestNewOpenAIAdapter(t *testing.T) {
	client := openai.NewClient()
	adapter := NewOpenAIAdapter(client)
	
	assert.NotNil(t, adapter)
	assert.NotNil(t, adapter.client)
}

// Note: ChatCompletion and GenerateSpeech require actual API calls or mocking
// which is complex with the openai-go library. These are tested via integration tests.
// We focus on unit tests for business logic that doesn't require external API calls.
