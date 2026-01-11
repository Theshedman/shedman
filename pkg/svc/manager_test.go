package svc

import (
	"errors"
	"testing"
)

// MockBackend for Service testing
type MockBackend struct {
	services map[string]*Service
}

func (m *MockBackend) Name() string { return "mock" }

func (m *MockBackend) List() ([]Service, error) {
	var list []Service
	for _, s := range m.services {
		list = append(list, *s)
	}
	return list, nil
}

func (m *MockBackend) Enable(name string) error {
	if s, ok := m.services[name]; ok {
		s.Enabled = true
		return nil
	}
	return errors.New("service not found")
}

func (m *MockBackend) Disable(name string) error {
	if s, ok := m.services[name]; ok {
		s.Enabled = false
		return nil
	}
	return errors.New("service not found")
}

func (m *MockBackend) Start(name string) error {
	if s, ok := m.services[name]; ok {
		s.Active = true
		return nil
	}
	return errors.New("service not found")
}

func (m *MockBackend) Stop(name string) error {
	if s, ok := m.services[name]; ok {
		s.Active = false
		return nil
	}
	return errors.New("service not found")
}

func (m *MockBackend) Restart(name string) error {
	return m.Start(name)
}

func (m *MockBackend) IsActive(name string) (bool, error) {
	if s, ok := m.services[name]; ok {
		return s.Active, nil
	}
	return false, errors.New("service not found")
}

func (m *MockBackend) IsEnabled(name string) (bool, error) {
	if s, ok := m.services[name]; ok {
		return s.Enabled, nil
	}
	return false, errors.New("service not found")
}

func TestManager_SwitchState(t *testing.T) {
	mock := &MockBackend{
		services: map[string]*Service{
			"nginx": {Name: "nginx", Active: false, Enabled: false},
		},
	}
	mgr := NewWithBackend(mock)

	// Test Enable
	if err := mgr.Enable("nginx"); err != nil {
		t.Fatalf("Enable failed: %v", err)
	}
	if !mock.services["nginx"].Enabled {
		t.Error("nginx should be enabled")
	}

	// Test Start
	if err := mgr.Start("nginx"); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if !mock.services["nginx"].Active {
		t.Error("nginx should be active")
	}
}

func TestManager_Status(t *testing.T) {
	mock := &MockBackend{
		services: map[string]*Service{
			"dbus": {Name: "dbus", Active: true, Enabled: true},
		},
	}
	mgr := NewWithBackend(mock)

	status, err := mgr.Status("dbus")
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}

	if !status.Active {
		t.Error("dbus should be active")
	}
	if !status.Enabled {
		t.Error("dbus should be enabled")
	}
}
