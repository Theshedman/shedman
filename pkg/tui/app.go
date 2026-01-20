package tui

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/de"
	"github.com/theshedman/shedman/pkg/tui/theme"
	"github.com/theshedman/shedman/pkg/tui/views"
)

// App represents the TUI application
type App struct {
	core  *core.Engine
	deMgr *de.Manager
}

// New creates a new TUI app
func New(c *core.Engine, dm *de.Manager) *App {
	return &App{
		core:  c,
		deMgr: dm,
	}
}

// Run runs the TUI
func (a *App) Run() error {
	m := newAppModel(a.core, a.deMgr)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("tui failed: %w", err)
	}
	return nil
}

// appModel is the top-level model that manages the split-pane layout
type appModel struct {
	core    *core.Engine
	deMgr   *de.Manager
	sidebar list.Model
	active  sessionState // Current view displayed in main pane
	focus   int          // 0 = Sidebar, 1 = Main Pane
	width   int
	height  int

	// Sub-models
	// Sub-models
	search    views.SearchModel
	health    views.HealthModel
	updates   views.UpdatesModel
	snapshots views.SnapshotsModel
	desktop   views.DesktopModel
	password  views.PasswordModel
	execution views.ExecutionModel
}

func newAppModel(c *core.Engine, dm *de.Manager) appModel {
	// Initialize Sidebar
	items := []list.Item{
		dashboardItem{title: "Dashboard", desc: "Overview", view: viewDashboard},
		dashboardItem{title: "Search", desc: "Browse Packages", view: viewSearch},
		dashboardItem{title: "Updates", desc: "System Updates", view: viewUpdates},
		dashboardItem{title: "Snapshots", desc: "Backup & Restore", view: viewSnapshots},
		dashboardItem{title: "Health", desc: "System Diagnostic", view: viewHealth},
		dashboardItem{title: "Desktop", desc: "Manage Environment", view: viewDesktop},
	}

	l := list.New(items, dashboardDelegate{}, 20, 14)
	l.Title = "Modules"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = theme.TitleStyle
	l.Styles.PaginationStyle = list.DefaultStyles().PaginationStyle.PaddingLeft(2)
	l.Styles.HelpStyle = list.DefaultStyles().HelpStyle.PaddingLeft(2)

	return appModel{
		core:      c,
		deMgr:     dm,
		sidebar:   l,
		active:    viewDashboard,
		focus:     0, // Start focused on Sidebar
		search:    views.NewSearchModel(c),
		health:    views.NewHealthModel(c),
		updates:   views.NewUpdatesModel(c),
		snapshots: views.NewSnapshotsModel(c),
		desktop:   views.NewDesktopModel(c, dm),
		password:  views.NewPasswordModel(),
		execution: views.NewExecutionModel(),
	}
}

func (m appModel) Init() tea.Cmd {
	return tea.Batch(
		m.search.Init(),
		m.health.Init(),
		m.updates.Init(),
		m.snapshots.Init(),
		m.desktop.Init(),
		m.password.Init(),
		m.execution.Init(),
	)
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "tab":
			// Toggle Focus
			m.focus = (m.focus + 1) % 2
			return m, nil
		case "enter":
			if m.focus == 0 {
				// Sidebar Selection
				i, ok := m.sidebar.SelectedItem().(dashboardItem)
				if ok {
					m.active = i.view
					m.focus = 1 // Auto-focus content
				}
			}
		case "q":
			if m.focus == 0 { // Only quit if in sidebar to avoid quitting search
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Layout logic: 30% sidebar, 70% content
		sidebarWidth := int(float64(msg.Width) * 0.3)
		contentWidth := msg.Width - sidebarWidth - 6 // minus borders/margins

		// Sidebar Height: app height - 2 (border)
		// We enforce size on sidebar list
		m.sidebar.SetSize(sidebarWidth-2, msg.Height-4)

		// Create adjusted message for content views
		// Content border takes 2 lines, so we give content H-2
		// But inside content, there might be more padding.
		// Let's give H-4 to be safe and ensure footer is visible.
		contentMsg := tea.WindowSizeMsg{
			Width:  contentWidth - 2, // Accounting for internal padding
			Height: msg.Height - 4,
		}

		// Propagate adjusted size to all views (so they resize correctly even if not active)
		// This keeps state consistent when switching tabs
		var cmd tea.Cmd

		// Search
		var sMsg tea.Model
		sMsg, cmd = m.search.Update(contentMsg)
		m.search = sMsg.(views.SearchModel)
		cmds = append(cmds, cmd)

		// Health
		var hMsg tea.Model
		hMsg, cmd = m.health.Update(contentMsg)
		m.health = hMsg.(views.HealthModel)
		cmds = append(cmds, cmd)

		// Updates
		var uMsg tea.Model
		uMsg, cmd = m.updates.Update(contentMsg)
		m.updates = uMsg.(views.UpdatesModel)
		cmds = append(cmds, cmd)

		// Snapshots
		var snMsg tea.Model
		snMsg, cmd = m.snapshots.Update(contentMsg)
		m.snapshots = snMsg.(views.SnapshotsModel)
		cmds = append(cmds, cmd)

		// Desktop
		var dMsg tea.Model
		dMsg, cmd = m.desktop.Update(contentMsg)
		m.desktop = dMsg.(views.DesktopModel)
		cmds = append(cmds, cmd)

		// Password
		// We don't always need to resize password view, but consistent state is good
		m.password.Width = contentWidth // Assuming we add Width to PasswordModel
		// m.password, cmd = m.password.Update(contentMsg)
		// Actually PasswordModel doesn't use viewport yet, but might later.
		// Let's call Update just in case.
		var pMsg tea.Model
		pMsg, cmd = m.password.Update(contentMsg)
		m.password = pMsg.(views.PasswordModel)
		cmds = append(cmds, cmd)

		// Execution
		var eMsg tea.Model
		eMsg, cmd = m.execution.Update(contentMsg)
		m.execution = eMsg.(views.ExecutionModel)
		cmds = append(cmds, cmd)

		return m, tea.Batch(cmds...)
	}

	// Handle global/background messages
	switch msg := msg.(type) {
	case views.UpdatesFinishedMsg:
		var uMsg tea.Model
		uMsg, cmd = m.updates.Update(msg)
		m.updates = uMsg.(views.UpdatesModel)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	case views.SnapshotsFinishedMsg:
		var snMsg tea.Model
		snMsg, cmd = m.snapshots.Update(msg)
		m.snapshots = snMsg.(views.SnapshotsModel)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	case views.HealthCheckFinishedMsg:
		var hMsg tea.Model
		hMsg, cmd = m.health.Update(msg)
		m.health = hMsg.(views.HealthModel)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	case views.DesktopListFinishedMsg:
		var dMsg tea.Model
		dMsg, cmd = m.desktop.Update(msg)
		m.desktop = dMsg.(views.DesktopModel)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	// Embedded Execution Requests
	case views.RequestUpdateMsg:
		// Check sudo access
		err := exec.Command("sudo", "-n", "true").Run()
		if err == nil {
			// Sudo available without password
			m.active = viewExecution
			m.focus = 1
			// Start Updates
			updateCmd := exec.Command("sudo", "shedman", "update")

			var eMsg tea.Model
			eMsg, cmd = m.execution.Update(views.ExecutionStartRequestMsg{
				Command: updateCmd,
				Title:   "System Update",
			})
			m.execution = eMsg.(views.ExecutionModel)
			cmds = append(cmds, cmd)
		} else {
			// Need password
			m.active = viewPassword
			m.focus = 1
			m.password.Input.Focus()
			m.password.NextCmd = func(pwd string) tea.Cmd {
				return func() tea.Msg {
					updateCmd := exec.Command("sudo", "shedman", "update")
					return views.ExecutionStartRequestMsg{
						Command: updateCmd,
						Title:   "System Update",
					}
				}
			}
		}
		return m, tea.Batch(cmds...)

	case views.RequestInstallMsg:
		// Check sudo access
		err := exec.Command("sudo", "-n", "true").Run()
		if err == nil {
			// Sudo available without password
			m.active = viewExecution
			m.focus = 1

			installCmd := exec.Command("sudo", "shedman", "install", msg.Package)

			var eMsg tea.Model
			eMsg, cmd = m.execution.Update(views.ExecutionStartRequestMsg{
				Command: installCmd,
				Title:   "Installing " + msg.Package,
			})
			m.execution = eMsg.(views.ExecutionModel)
			cmds = append(cmds, cmd)
		} else {
			// Need password
			m.active = viewPassword
			m.focus = 1
			m.password.Input.Focus()
			m.password.NextCmd = func(pwd string) tea.Cmd {
				return func() tea.Msg {
					installCmd := exec.Command("sudo", "shedman", "install", msg.Package)
					return views.ExecutionStartRequestMsg{
						Command: installCmd,
						Title:   "Installing " + msg.Package,
					}
				}
			}
		}
		return m, tea.Batch(cmds...)

	case views.RequestSnapshotCreateMsg:
		// Check sudo access
		err := exec.Command("sudo", "-n", "true").Run()
		if err == nil {
			m.active = viewExecution
			m.focus = 1

			createCmd := exec.Command("sudo", "shedman", "snapshot", "create")

			var eMsg tea.Model
			eMsg, cmd = m.execution.Update(views.ExecutionStartRequestMsg{
				Command: createCmd,
				Title:   "Creating Snapshot",
			})
			m.execution = eMsg.(views.ExecutionModel)
			cmds = append(cmds, cmd)
		} else {
			m.active = viewPassword
			m.focus = 1
			m.password.Input.Focus()
			m.password.NextCmd = func(pwd string) tea.Cmd {
				return func() tea.Msg {
					createCmd := exec.Command("sudo", "shedman", "snapshot", "create")
					return views.ExecutionStartRequestMsg{
						Command: createCmd,
						Title:   "Creating Snapshot",
					}
				}
			}
		}
		return m, tea.Batch(cmds...)

	case views.RequestSnapshotPushMsg:
		// Check sudo access
		err := exec.Command("sudo", "-n", "true").Run()
		if err == nil {
			m.active = viewExecution
			m.focus = 1

			pushCmd := exec.Command("sudo", "shedman", "snapshot", "remote", "push", msg.ID)

			var eMsg tea.Model
			eMsg, cmd = m.execution.Update(views.ExecutionStartRequestMsg{
				Command: pushCmd,
				Title:   "Pushing Snapshot " + msg.ID,
			})
			m.execution = eMsg.(views.ExecutionModel)
			cmds = append(cmds, cmd)
		} else {
			m.active = viewPassword
			m.focus = 1
			m.password.Input.Focus()
			m.password.NextCmd = func(pwd string) tea.Cmd {
				return func() tea.Msg {
					pushCmd := exec.Command("sudo", "shedman", "snapshot", "remote", "push", msg.ID)
					return views.ExecutionStartRequestMsg{
						Command: pushCmd,
						Title:   "Pushing Snapshot " + msg.ID,
					}
				}
			}
		}

		return m, tea.Batch(cmds...)

	case views.RequestSnapshotRestoreMsg:
		// Check sudo access
		err := exec.Command("sudo", "-n", "true").Run()
		if err == nil {
			m.active = viewExecution
			m.focus = 1

			restoreCmd := exec.Command("sudo", "shedman", "snapshot", "restore", msg.ID)

			var eMsg tea.Model
			eMsg, cmd = m.execution.Update(views.ExecutionStartRequestMsg{
				Command: restoreCmd,
				Title:   "Restoring Snapshot " + msg.ID,
			})
			m.execution = eMsg.(views.ExecutionModel)
			cmds = append(cmds, cmd)
		} else {
			m.active = viewPassword
			m.focus = 1
			m.password.Input.Focus()
			m.password.NextCmd = func(pwd string) tea.Cmd {
				return func() tea.Msg {
					restoreCmd := exec.Command("sudo", "shedman", "snapshot", "restore", msg.ID)
					return views.ExecutionStartRequestMsg{
						Command: restoreCmd,
						Title:   "Restoring Snapshot " + msg.ID,
					}
				}
			}
		}
		return m, tea.Batch(cmds...)

	case views.PasswordProvidedMsg:
		// Validate Password
		cmdVal := exec.Command("sudo", "-S", "-v")
		cmdVal.Stdin = bytes.NewBufferString(msg.Password + "\n")
		var stderr bytes.Buffer
		cmdVal.Stderr = &stderr

		if err := cmdVal.Run(); err != nil {
			m.password.Input.SetValue("")
			m.password.Input.Placeholder = "Invalid Password. Try again."
		} else {
			m.active = viewExecution

			if msg.Cmd != nil {
				cmds = append(cmds, msg.Cmd(msg.Password))
			}
		}
		return m, tea.Batch(cmds...)

	case views.CanceledMsg:
		m.active = viewUpdates
		m.focus = 1
		// Optionally trigger reload of updates
		var uMsg tea.Model
		uMsg, cmd = m.updates.Update("reload_updates")
		m.updates = uMsg.(views.UpdatesModel)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case views.ExecutionStartedMsg, views.ExecutionChunkMsg, views.ExecutionFinishedMsg:
		// Route to execution view
		var eMsg tea.Model
		eMsg, cmd = m.execution.Update(msg)
		m.execution = eMsg.(views.ExecutionModel)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	}

	// Update Focused Component
	if m.focus == 0 {
		m.sidebar, cmd = m.sidebar.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		// Update Active View
		switch m.active {
		case viewSearch:
			var sMsg tea.Model
			sMsg, cmd = m.search.Update(msg)
			m.search = sMsg.(views.SearchModel)
			cmds = append(cmds, cmd)
		case viewHealth:
			// Already handled globally if it's a report, but passed for other events (keys)
			var hMsg tea.Model
			hMsg, cmd = m.health.Update(msg)
			m.health = hMsg.(views.HealthModel)
			cmds = append(cmds, cmd)
		case viewUpdates:
			var uMsg tea.Model
			uMsg, cmd = m.updates.Update(msg)
			m.updates = uMsg.(views.UpdatesModel)
			cmds = append(cmds, cmd)
		case viewSnapshots:
			var snMsg tea.Model
			snMsg, cmd = m.snapshots.Update(msg)
			m.snapshots = snMsg.(views.SnapshotsModel)
			cmds = append(cmds, cmd)
		case viewDesktop:
			var dMsg tea.Model
			dMsg, cmd = m.desktop.Update(msg)
			m.desktop = dMsg.(views.DesktopModel)
			cmds = append(cmds, cmd)
		case viewPassword:
			var pMsg tea.Model
			pMsg, cmd = m.password.Update(msg)
			m.password = pMsg.(views.PasswordModel)
			cmds = append(cmds, cmd)
		case viewExecution:
			var eMsg tea.Model
			eMsg, cmd = m.execution.Update(msg)
			m.execution = eMsg.(views.ExecutionModel)
			cmds = append(cmds, cmd)
		}
	}

	// Handle view-specific messages
	switch msg.(type) {
	case views.FocusSidebarMsg:
		m.focus = 0
	}

	return m, tea.Batch(cmds...)
}

func (m appModel) View() string {
	// Calculate Dimensions
	sidebarWidth := int(float64(m.width) * 0.3)
	contentWidth := m.width - sidebarWidth - 6

	// Render Sidebar
	sidebarStyle := theme.InactiveBorderStyle.Width(sidebarWidth).Height(m.height - 2)
	if m.focus == 0 {
		sidebarStyle = theme.ActiveBorderStyle.Width(sidebarWidth).Height(m.height - 2)
	}
	sidebarView := sidebarStyle.Render(m.sidebar.View())

	// Render Content
	contentStyle := theme.InactiveBorderStyle.Width(contentWidth).Height(m.height - 2)
	if m.focus == 1 {
		contentStyle = theme.ActiveBorderStyle.Width(contentWidth).Height(m.height - 2)
	}

	var contentView string
	switch m.active {
	case viewDashboard:
		contentView = lipgloss.Place(contentWidth, m.height-4, lipgloss.Center, lipgloss.Center, "Welcome to ShedMan\n\nSelect a module from the sidebar.")
	case viewSearch:
		contentView = m.search.View()
	case viewHealth:
		contentView = m.health.View()
	case viewUpdates:
		contentView = m.updates.View()
	case viewSnapshots:
		contentView = m.snapshots.View()
	case viewDesktop:
		contentView = m.desktop.View()
	case viewPassword:
		contentView = lipgloss.Place(contentWidth, m.height-4, lipgloss.Center, lipgloss.Center, m.password.View())
	case viewExecution:
		contentView = m.execution.View()
	default:
		contentView = lipgloss.Place(contentWidth, m.height-4, lipgloss.Center, lipgloss.Center, "Not Implemented Yet")
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, contentStyle.Render(contentView))
}
