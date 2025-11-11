package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// ProgressAdapter adapts progress callbacks to bubbletea messages
type ProgressAdapter struct {
	program *tea.Program
}

// NewProgressAdapter creates a new progress adapter
func NewProgressAdapter(program *tea.Program) *ProgressAdapter {
	return &ProgressAdapter{
		program: program,
	}
}

// OnStageStart is called when a stage starts
func (p *ProgressAdapter) OnStageStart(stage string) {
	if p.program != nil {
		p.program.Send(StageUpdateMsg{
			StageName: stage,
			Status:    StatusInProgress,
			Progress:  0,
		})
	}
}

// OnStageProgress is called when stage progress updates
func (p *ProgressAdapter) OnStageProgress(stage string, progress int, message string) {
	if p.program != nil {
		p.program.Send(StageUpdateMsg{
			StageName: stage,
			Status:    StatusInProgress,
			Progress:  progress,
			Message:   message,
		})
	}
}

// OnStageComplete is called when a stage completes
func (p *ProgressAdapter) OnStageComplete(stage string, success bool, message string) {
	if p.program != nil {
		p.program.Send(StageCompleteMsg{
			StageName: stage,
			Failed:    !success,
			Message:   message,
		})
	}
}

// OnItemStart is called when an item starts
func (p *ProgressAdapter) OnItemStart(stage string, item string) {
	if p.program != nil {
		p.program.Send(StageUpdateMsg{
			StageName:  stage,
			Status:     StatusInProgress,
			ItemName:   item,
			ItemStatus: StatusInProgress,
		})
	}
}

// OnItemProgress is called when item progress updates
func (p *ProgressAdapter) OnItemProgress(stage string, item string, progress int, message string) {
	if p.program != nil {
		p.program.Send(StageUpdateMsg{
			StageName:    stage,
			Status:       StatusInProgress,
			ItemName:     item,
			ItemStatus:   StatusInProgress,
			ItemProgress: progress,
			ItemMessage:  message,
		})
	}
}

// OnItemComplete is called when an item completes
func (p *ProgressAdapter) OnItemComplete(stage string, item string, success bool, message string) {
	if p.program != nil {
		status := StatusCompleted
		if !success {
			status = StatusFailed
		}
		p.program.Send(StageUpdateMsg{
			StageName:   stage,
			Status:      StatusInProgress,
			ItemName:    item,
			ItemStatus:  status,
			ItemMessage: message,
		})
	}
}
