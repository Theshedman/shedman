package cmd

import (
	"bytes"
	"testing"
)

type mockServiceManager struct {
	enabled []string
}

func (m *mockServiceManager) Enable(name string) error {
	m.enabled = append(m.enabled, name)
	return nil
}

func TestDetectSystemdUnits(t *testing.T) {
	files := []string{
		"/usr/lib/systemd/system/docker.service",
		"/usr/lib/systemd/system/docker.socket",
		"/usr/lib/systemd/system/ignore.timer",
		"/usr/lib/systemd/user/user.service",
	}

	units := detectSystemdUnits(files)
	if len(units) != 2 {
		t.Fatalf("expected 2 units, got %d", len(units))
	}
	if units[0] != "docker.service" || units[1] != "docker.socket" {
		t.Errorf("unexpected units: %v", units)
	}
}

func TestPromptEnableServices(t *testing.T) {
	manager := &mockServiceManager{}
	origConfirm := confirmServicePrompt
	confirmServicePrompt = func(_ string, _ ConfirmOptions) bool { return true }
	t.Cleanup(func() { confirmServicePrompt = origConfirm })

	var buf bytes.Buffer
	services := []string{"docker.service", "docker.socket"}

	if err := promptEnableServices(&buf, manager, services); err != nil {
		t.Fatalf("promptEnableServices failed: %v", err)
	}

	if len(manager.enabled) != 2 {
		t.Errorf("expected 2 services enabled, got %d", len(manager.enabled))
	}
}
