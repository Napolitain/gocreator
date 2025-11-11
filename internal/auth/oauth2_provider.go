package auth

import (
	"context"
	"fmt"
	"os"

	"gocreator/internal/interfaces"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

// OAuth2Provider provides OAuth 2.0 credentials for Google API services
type OAuth2Provider struct {
	credentialsPath string
	tokenStore      TokenStore
	authorizer      UserAuthorizer
	logger          interfaces.Logger
	scopes          []string
}

// NewOAuth2Provider creates a new OAuth2Provider
func NewOAuth2Provider(credentialsPath string, tokenStore TokenStore, authorizer UserAuthorizer, logger interfaces.Logger, scopes []string) *OAuth2Provider {
	return &OAuth2Provider{
		credentialsPath: credentialsPath,
		tokenStore:      tokenStore,
		authorizer:      authorizer,
		logger:          logger,
		scopes:          scopes,
	}
}

// GetClientOption returns the Google API client option with OAuth 2.0 credentials
func (p *OAuth2Provider) GetClientOption(ctx context.Context) (option.ClientOption, error) {
	// Read OAuth 2.0 credentials file
	credData, err := os.ReadFile(p.credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read OAuth credentials file: %w", err)
	}

	// Parse OAuth 2.0 config
	config, err := google.ConfigFromJSON(credData, p.scopes...)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OAuth credentials: %w", err)
	}

	// Get or refresh access token
	token, err := p.getToken(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth token: %w", err)
	}

	// Create HTTP client with token
	client := config.Client(ctx, token)

	return option.WithHTTPClient(client), nil
}

// getToken retrieves a token from storage or initiates OAuth flow
func (p *OAuth2Provider) getToken(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	// Try to load token from storage
	token, err := p.tokenStore.LoadToken()
	if err == nil {
		// Token loaded successfully, check if it's valid or refresh it
		if token.Valid() {
			p.logger.Debug("Using cached OAuth token")
			return token, nil
		}

		// Try to refresh the token
		p.logger.Debug("Refreshing OAuth token")
		tokenSource := config.TokenSource(ctx, token)
		newToken, err := tokenSource.Token()
		if err == nil {
			// Save refreshed token
			if saveErr := p.tokenStore.SaveToken(newToken); saveErr != nil {
				p.logger.Error("Failed to save refreshed token", "error", saveErr)
			}
			return newToken, nil
		}
		p.logger.Debug("Failed to refresh token, will request new authorization", "error", err)
	}

	// No valid token, initiate OAuth flow
	p.logger.Info("No valid OAuth token found. Initiating authorization flow...")
	token, err = p.getTokenFromWeb(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to get token from web: %w", err)
	}

	// Save token for future use
	if err := p.tokenStore.SaveToken(token); err != nil {
		p.logger.Error("Failed to save OAuth token", "error", err)
	}

	return token, nil
}

// getTokenFromWeb initiates the OAuth 2.0 authorization flow
func (p *OAuth2Provider) getTokenFromWeb(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	// Generate authorization URL
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)

	// Get authorization code from user
	authCode, err := p.authorizer.GetAuthorizationCode(authURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get authorization code: %w", err)
	}

	// Exchange authorization code for token
	token, err := config.Exchange(ctx, authCode)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange authorization code for token: %w", err)
	}

	fmt.Printf("\n✓ Authorization successful!\n\n")

	return token, nil
}
