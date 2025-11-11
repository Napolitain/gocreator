package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockLogger(t *testing.T) {
	logger := &mockLogger{}
	
	// Test that all methods can be called without panicking
	assert.NotPanics(t, func() {
		logger.Debug("debug message")
		logger.Info("info message")
		logger.Warn("warn message")
		logger.Error("error message")
	})
	
	// Test With method
	newLogger := logger.With("key", "value")
	assert.Equal(t, logger, newLogger)
}
