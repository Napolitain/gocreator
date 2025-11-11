package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/stretchr/testify/assert"
)

func TestNewFileTokenStore(t *testing.T) {
	store := NewFileTokenStore("/path/to/token.json")

	assert.NotNil(t, store)
	assert.Equal(t, "/path/to/token.json", store.tokenPath)
}

func TestFileTokenStore_SaveAndLoadToken(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "token-store-test")
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	tokenPath := filepath.Join(tmpDir, "token.json")
	store := NewFileTokenStore(tokenPath)

	// Create a test token
	testToken := &oauth2.Token{
		AccessToken:  "test-access-token",
		TokenType:    "Bearer",
		RefreshToken: "test-refresh-token",
		Expiry:       time.Now().Add(time.Hour),
	}

	// Save the token
	err = store.SaveToken(testToken)
	assert.NoError(t, err)

	// Load the token
	loadedToken, err := store.LoadToken()
	assert.NoError(t, err)
	assert.NotNil(t, loadedToken)
	assert.Equal(t, testToken.AccessToken, loadedToken.AccessToken)
	assert.Equal(t, testToken.RefreshToken, loadedToken.RefreshToken)
	assert.Equal(t, testToken.TokenType, loadedToken.TokenType)
}

func TestFileTokenStore_LoadToken_FileNotFound(t *testing.T) {
	store := NewFileTokenStore("/nonexistent/token.json")

	_, err := store.LoadToken()
	assert.Error(t, err)
}

func TestFileTokenStore_SaveToken_CreatesDirectory(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "token-store-test")
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	tokenPath := filepath.Join(tmpDir, "nested", "dir", "token.json")
	store := NewFileTokenStore(tokenPath)

	testToken := &oauth2.Token{
		AccessToken: "test-token",
	}

	// Save should create the nested directory
	err = store.SaveToken(testToken)
	assert.NoError(t, err)

	// Verify the file exists
	_, err = os.Stat(tokenPath)
	assert.NoError(t, err)
}
