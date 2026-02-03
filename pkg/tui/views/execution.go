package views

import (
	"bufio"
	"io"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/theshedman/shedman/pkg/tui/theme"
)

type ExecutionModel struct {
	Viewport viewport.Model
	Output   strings.Builder
	Running  bool
	Done     bool
	Title    string

	// Internal state
	cmd    *exec.Cmd
	reader *bufio.Reader
	stdin  io.WriteCloser
}

func NewExecutionModel() ExecutionModel {
	vp := viewport.New(0, 0)
	vp.Style = theme.ContentStyle
	return ExecutionModel{
		Viewport: vp,
		Running:  false,
	}
}

func (m ExecutionModel) Init() tea.Cmd {
	return nil
}

// Msg types
type ExecutionStartRequestMsg struct {
	Command *exec.Cmd
	Title   string
}

type ExecutionStartedMsg struct {
	Title string
}

type ExecutionChunkMsg struct {
	Content []byte
}

type ExecutionFinishedMsg struct {
	Err error
}

func (m ExecutionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Viewport.Width = msg.Width - 4
		m.Viewport.Height = msg.Height - 4
		m.Viewport.SetContent(m.Output.String())

	case ExecutionStartRequestMsg:
		m.cmd = msg.Command
		m.Title = msg.Title // Store title early
		m.Output.Reset()

		stdout, err := m.cmd.StdoutPipe()
		if err != nil {
			return m, func() tea.Msg { return ExecutionFinishedMsg{Err: err} }
		}
		m.cmd.Stderr = m.cmd.Stdout // Merge stderr

		// Setup Stdin for interactive confirmation
		stdin, err := m.cmd.StdinPipe()
		if err != nil {
			return m, func() tea.Msg { return ExecutionFinishedMsg{Err: err} }
		}
		m.stdin = stdin

		if err := m.cmd.Start(); err != nil {
			return m, func() tea.Msg { return ExecutionFinishedMsg{Err: err} }
		}

		m.reader = bufio.NewReader(stdout)
		m.Running = true
		m.Done = false

		// Signal started and begin reading
		return m, tea.Batch(
			func() tea.Msg { return ExecutionStartedMsg{Title: msg.Title} },
			waitForOutput(m.reader),
		)

	case ExecutionStartedMsg:
		m.Viewport.SetContent("Starting " + msg.Title + "...\n")

	case ExecutionChunkMsg:
		if IsClosed(msg.Content) {
			return m, nil
		}

		m.Output.Write(msg.Content)
		m.Viewport.SetContent(m.Output.String())
		if m.Viewport.Height > 0 && m.Viewport.Width > 0 {
			m.Viewport.GotoBottom()
		}

		// Continue reading
		return m, waitForOutput(m.reader)

	case ExecutionFinishedMsg:
		m.Running = false
		m.Done = true
		m.stdin = nil // Close stdin ref
		if msg.Err != nil {
			m.Output.WriteString("\n\nFailed: " + msg.Err.Error())
		} else {
			m.Output.WriteString("\n\nDone.")
		}
		m.Viewport.SetContent(m.Output.String())
		m.Viewport.GotoBottom()
		m.cmd = nil
		m.reader = nil

	case tea.KeyMsg:
		if m.Running && m.stdin != nil {
			// Forward keys to subprocess
			// Note: Write errors are logged but not propagated to avoid interrupting
			// the user's typing flow. The subprocess will handle EOF/broken pipe.
			var writeErr error
			switch msg.String() {
			case "enter":
				_, writeErr = m.stdin.Write([]byte("\n"))
			case "backspace":
				_, writeErr = m.stdin.Write([]byte("\x7f"))
			default:
				if len(msg.Runes) > 0 {
					_, writeErr = m.stdin.Write([]byte(string(msg.Runes)))
				}
			}
			if writeErr != nil {
				// Stdin write failed - subprocess may have exited
				// Continue gracefully, the finish message will arrive shortly
				m.Output.WriteString("\n[input write failed: " + writeErr.Error() + "]\n")
				m.Viewport.SetContent(m.Output.String())
			}
			return m, nil
		}

		if m.Done {
			if msg.String() == "enter" || msg.String() == "esc" || msg.String() == "q" {
				return m, func() tea.Msg { return CanceledMsg{} }
			}
		}
		m.Viewport, cmd = m.Viewport.Update(msg)
		return m, cmd
	}

	return m, nil
}

func waitForOutput(r *bufio.Reader) tea.Cmd {
	return func() tea.Msg {
		if r == nil {
			return nil
		}
		buf := make([]byte, 1024)
		n, err := r.Read(buf)
		if n > 0 {
			// Copy buffer to avoid race if reused (though we make new one)
			c := make([]byte, n)
			copy(c, buf[:n])
			return ExecutionChunkMsg{Content: c}
		}
		if err != nil {
			if err == io.EOF {
				return ExecutionFinishedMsg{Err: nil}
			}
			return ExecutionFinishedMsg{Err: err}
		}
		return nil // Should not happen
	}
}

func IsClosed(b []byte) bool {
	return len(b) == 0
}

func (m ExecutionModel) View() string {
	header := theme.TitleStyle.Render(m.Title)
	if m.Running {
		header += " (Running...)"
	} else if m.Done {
		header += " (Finished - Press Enter to Close)"
	}

	return header + "\n" + theme.ContentStyle.Render(m.Viewport.View())
}
