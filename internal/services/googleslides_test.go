package services

import (
	"context"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

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
	assert.Contains(t, err.Error(), "GOOGLE_APPLICATION_CREDENTIALS")
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
