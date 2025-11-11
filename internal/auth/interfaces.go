package auth

import (
	"context"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

// CredentialsProvider provides credentials for Google API services
type CredentialsProvider interface {
	// GetClientOption returns the Google API client option with credentials
	GetClientOption(ctx context.Context) (option.ClientOption, error)
}

// TokenStore handles OAuth token storage and retrieval
type TokenStore interface {
	// LoadToken loads a token from storage
	LoadToken() (*oauth2.Token, error)
	// SaveToken saves a token to storage
	SaveToken(token *oauth2.Token) error
}

// UserAuthorizer handles user authorization flow
type UserAuthorizer interface {
	// GetAuthorizationCode prompts user and returns authorization code
	GetAuthorizationCode(authURL string) (string, error)
}
