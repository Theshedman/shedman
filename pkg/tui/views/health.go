package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/tui/theme"
)

type HealthModel struct {
	viewport viewport.Model
	engine   *core.Engine
	report   core.SystemReport
	ready    bool
	err      error
}

func NewHealthModel(eng *core.Engine) HealthModel {
	return HealthModel{
		engine: eng,
	}
}

// Custom msg
type HealthCheckFinishedMsg struct {
	Report core.SystemReport
}

func (m HealthModel) Init() tea.Cmd {
	return func() tea.Msg {
		if m.engine == nil {
			// Should fail differently?
			return HealthCheckFinishedMsg{
				Report: core.SystemReport{
					Items: []core.DiagnoseItem{{Name: "Init", Status: core.DiagnoseStatusFail, Message: "Engine nil"}},
				},
			}
		}

		// Temporary: Simple checks since core logic refactor is partial or we need to construct checks
		checks := core.DoctorChecks{
			CheckConnection: func() bool { return true },
			CheckLockFile:   func() bool { return false },
			CheckDiskSpace:  func(p string) float64 { return 100 },
			CheckServices:   func() []string { return nil },
		}

		return HealthCheckFinishedMsg{Report: core.Diagnose(m.engine, checks)}
	}
}

func (m HealthModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height)
			m.ready = true
		}
	case HealthCheckFinishedMsg:
		m.report = msg.Report
		m.viewport.SetContent(m.renderReport())
	case error: // Fallback for other errors (unlikely now from init)
		m.err = msg
		m.viewport.SetContent(fmt.Sprintf("\n%s\n", theme.ErrorStyle.Render("Diagnosis Failed: "+msg.Error())))
	}
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m HealthModel) View() string {
	if !m.ready {
		return "Initializing Health Check..."
	}
	return m.viewport.View()
}

func (m HealthModel) renderReport() string {
	s := strings.Builder{}
	s.WriteString(theme.TitleStyle.Render(" System Health Report ") + "\n\n")

	for _, item := range m.report.Items {
		icon := "✓"
		style := theme.SuccessStyle
		switch item.Status {
		case core.DiagnoseStatusFail:
			icon = "✗"
			style = theme.ErrorStyle
		case core.DiagnoseStatusWarn:
			icon = "!"
			style = theme.HighlightStyle
		}

		s.WriteString(fmt.Sprintf(" %s %s: %s\n", style.Render(icon), item.Name, item.Message))
	}

	if len(m.report.Items) == 0 {
		s.WriteString(" No checks ran.")
	}

	return s.String()
}
