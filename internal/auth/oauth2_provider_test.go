package auth

import (
	"context"
	"errors"
	"testing"

	"gocreator/internal/interfaces"

	"golang.org/x/oauth2"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTokenStore is a mock implementation of TokenStore
type MockTokenStore struct {
	mock.Mock
}

func (m *MockTokenStore) LoadToken() (*oauth2.Token, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*oauth2.Token), args.Error(1)
}

func (m *MockTokenStore) SaveToken(token *oauth2.Token) error {
	args := m.Called(token)
	return args.Error(0)
}

// MockUserAuthorizer is a mock implementation of UserAuthorizer
type MockUserAuthorizer struct {
	mock.Mock
}

func (m *MockUserAuthorizer) GetAuthorizationCode(authURL string) (string, error) {
	args := m.Called(authURL)
	return args.String(0), args.Error(1)
}

// MockLogger is a mock implementation of Logger
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Debug(msg string, args ...any) {
	m.Called(msg, args)
}

func (m *MockLogger) Info(msg string, args ...any) {
	m.Called(msg, args)
}

func (m *MockLogger) Warn(msg string, args ...any) {
	m.Called(msg, args)
}

func (m *MockLogger) Error(msg string, args ...any) {
	m.Called(msg, args)
}

func (m *MockLogger) With(args ...any) interfaces.Logger {
	return m
}

func TestOAuth2Provider_GetClientOption_InvalidCredentialsFile(t *testing.T) {
	mockTokenStore := new(MockTokenStore)
	mockAuthorizer := new(MockUserAuthorizer)
	mockLogger := new(MockLogger)

	provider := NewOAuth2Provider(
		"/nonexistent/credentials.json",
		mockTokenStore,
		mockAuthorizer,
		mockLogger,
		[]string{"https://www.googleapis.com/auth/presentations"},
	)

	ctx := context.Background()
	_, err := provider.GetClientOption(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read OAuth credentials file")
}

func TestNewOAuth2Provider(t *testing.T) {
	mockTokenStore := new(MockTokenStore)
	mockAuthorizer := new(MockUserAuthorizer)
	mockLogger := new(MockLogger)
	scopes := []string{"https://www.googleapis.com/auth/presentations"}

	provider := NewOAuth2Provider(
		"/path/to/credentials.json",
		mockTokenStore,
		mockAuthorizer,
		mockLogger,
		scopes,
	)

	assert.NotNil(t, provider)
	assert.Equal(t, "/path/to/credentials.json", provider.credentialsPath)
	assert.Equal(t, mockTokenStore, provider.tokenStore)
	assert.Equal(t, mockAuthorizer, provider.authorizer)
	assert.Equal(t, mockLogger, provider.logger)
	assert.Equal(t, scopes, provider.scopes)
}

func TestOAuth2Provider_GetToken_LoadFromCache(t *testing.T) {
	mockTokenStore := new(MockTokenStore)
	mockAuthorizer := new(MockUserAuthorizer)
	mockLogger := new(MockLogger)

	// Mock a valid token
	validToken := &oauth2.Token{
		AccessToken: "test-token",
		TokenType:   "Bearer",
	}

	mockTokenStore.On("LoadToken").Return(validToken, nil)
	mockLogger.On("Debug", "Using cached OAuth token", mock.Anything).Return()

	provider := NewOAuth2Provider(
		"/path/to/credentials.json",
		mockTokenStore,
		mockAuthorizer,
		mockLogger,
		[]string{"https://www.googleapis.com/auth/presentations"},
	)

	// Create a mock config (won't be used since token is valid)
	config := &oauth2.Config{}

	token, err := provider.getToken(context.Background(), config)

	assert.NoError(t, err)
	assert.Equal(t, validToken, token)
	mockTokenStore.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
}

func TestOAuth2Provider_GetToken_NoToken(t *testing.T) {
	mockTokenStore := new(MockTokenStore)
	mockAuthorizer := new(MockUserAuthorizer)
	mockLogger := new(MockLogger)

	mockTokenStore.On("LoadToken").Return(nil, errors.New("token not found"))
	mockLogger.On("Info", "No valid OAuth token found. Initiating authorization flow...", mock.Anything).Return()
	mockAuthorizer.On("GetAuthorizationCode", mock.Anything).Return("", errors.New("user cancelled"))

	provider := NewOAuth2Provider(
		"/path/to/credentials.json",
		mockTokenStore,
		mockAuthorizer,
		mockLogger,
		[]string{"https://www.googleapis.com/auth/presentations"},
	)

	config := &oauth2.Config{
		ClientID: "test-client-id",
	}

	_, err := provider.getToken(context.Background(), config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get token from web")
	mockTokenStore.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
	mockAuthorizer.AssertExpectations(t)
}
