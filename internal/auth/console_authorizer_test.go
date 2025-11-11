package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConsoleAuthorizer(t *testing.T) {
	authorizer := NewConsoleAuthorizer()

	assert.NotNil(t, authorizer)
}

// Note: GetAuthorizationCode requires user input, so we can't fully test it in an automated test
// In practice, this would be mocked in integration tests
