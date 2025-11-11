package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewServiceAccountProvider(t *testing.T) {
	provider := NewServiceAccountProvider("/path/to/service-account.json")

	assert.NotNil(t, provider)
	assert.Equal(t, "/path/to/service-account.json", provider.credentialsPath)
}

func TestServiceAccountProvider_GetClientOption(t *testing.T) {
	provider := NewServiceAccountProvider("/path/to/service-account.json")

	ctx := context.Background()
	clientOption, err := provider.GetClientOption(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, clientOption)
}
