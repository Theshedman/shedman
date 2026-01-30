package snapshot

import (
	"strconv"
	"strings"
	"time"

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

	out, err := s.exec.Output(
		"systemctl",
		"--user",
		"show",
		"shedman-snapshot.timer",
		"-p", "UnitFileState",
		"-p", "ActiveState",
		"-p", "NextElapseUSecRealtime",
		"-p", "LastTriggerUSecRealtime",
		"-p", "OnCalendar",
	)
	if err != nil {
		activeOut, activeErr := s.exec.Output("systemctl", "--user", "is-active", "shedman-snapshot.timer")
		status.Enabled = activeErr == nil && strings.TrimSpace(string(activeOut)) == "active"
		return status, nil
	}

	props := parseSystemdShow(string(out))
	status.Enabled = isEnabledState(props["UnitFileState"])
	if !status.Enabled && strings.TrimSpace(props["ActiveState"]) == "active" {
		status.Enabled = true
	}

	if next, ok := parseSystemdUSec(props["NextElapseUSecRealtime"]); ok {
		status.NextRun = next
	}
	if last, ok := parseSystemdUSec(props["LastTriggerUSecRealtime"]); ok {
		status.LastRun = last
	}
	if freq := strings.TrimSpace(props["OnCalendar"]); freq != "" && freq != "n/a" {
		status.Frequency = freq
	}

	return status, nil
}

// RunNow triggers the service immediately
func (s *SystemdScheduler) RunNow() error {
	_, err := s.exec.Output("systemctl", "--user", "start", "shedman-snapshot.service")
	return err
}

func parseSystemdShow(output string) map[string]string {
	props := make(map[string]string)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		props[parts[0]] = parts[1]
	}
	return props
}

func parseSystemdUSec(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" || value == "n/a" {
		return time.Time{}, false
	}
	us, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	return time.Unix(0, us*1000), true
}

func isEnabledState(state string) bool {
	switch strings.TrimSpace(state) {
	case "enabled", "enabled-runtime":
		return true
	default:
		return false
	}
}
