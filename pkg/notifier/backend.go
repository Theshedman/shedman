package notifier

// NotifierBackend defines the interface for desktop notifications
type NotifierBackend interface {
	Name() string
	Notify(title, message, level string) error
	IsAvailable() bool
}

// CheckResult represents the result of an update check
type CheckResult struct {
	Available bool
	Count     int
	Message   string
}
