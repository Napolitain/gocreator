package services

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestNewVideoService(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := &mockLogger{}
	
	service := NewVideoService(fs, logger)
	
	assert.NotNil(t, service)
	assert.Equal(t, fs, service.fs)
	assert.Equal(t, logger, service.logger)
}

// Note: isVideoFile, getMediaDimensions, concatenateVideos, and generateSingleVideo
// all depend on ffmpeg/ffprobe being installed and available.
// These functions are tested in integration tests but cannot be easily unit tested
// without mocking the exec.Command functionality or having ffmpeg installed.
