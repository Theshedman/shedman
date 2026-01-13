package cmd

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

func TestRunGroupList(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	mock.ListGroupsFunc = func() ([]string, error) {
		return []string{"base", "base-devel", "gnome"}, nil
	}

	var buf bytes.Buffer
	if err := RunGroupList(eng, &buf); err != nil {
		t.Errorf("RunGroupList() error = %v", err)
	}

	output := buf.String()
	expectedGroups := []string{"base", "base-devel", "gnome"}
	for _, g := range expectedGroups {
		if !strings.Contains(output, g) {
			t.Errorf("Output missing group %q. Got:\n%s", g, output)
		}
	}
}

func TestRunGroupInfo(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	mock.GetGroupPackagesFunc = func(group string) ([]string, error) {
		if group == "gnome" {
			return []string{"gnome-shell", "nautilus"}, nil
		}
		return nil, fmt.Errorf("group not found")
	}

	// Success case
	var buf bytes.Buffer
	if err := RunGroupInfo(eng, &buf, "gnome"); err != nil {
		t.Errorf("RunGroupInfo(gnome) error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Group: gnome") {
		t.Errorf("Output missing group name. Got:\n%s", output)
	}
	if !strings.Contains(output, "gnome-shell") || !strings.Contains(output, "nautilus") {
		t.Errorf("Output missing packages. Got:\n%s", output)
	}

	// Fail case
	if err := RunGroupInfo(eng, &buf, "kde"); err == nil {
		t.Error("Expected error for non-existent group")
	}
}

func TestRunGroupInstall(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	mock.GetGroupPackagesFunc = func(group string) ([]string, error) {
		if group == "gnome" {
			return []string{"gnome-shell", "nautilus"}, nil
		}
		return nil, fmt.Errorf("group not found")
	}

	installCalled := false
	mock.InstallFunc = func(pkgs []string, opts core.InstallOptions) error {
		installCalled = true
		if len(pkgs) != 2 {
			t.Errorf("Expected 2 packages to install, got %d", len(pkgs))
		}
		return nil
	}

	var buf bytes.Buffer
	if err := RunGroupInstall(eng, &buf, "gnome"); err != nil {
		t.Errorf("RunGroupInstall error = %v", err)
	}

	if !installCalled {
		t.Error("Install was not called")
	}

	output := buf.String()
	if !strings.Contains(output, "Installing 2 packages") {
		t.Errorf("Output missing install confirmation. Got:\n%s", output)
	}
}

func TestRunGroupRemove(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	mock.GetGroupPackagesFunc = func(group string) ([]string, error) {
		if group == "gnome" {
			return []string{"gnome-shell", "nautilus"}, nil
		}
		return nil, fmt.Errorf("group not found")
	}

	removeCalled := false
	mock.RemoveFunc = func(pkgs []string, opts core.RemoveOptions) error {
		removeCalled = true
		if len(pkgs) != 2 {
			t.Errorf("Expected 2 packages to remove, got %d", len(pkgs))
		}
		return nil
	}

	var buf bytes.Buffer
	if err := RunGroupRemove(eng, &buf, "gnome"); err != nil {
		t.Errorf("RunGroupRemove error = %v", err)
	}

	if !removeCalled {
		t.Error("Remove was not called")
	}

	output := buf.String()
	if !strings.Contains(output, "Removing 2 packages") {
		t.Errorf("Output missing remove confirmation. Got:\n%s", output)
	}
}
