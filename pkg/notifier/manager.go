package notifier

import "github.com/theshedman/shedman/pkg/core"

// Notifier handles update notifications
type Notifier struct {
	core *core.Engine
}

// New creates a new notifier
func New(c *core.Engine) *Notifier {
	return &Notifier{
		core: c,
	}
}

// Check checks for updates
func (n *Notifier) Check() error {
	return nil
}

// Notify sends a notification
func (n *Notifier) Notify(msg string) error {
	return nil
}
