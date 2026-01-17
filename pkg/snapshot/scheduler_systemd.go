package snapshot

import (
	"strings"

	"github.com/theshedman/shedman/pkg/executor"
)

// SystemdScheduler implements Scheduler using systemd user timers
type SystemdScheduler struct {
	exec executor.Executor
}

// NewSystemdScheduler creates a new scheduler
func NewSystemdScheduler(exec executor.Executor) *SystemdScheduler {
	return &SystemdScheduler{exec: exec}
}

// Enable enables the snapshot timer
func (s *SystemdScheduler) Enable() error {
	// Enable and start the timer
	_, err := s.exec.Output("systemctl", "--user", "enable", "--now", "shedman-snapshot.timer")
	return err
}

// Disable disables the snapshot timer
func (s *SystemdScheduler) Disable() error {
	_, err := s.exec.Output("systemctl", "--user", "disable", "--now", "shedman-snapshot.timer")
	return err
}

// Status returns the current status of the scheduler
func (s *SystemdScheduler) Status() (ScheduleStatus, error) {
	status := ScheduleStatus{}

	// Check if active
	out, err := s.exec.Output("systemctl", "--user", "is-active", "shedman-snapshot.timer")
	status.Enabled = err == nil && strings.TrimSpace(string(out)) == "active"

	return status, nil
}

// RunNow triggers the service immediately
func (s *SystemdScheduler) RunNow() error {
	_, err := s.exec.Output("systemctl", "--user", "start", "shedman-snapshot.service")
	return err
}
