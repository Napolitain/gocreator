package auth

import (
	"context"

	"google.golang.org/api/option"
)

// ServiceAccountProvider provides service account credentials for Google API services
type ServiceAccountProvider struct {
	credentialsPath string
}

// NewServiceAccountProvider creates a new ServiceAccountProvider
func NewServiceAccountProvider(credentialsPath string) *ServiceAccountProvider {
	return &ServiceAccountProvider{
		credentialsPath: credentialsPath,
	}
}

// GetClientOption returns the Google API client option with service account credentials
func (p *ServiceAccountProvider) GetClientOption(ctx context.Context) (option.ClientOption, error) {
	return option.WithCredentialsFile(p.credentialsPath), nil
}
