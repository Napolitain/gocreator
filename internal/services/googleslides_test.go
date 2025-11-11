package services

import (
	"context"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/api/option"
)

// MockCredentialsProvider is a mock implementation of auth.CredentialsProvider
type MockCredentialsProvider struct {
	mock.Mock
}

func (m *MockCredentialsProvider) GetClientOption(ctx context.Context) (option.ClientOption, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(option.ClientOption), args.Error(1)
}

func TestGoogleSlidesService_LoadSlides(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewGoogleSlidesService(fs, logger)

	ctx := context.Background()
	slides, err := service.LoadSlides(ctx, "/test/dir")

	assert.Nil(t, slides)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
}

func TestGoogleSlidesService_LoadFromGoogleSlides_MissingCredentials(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewGoogleSlidesService(fs, logger)

	ctx := context.Background()

	// Test without credentials
	slides, notes, err := service.LoadFromGoogleSlides(ctx, "test-presentation-id", "/test/output")

	assert.Nil(t, slides)
	assert.Nil(t, notes)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no Google credentials found")
}

func TestGoogleSlidesService_NewService(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewGoogleSlidesService(fs, logger)

	// Test that the service was created correctly
	assert.NotNil(t, service)
	assert.Equal(t, fs, service.fs)
	assert.Equal(t, logger, service.logger)
}

func TestGoogleSlidesService_NewServiceWithAuth(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	mockProvider := new(MockCredentialsProvider)

	service := NewGoogleSlidesServiceWithAuth(fs, logger, mockProvider)

	// Test that the service was created correctly
	assert.NotNil(t, service)
	assert.Equal(t, fs, service.fs)
	assert.Equal(t, logger, service.logger)
	assert.Equal(t, mockProvider, service.credentialsProvider)
}

func TestGoogleSlidesService_CreateSlidesService_WithProvider(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	mockProvider := new(MockCredentialsProvider)

	// Mock the provider to return an error (since we can't create a real slides service in tests)
	mockProvider.On("GetClientOption", mock.Anything).Return(nil, assert.AnError)

	service := NewGoogleSlidesServiceWithAuth(fs, logger, mockProvider)

	ctx := context.Background()
	_, err := service.createSlidesService(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get client option from credentials provider")
	mockProvider.AssertExpectations(t)
}

func TestGoogleSlidesService_CreateSlidesService_NoProviderNoEnv(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	service := NewGoogleSlidesService(fs, logger)

	ctx := context.Background()
	_, err := service.createSlidesService(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no Google credentials found")
}

func TestNewGoogleSlidesService_Construction(t *testing.T) {
	tests := []struct {
		name string
		fs   afero.Fs
		want bool
	}{
		{
			name: "with mem fs",
			fs:   afero.NewMemMapFs(),
			want: true,
		},
		{
			name: "with os fs",
			fs:   afero.NewOsFs(),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := &mockLogger{}
			service := NewGoogleSlidesService(tt.fs, logger)

			assert.NotNil(t, service)
			assert.Equal(t, tt.want, service != nil)
		})
	}
}
