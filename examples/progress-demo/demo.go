package main

import (
	"context"
	"fmt"
	"time"

	"gocreator/internal/interfaces"
	"gocreator/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

// MockProgressCallback demonstrates progress tracking
type MockProgressCallback struct {
	program *tea.Program
}

func (m *MockProgressCallback) OnStageStart(stage string) {
	if m.program != nil {
		m.program.Send(ui.StageUpdateMsg{
			StageName: stage,
			Status:    ui.StatusInProgress,
		})
	}
}

func (m *MockProgressCallback) OnStageProgress(stage string, progress int, message string) {
	if m.program != nil {
		m.program.Send(ui.StageUpdateMsg{
			StageName: stage,
			Status:    ui.StatusInProgress,
			Progress:  progress,
			Message:   message,
		})
	}
}

func (m *MockProgressCallback) OnStageComplete(stage string, success bool, message string) {
	if m.program != nil {
		m.program.Send(ui.StageCompleteMsg{
			StageName: stage,
			Failed:    !success,
			Message:   message,
		})
	}
}

func (m *MockProgressCallback) OnItemStart(stage string, item string) {
	if m.program != nil {
		m.program.Send(ui.StageUpdateMsg{
			StageName:  stage,
			ItemName:   item,
			ItemStatus: ui.StatusInProgress,
		})
	}
}

func (m *MockProgressCallback) OnItemProgress(stage string, item string, progress int, message string) {
	if m.program != nil {
		m.program.Send(ui.StageUpdateMsg{
			StageName:    stage,
			ItemName:     item,
			ItemStatus:   ui.StatusInProgress,
			ItemProgress: progress,
			ItemMessage:  message,
		})
	}
}

func (m *MockProgressCallback) OnItemComplete(stage string, item string, success bool, message string) {
	status := ui.StatusCompleted
	if !success {
		status = ui.StatusFailed
	}
	if m.program != nil {
		m.program.Send(ui.StageUpdateMsg{
			StageName:   stage,
			ItemName:    item,
			ItemStatus:  status,
			ItemMessage: message,
		})
	}
}

func simulateVideoCreation(ctx context.Context, progress interfaces.ProgressCallback) {
	// Loading stage
	progress.OnStageStart("Loading")
	time.Sleep(500 * time.Millisecond)
	progress.OnStageProgress("Loading", 50, "Loading slides...")
	time.Sleep(500 * time.Millisecond)
	progress.OnStageComplete("Loading", true, "Loaded 5 slides")

	// Translation stage
	progress.OnStageStart("Translation")
	languages := []string{"en", "fr", "es"}
	for i, lang := range languages {
		progress.OnItemStart("Translation", lang)
		time.Sleep(300 * time.Millisecond)
		progress.OnItemProgress("Translation", lang, 50, "Translating...")
		time.Sleep(300 * time.Millisecond)
		if lang == "en" {
			progress.OnItemComplete("Translation", lang, true, "Using original")
		} else {
			progress.OnItemComplete("Translation", lang, true, "Translated 5 texts")
		}
		progress.OnStageProgress("Translation", ((i+1)*100)/len(languages), fmt.Sprintf("%d/%d languages", i+1, len(languages)))
	}
	progress.OnStageComplete("Translation", true, "All translations complete")

	// Audio Generation stage
	progress.OnStageStart("Audio Generation")
	for i, lang := range languages {
		progress.OnItemStart("Audio Generation", lang)
		time.Sleep(400 * time.Millisecond)
		progress.OnItemProgress("Audio Generation", lang, 50, "Generating audio...")
		time.Sleep(400 * time.Millisecond)
		progress.OnItemComplete("Audio Generation", lang, true, "Generated 5 files")
		progress.OnStageProgress("Audio Generation", ((i+1)*100)/len(languages), fmt.Sprintf("%d/%d languages", i+1, len(languages)))
	}
	progress.OnStageComplete("Audio Generation", true, "All audio generated")

	// Video Assembly stage
	progress.OnStageStart("Video Assembly")
	for i, lang := range languages {
		progress.OnItemStart("Video Assembly", lang)
		time.Sleep(600 * time.Millisecond)
		progress.OnItemProgress("Video Assembly", lang, 50, "Assembling video...")
		time.Sleep(600 * time.Millisecond)
		progress.OnItemComplete("Video Assembly", lang, true, "Video complete")
		progress.OnStageProgress("Video Assembly", ((i+1)*100)/len(languages), fmt.Sprintf("%d/%d videos", i+1, len(languages)))
	}
	progress.OnStageComplete("Video Assembly", true, "All videos created")
}

func main() {
	// Create progress model
	progressModel := ui.NewProgressModel()
	prog := tea.NewProgram(progressModel)

	// Create progress callback
	progress := &MockProgressCallback{program: prog}

	// Run simulation in background
	go func() {
		ctx := context.Background()
		simulateVideoCreation(ctx, progress)
		time.Sleep(1 * time.Second)
		prog.Send(ui.CompleteMsg{})
	}()

	// Run UI
	if _, err := prog.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	fmt.Println("\n✓ Demo complete!")
}
