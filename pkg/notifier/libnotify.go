package notifier

import "os/exec"

// LibnotifyBackend implements NotifierBackend using notify-send
type LibnotifyBackend struct{}

func NewLibnotifyBackend() *LibnotifyBackend {
	return &LibnotifyBackend{}
}

func (l *LibnotifyBackend) Name() string { return "libnotify" }

func (l *LibnotifyBackend) Notify(title, message, level string) error {
	// Map level to urgency
	urgency := "normal"
	switch level {
	case "error":

		urgency = "critical"
	case "info":
		urgency = "low"
	}

	cmd := exec.Command("notify-send", "-u", urgency, title, message)
	return cmd.Run()
}

func (l *LibnotifyBackend) IsAvailable() bool {
	_, err := exec.LookPath("notify-send")
	return err == nil
}
