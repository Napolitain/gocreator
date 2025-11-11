package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestStageStatus_String(t *testing.T) {
	tests := []struct {
		status   StageStatus
		expected string
	}{
		{StatusPending, "pending"},
		{StatusInProgress, "in progress"},
		{StatusCompleted, "completed"},
		{StatusFailed, "failed"},
		{StatusSkipped, "skipped"},
		{StageStatus(999), "unknown"},
	}
	
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.status.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewProgressModel(t *testing.T) {
	model := NewProgressModel()
	
	assert.NotNil(t, model)
	assert.Len(t, model.stages, 4)
	assert.Equal(t, "Loading", model.stages[0].Name)
	assert.Equal(t, "Translation", model.stages[1].Name)
	assert.Equal(t, "Audio Generation", model.stages[2].Name)
	assert.Equal(t, "Video Assembly", model.stages[3].Name)
	
	for _, stage := range model.stages {
		assert.Equal(t, StatusPending, stage.Status)
		assert.Equal(t, 0, stage.Progress)
	}
}

func TestProgressModel_Init(t *testing.T) {
	model := NewProgressModel()
	
	cmd := model.Init()
	assert.Nil(t, cmd)
}

func TestProgressModel_View(t *testing.T) {
	model := NewProgressModel()
	
	view := model.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Loading")
	assert.Contains(t, view, "Translation")
	assert.Contains(t, view, "Audio Generation")
	assert.Contains(t, view, "Video Assembly")
}

func TestProgressModel_Update_StageUpdateMsg(t *testing.T) {
	t.Run("update existing stage", func(t *testing.T) {
		model := NewProgressModel()
		
		msg := StageUpdateMsg{
			StageName: "Loading",
			Status:    StatusInProgress,
			Progress:  50,
			Message:   "halfway",
		}
		
		updatedModel, cmd := model.Update(msg)
		assert.Nil(t, cmd)
		
		pm := updatedModel.(*ProgressModel)
		assert.Equal(t, StatusInProgress, pm.stages[0].Status)
		assert.Equal(t, 50, pm.stages[0].Progress)
		assert.Equal(t, "halfway", pm.stages[0].Message)
	})
	
	t.Run("update non-existent stage", func(t *testing.T) {
		model := NewProgressModel()
		
		msg := StageUpdateMsg{
			StageName: "nonexistent",
			Status:    StatusInProgress,
		}
		
		updatedModel, cmd := model.Update(msg)
		assert.Nil(t, cmd)
		
		pm := updatedModel.(*ProgressModel)
		// Should not panic, stages remain unchanged
		assert.Equal(t, StatusPending, pm.stages[0].Status)
	})
}

func TestProgressModel_Update_StageCompleteMsg(t *testing.T) {
	t.Run("complete stage successfully", func(t *testing.T) {
		model := NewProgressModel()
		
		msg := StageCompleteMsg{
			StageName: "Loading",
			Failed:    false,
			Message:   "done",
		}
		
		updatedModel, cmd := model.Update(msg)
		
		pm := updatedModel.(*ProgressModel)
		assert.Equal(t, StatusCompleted, pm.stages[0].Status)
		assert.Equal(t, "done", pm.stages[0].Message)
		
		// Should not quit until all stages complete
		assert.Nil(t, cmd)
	})
	
	t.Run("fail stage", func(t *testing.T) {
		model := NewProgressModel()
		
		msg := StageCompleteMsg{
			StageName: "Loading",
			Failed:    true,
			Message:   "error occurred",
		}
		
		updatedModel, cmd := model.Update(msg)
		
		pm := updatedModel.(*ProgressModel)
		assert.Equal(t, StatusFailed, pm.stages[0].Status)
		assert.Equal(t, "error occurred", pm.stages[0].Message)
		
		// May or may not quit depending on implementation
		_ = cmd
	})
}

func TestProgressModel_Update_KeyMsg(t *testing.T) {
	model := NewProgressModel()
	
	keyMsg := tea.KeyMsg{
		Type: tea.KeyCtrlC,
	}
	
	_, cmd := model.Update(keyMsg)
	assert.NotNil(t, cmd)
}
