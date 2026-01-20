package tui

// sessionState defines the current active view
type sessionState int

const (
	viewDashboard sessionState = iota
	viewSearch
	viewUpdates
	viewSnapshots
	viewHealth
	viewDesktop
	viewPassword
	viewExecution
)

// Messages
