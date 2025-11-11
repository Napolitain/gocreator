package auth

import (
	"fmt"
)

// ConsoleAuthorizer handles user authorization via console
type ConsoleAuthorizer struct{}

// NewConsoleAuthorizer creates a new ConsoleAuthorizer
func NewConsoleAuthorizer() *ConsoleAuthorizer {
	return &ConsoleAuthorizer{}
}

// GetAuthorizationCode prompts user and returns authorization code
func (a *ConsoleAuthorizer) GetAuthorizationCode(authURL string) (string, error) {
	fmt.Printf("\n🔐 Google Slides Authorization Required\n")
	fmt.Printf("═══════════════════════════════════════\n\n")
	fmt.Printf("Please visit this URL to authorize this application:\n\n")
	fmt.Printf("%s\n\n", authURL)
	fmt.Printf("After authorization, you will receive an authorization code.\n")
	fmt.Printf("Enter the authorization code here: ")

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		return "", fmt.Errorf("failed to read authorization code: %w", err)
	}

	return authCode, nil
}
