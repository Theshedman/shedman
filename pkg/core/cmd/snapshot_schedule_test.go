package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/snapshot"
)

type MockScheduler struct {
	EnableFunc  func() error
	DisableFunc func() error
	StatusFunc  func() (snapshot.ScheduleStatus, error)
	RunNowFunc  func() error
}

func (m *MockScheduler) Enable() error {
	if m.EnableFunc != nil {
		return m.EnableFunc()
	}
	return nil
}
func (m *MockScheduler) Disable() error {
	if m.DisableFunc != nil {
		return m.DisableFunc()
	}
	return nil
}
func (m *MockScheduler) Status() (snapshot.ScheduleStatus, error) {
	if m.StatusFunc != nil {
		return m.StatusFunc()
	}
	return snapshot.ScheduleStatus{}, nil
}
func (m *MockScheduler) RunNow() error {
	if m.RunNowFunc != nil {
		return m.RunNowFunc()
	}
	return nil
}

func TestSnapshotScheduleCmd(t *testing.T) {
	// Setup
	engine := core.NewEngine()
	mockSch := &MockScheduler{
		EnableFunc: func() error {
			return nil
		},
	}
	// Need to set scheduler on engine.
	// We haven't added SetScheduler to Engine yet!
	// Will address this in implementation.
	engine.SetScheduler(mockSch)

	buf := new(bytes.Buffer)

	// Test Enable
	if err := RunSnapshotScheduleEnable(engine, buf); err != nil {
		t.Fatalf("Enable execution failed: %v", err)
	}
	if !strings.Contains(buf.String(), "enabled") {
		t.Errorf("Expected enabled message, got: %s", buf.String())
	}
}
