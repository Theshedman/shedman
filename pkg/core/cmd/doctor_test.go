package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

func TestRunDoctor(t *testing.T) {
	mockEng := core.NewEngineWithBackend(&MockBackend{})

	perfectChecks := DoctorChecks{
		CheckConnection: func() bool { return true },
		CheckDiskSpace:  func(path string) float64 { return 100.0 },
		CheckServices:   func() []string { return nil },
		CheckLockFile:   func() bool { return false },
	}

	failingChecks := DoctorChecks{
		CheckConnection: func() bool { return false },
		CheckDiskSpace:  func(path string) float64 { return 0.1 },
		CheckServices:   func() []string { return []string{"docker.service"} },
		CheckLockFile:   func() bool { return true },
	}

	mocks := DoctorRepairs{
		RemoveLock:          func() error { return nil },
		ResetFailedServices: func() error { return nil },
	}

	t.Run("Perfect Health", func(t *testing.T) {
		var buf bytes.Buffer
		RunDoctor(mockEng, &buf, perfectChecks, mocks, false)
		if !strings.Contains(buf.String(), "OK") {
			t.Error("Expected OK output")
		}
	})

	t.Run("Failing Health No Fix", func(t *testing.T) {
		var buf bytes.Buffer
		RunDoctor(mockEng, &buf, failingChecks, mocks, false)
		if !strings.Contains(buf.String(), "FAILED") {
			t.Error("Expected FAILED output")
		}
	})

	t.Run("Failing Health With Fix", func(t *testing.T) {
		lockRemoved := false
		servicesReset := false

		fixMocks := DoctorRepairs{
			RemoveLock: func() error {
				lockRemoved = true
				return nil
			},
			ResetFailedServices: func() error {
				servicesReset = true
				return nil
			},
		}

		// We expect lock removal and service reset
		var buf bytes.Buffer
		RunDoctor(mockEng, &buf, failingChecks, fixMocks, true)

		if !lockRemoved {
			t.Error("Expected lock removal")
		}
		if !servicesReset {
			t.Error("Expected services reset")
		}
		if !strings.Contains(buf.String(), "Lock file removed") {
			t.Error("Expected fix output")
		}
	})

	t.Run("Nil Engine", func(t *testing.T) {
		var buf bytes.Buffer
		RunDoctor(nil, &buf, perfectChecks, mocks, false)
		if !strings.Contains(buf.String(), "FAILED (Engine Init)") {
			t.Error("Expected Engine Init failure")
		}
	})

	t.Run("OrphansAndConflictsWarning", func(t *testing.T) {
		checks := DoctorChecks{
			CheckConnection: func() bool { return true },
			CheckDiskSpace:  func(path string) float64 { return 100.0 },
			CheckServices:   func() []string { return nil },
			CheckLockFile:   func() bool { return false },
			CheckOrphans:    func() ([]string, error) { return []string{"pkg1"}, nil },
			CheckDatabase:   func() error { return nil },
			CheckConflicts:  func() ([]core.FileConflict, error) { return []core.FileConflict{{FilePath: "/etc/test"}}, nil },
		}

		var buf bytes.Buffer
		RunDoctor(mockEng, &buf, checks, mocks, false)

		output := buf.String()
		if !strings.Contains(output, "WARNING (Orphan Packages)") {
			t.Error("Expected orphan warning output")
		}
		if !strings.Contains(output, "WARNING (File Conflicts)") {
			t.Error("Expected file conflict warning output")
		}
	})
}
