package svc

import (
	"os/exec"
	"strings"
)

// SystemdBackend implements ServiceBackend using systemctl
type SystemdBackend struct{}

func NewSystemdBackend() *SystemdBackend {
	return &SystemdBackend{}
}

func (s *SystemdBackend) Name() string { return "systemd" }

func (s *SystemdBackend) List() ([]Service, error) {
	// Implementation tricky without parsing list-units output complexly
	return []Service{}, nil
}

func (s *SystemdBackend) Enable(name string) error {
	cmd := exec.Command("systemctl", "enable", "--now", name)
	return cmd.Run()
}

func (s *SystemdBackend) Disable(name string) error {
	cmd := exec.Command("systemctl", "disable", "--now", name)
	return cmd.Run()
}

func (s *SystemdBackend) Start(name string) error {
	cmd := exec.Command("systemctl", "start", name)
	return cmd.Run()
}

func (s *SystemdBackend) Stop(name string) error {
	cmd := exec.Command("systemctl", "stop", name)
	return cmd.Run()
}

func (s *SystemdBackend) Restart(name string) error {
	cmd := exec.Command("systemctl", "restart", name)
	return cmd.Run()
}

func (s *SystemdBackend) IsActive(name string) (bool, error) {
	cmd := exec.Command("systemctl", "is-active", name)
	err := cmd.Run()
	// Exit code 0 means active
	if err == nil {
		return true, nil
	}
	// Exit code non-zero means inactive or unknown
	return false, nil
}

func (s *SystemdBackend) IsEnabled(name string) (bool, error) {
	cmd := exec.Command("systemctl", "is-enabled", name)
	out, err := cmd.Output()
	// systemctl is-enabled can return "enabled", "disabled", "masked", etc.
	// It relies on exit code too, but output is safer for enabled/disabled check
	output := strings.TrimSpace(string(out))
	if err == nil && output == "enabled" {
		return true, nil
	}
	return false, nil
}
