package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ProgressModel represents the progress UI model
type ProgressModel struct {
	stages       []Stage
	currentStage int
	width        int
	height       int
	quitting     bool
	startTime    time.Time
}

// Stage represents a stage in the video creation process
type Stage struct {
	Name     string
	Status   StageStatus
	Progress int // 0-100
	Message  string
	Items    []StageItem
}

// StageItem represents an item within a stage (e.g., language being processed)
type StageItem struct {
	Name     string
	Status   StageStatus
	Message  string
	Progress int // 0-100
}

// StageStatus represents the status of a stage
type StageStatus int

const (
	StatusPending StageStatus = iota
	StatusInProgress
	StatusCompleted
	StatusFailed
	StatusSkipped
)

// String returns the string representation of a status
func (s StageStatus) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusInProgress:
		return "in progress"
	case StatusCompleted:
		return "completed"
	case StatusFailed:
		return "failed"
	case StatusSkipped:
		return "skipped"
	default:
		return "unknown"
	}
}

// Style definitions
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00FF00")).
			MarginBottom(1)

	stageStyle = lipgloss.NewStyle().
			MarginLeft(2).
			MarginBottom(1)

	itemStyle = lipgloss.NewStyle().
			MarginLeft(4)

	progressBarStyle = lipgloss.NewStyle().
				MarginLeft(2)

	statusPendingStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888"))

	statusInProgressStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFD700")).
				Bold(true)

	statusCompletedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00FF00"))

	statusFailedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF0000")).
				Bold(true)

	statusSkippedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888"))
)

// NewProgressModel creates a new progress model
func NewProgressModel() *ProgressModel {
	return &ProgressModel{
		stages: []Stage{
			{Name: "Loading", Status: StatusPending},
			{Name: "Translation", Status: StatusPending},
			{Name: "Audio Generation", Status: StatusPending},
			{Name: "Video Assembly", Status: StatusPending},
		},
		width:     80,
		height:    24,
		startTime: time.Now(),
	}
}

// Init initializes the model
func (m *ProgressModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m *ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case StageUpdateMsg:
		m.updateStage(msg)

	case StageCompleteMsg:
		m.completeStage(msg)

	case CompleteMsg:
		m.quitting = true
		return m, tea.Quit
	}

	return m, nil
}

// View renders the UI
func (m *ProgressModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Title
	elapsed := time.Since(m.startTime).Round(time.Second)
	title := fmt.Sprintf("GoCreator - Video Generation (Elapsed: %s)", elapsed)
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	// Stages
	for i, stage := range m.stages {
		b.WriteString(m.renderStage(stage, i == m.currentStage))
		b.WriteString("\n")
	}

	// Overall progress
	totalProgress := m.calculateOverallProgress()
	b.WriteString("\n")
	b.WriteString(m.renderProgressBar("Overall Progress", totalProgress, 40))
	b.WriteString("\n\n")

	// Footer
	b.WriteString("Press q to quit (will not stop video generation)\n")

	return b.String()
}

// renderStage renders a single stage
func (m *ProgressModel) renderStage(stage Stage, isCurrent bool) string {
	var b strings.Builder

	// Stage icon and name
	icon := m.getStatusIcon(stage.Status)
	style := m.getStatusStyle(stage.Status)
	
	stageLine := fmt.Sprintf("%s %s", icon, stage.Name)
	if stage.Progress > 0 && stage.Status == StatusInProgress {
		stageLine += fmt.Sprintf(" [%d%%]", stage.Progress)
	}
	if stage.Message != "" {
		stageLine += fmt.Sprintf(" - %s", stage.Message)
	}

	b.WriteString(stageStyle.Render(style.Render(stageLine)))
	b.WriteString("\n")

	// Stage items
	if len(stage.Items) > 0 && (stage.Status == StatusInProgress || stage.Status == StatusCompleted) {
		for _, item := range stage.Items {
			itemIcon := m.getStatusIcon(item.Status)
			itemStyle := m.getStatusStyle(item.Status)
			
			itemLine := fmt.Sprintf("%s %s", itemIcon, item.Name)
			if item.Progress > 0 && item.Status == StatusInProgress {
				itemLine += fmt.Sprintf(" (%d%%)", item.Progress)
			}
			if item.Message != "" {
				itemLine += fmt.Sprintf(": %s", item.Message)
			}
			
			b.WriteString(itemStyle.Render(itemStyle.Render(itemLine)))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// renderProgressBar renders a progress bar
func (m *ProgressModel) renderProgressBar(label string, progress int, width int) string {
	filled := int(float64(width) * float64(progress) / 100.0)
	empty := width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	return progressBarStyle.Render(fmt.Sprintf("%s [%s] %d%%", label, bar, progress))
}

// getStatusIcon returns the icon for a status
func (m *ProgressModel) getStatusIcon(status StageStatus) string {
	switch status {
	case StatusPending:
		return "⋯"
	case StatusInProgress:
		return "→"
	case StatusCompleted:
		return "✓"
	case StatusFailed:
		return "✗"
	case StatusSkipped:
		return "○"
	default:
		return "?"
	}
}

// getStatusStyle returns the style for a status
func (m *ProgressModel) getStatusStyle(status StageStatus) lipgloss.Style {
	switch status {
	case StatusPending:
		return statusPendingStyle
	case StatusInProgress:
		return statusInProgressStyle
	case StatusCompleted:
		return statusCompletedStyle
	case StatusFailed:
		return statusFailedStyle
	case StatusSkipped:
		return statusSkippedStyle
	default:
		return lipgloss.NewStyle()
	}
}

// calculateOverallProgress calculates overall progress
func (m *ProgressModel) calculateOverallProgress() int {
	if len(m.stages) == 0 {
		return 0
	}

	total := 0
	for _, stage := range m.stages {
		switch stage.Status {
		case StatusCompleted:
			total += 100
		case StatusInProgress:
			total += stage.Progress
		case StatusSkipped:
			total += 100
		}
	}

	return total / len(m.stages)
}

// updateStage updates a stage
func (m *ProgressModel) updateStage(msg StageUpdateMsg) {
	for i := range m.stages {
		if m.stages[i].Name == msg.StageName {
			m.stages[i].Status = msg.Status
			m.stages[i].Progress = msg.Progress
			m.stages[i].Message = msg.Message
			
			if msg.Status == StatusInProgress {
				m.currentStage = i
			}
			
			// Update or add item
			if msg.ItemName != "" {
				found := false
				for j := range m.stages[i].Items {
					if m.stages[i].Items[j].Name == msg.ItemName {
						m.stages[i].Items[j].Status = msg.ItemStatus
						m.stages[i].Items[j].Progress = msg.ItemProgress
						m.stages[i].Items[j].Message = msg.ItemMessage
						found = true
						break
					}
				}
				if !found {
					m.stages[i].Items = append(m.stages[i].Items, StageItem{
						Name:     msg.ItemName,
						Status:   msg.ItemStatus,
						Progress: msg.ItemProgress,
						Message:  msg.ItemMessage,
					})
				}
			}
			break
		}
	}
}

// completeStage marks a stage as complete
func (m *ProgressModel) completeStage(msg StageCompleteMsg) {
	for i := range m.stages {
		if m.stages[i].Name == msg.StageName {
			if msg.Failed {
				m.stages[i].Status = StatusFailed
			} else {
				m.stages[i].Status = StatusCompleted
			}
			m.stages[i].Progress = 100
			m.stages[i].Message = msg.Message
			break
		}
	}
}

// Messages

// StageUpdateMsg updates a stage
type StageUpdateMsg struct {
	StageName    string
	Status       StageStatus
	Progress     int
	Message      string
	ItemName     string
	ItemStatus   StageStatus
	ItemProgress int
	ItemMessage  string
}

// StageCompleteMsg marks a stage as complete
type StageCompleteMsg struct {
	StageName string
	Failed    bool
	Message   string
}

// CompleteMsg signals completion
type CompleteMsg struct{}
