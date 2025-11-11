package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
)

// FileTokenStore stores OAuth tokens in a file
type FileTokenStore struct {
	tokenPath string
}

// NewFileTokenStore creates a new FileTokenStore
func NewFileTokenStore(tokenPath string) *FileTokenStore {
	return &FileTokenStore{
		tokenPath: tokenPath,
	}
}

// LoadToken loads a token from file
func (s *FileTokenStore) LoadToken() (*oauth2.Token, error) {
	f, err := os.Open(s.tokenPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	token := &oauth2.Token{}
	if err := json.NewDecoder(f).Decode(token); err != nil {
		return nil, err
	}

	return token, nil
}

// SaveToken saves a token to file
func (s *FileTokenStore) SaveToken(token *oauth2.Token) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(s.tokenPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create token directory: %w", err)
	}

	f, err := os.OpenFile(s.tokenPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create token file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if err := json.NewEncoder(f).Encode(token); err != nil {
		return fmt.Errorf("failed to encode token: %w", err)
	}

	return nil
}
