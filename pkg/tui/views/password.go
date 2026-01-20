package views

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/theshedman/shedman/pkg/tui/theme"
)

type PasswordModel struct {
	Input    textinput.Model
	Active   bool
	NextCmd  func(string) tea.Cmd // Command to run after password (receives password)
	Canceled bool
	Width    int
}

type PasswordProvidedMsg struct {
	Password string
	Cmd      func(string) tea.Cmd
}

func NewPasswordModel() PasswordModel {
	ti := textinput.New()
	ti.Placeholder = "Sudo Password"
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	ti.Focus()
	ti.CharLimit = 126
	ti.Width = 30

	return PasswordModel{
		Input: ti,
	}
}

func (m PasswordModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m PasswordModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			val := m.Input.Value()
			m.Input.SetValue("") // Clear immediately
			return m, func() tea.Msg {
				return PasswordProvidedMsg{Password: val, Cmd: m.NextCmd}
			}
		case tea.KeyEsc:
			m.Canceled = true
			m.Input.SetValue("")
			return m, func() tea.Msg { return CanceledMsg{} } // Define CanceledMsg
		}
	}

	m.Input, cmd = m.Input.Update(msg)
	return m, cmd
}

func (m PasswordModel) View() string {
	return theme.DialogStyle.Render(
		"Authentication Required\n\n" +
			m.Input.View() + "\n\n" +
			"(Esc to cancel)",
	)
}

type CanceledMsg struct{}
