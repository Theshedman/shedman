package system

import (
	"fmt"
	"os/exec"
)

// ServiceManager handles system service operations
type ServiceManager interface {
	Enable(service string) error
	Disable(service string) error
	Start(service string) error
	Stop(service string) error
	Restart(service string) error
	IsActive(service string) (bool, error)
	IsEnabled(service string) (bool, error)
}

// SystemdManager implements ServiceManager for systemd
type SystemdManager struct{}

func NewSystemdManager() *SystemdManager {
	return &SystemdManager{}
}

func (s *SystemdManager) Enable(service string) error {
	return s.run("enable", service)
}

func (s *SystemdManager) Disable(service string) error {
	return s.run("disable", service)
}

func (s *SystemdManager) Start(service string) error {
	return s.run("start", service)
}

func (s *SystemdManager) Stop(service string) error {
	return s.run("stop", service)
}

func (s *SystemdManager) Restart(service string) error {
	return s.run("restart", service)
}

func (s *SystemdManager) IsActive(service string) (bool, error) {
	cmd := exec.Command("systemctl", "is-active", "--quiet", service)
	err := cmd.Run()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *SystemdManager) IsEnabled(service string) (bool, error) {
	cmd := exec.Command("systemctl", "is-enabled", "--quiet", service)
	err := cmd.Run()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *SystemdManager) run(action, service string) error {
	cmd := exec.Command("systemctl", action, service)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to %s %s: %s: %w", action, service, string(out), err)
	}
	return nil
}
