package notifier

import (
	"fmt"
)

// Notifier handles update notifications
type Notifier struct {
	backend NotifierBackend
}

// New creates a new notifier with default libnotify backend
func New() *Notifier {
	return &Notifier{
		backend: NewLibnotifyBackend(),
	}
}

// NewWithBackend creates a notifier with a specific backend
func NewWithBackend(b NotifierBackend) *Notifier {
	return &Notifier{
		backend: b,
	}
}

// Check checks for updates (Stub for future logic)
func (n *Notifier) Check() error {
	return nil
}

// Notify sends a notification
func (n *Notifier) Notify(title, message, level string) error {
	if n.backend == nil {
		return fmt.Errorf("no notifier backend configured")
	}
	return n.backend.Notify(title, message, level)
}
