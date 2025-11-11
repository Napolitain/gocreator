package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNewProgressAdapter(t *testing.T) {
	t.Run("with program", func(t *testing.T) {
		model := NewProgressModel()
		program := tea.NewProgram(model)
		adapter := NewProgressAdapter(program)
		
		assert.NotNil(t, adapter)
		assert.Equal(t, program, adapter.program)
	})
	
	t.Run("with nil program", func(t *testing.T) {
		adapter := NewProgressAdapter(nil)
		
		assert.NotNil(t, adapter)
		assert.Nil(t, adapter.program)
	})
}

func TestProgressAdapter_OnStageStart(t *testing.T) {
	t.Run("with nil program does not panic", func(t *testing.T) {
		adapter := NewProgressAdapter(nil)
		
		// This should not panic
		assert.NotPanics(t, func() {
			adapter.OnStageStart("test-stage")
		})
	})
}

func TestProgressAdapter_OnStageProgress(t *testing.T) {
	t.Run("with nil program does not panic", func(t *testing.T) {
		adapter := NewProgressAdapter(nil)
		
		// This should not panic
		assert.NotPanics(t, func() {
			adapter.OnStageProgress("test-stage", 50, "halfway done")
		})
	})
}

func TestProgressAdapter_OnStageComplete(t *testing.T) {
	t.Run("success with nil program does not panic", func(t *testing.T) {
		adapter := NewProgressAdapter(nil)
		
		// This should not panic
		assert.NotPanics(t, func() {
			adapter.OnStageComplete("test-stage", true, "completed successfully")
		})
	})
	
	t.Run("failure with nil program does not panic", func(t *testing.T) {
		adapter := NewProgressAdapter(nil)
		
		// This should not panic
		assert.NotPanics(t, func() {
			adapter.OnStageComplete("test-stage", false, "failed")
		})
	})
}

func TestProgressAdapter_OnItemStart(t *testing.T) {
	t.Run("with nil program does not panic", func(t *testing.T) {
		adapter := NewProgressAdapter(nil)
		
		// This should not panic
		assert.NotPanics(t, func() {
			adapter.OnItemStart("test-stage", "item1")
		})
	})
}

func TestProgressAdapter_OnItemProgress(t *testing.T) {
	t.Run("with nil program does not panic", func(t *testing.T) {
		adapter := NewProgressAdapter(nil)
		
		// This should not panic
		assert.NotPanics(t, func() {
			adapter.OnItemProgress("test-stage", "item1", 75, "processing")
		})
	})
}

func TestProgressAdapter_OnItemComplete(t *testing.T) {
	t.Run("success with nil program does not panic", func(t *testing.T) {
		adapter := NewProgressAdapter(nil)
		
		// This should not panic
		assert.NotPanics(t, func() {
			adapter.OnItemComplete("test-stage", "item1", true, "item completed")
		})
	})
	
	t.Run("failure with nil program does not panic", func(t *testing.T) {
		adapter := NewProgressAdapter(nil)
		
		// This should not panic
		assert.NotPanics(t, func() {
			adapter.OnItemComplete("test-stage", "item1", false, "item failed")
		})
	})
}
