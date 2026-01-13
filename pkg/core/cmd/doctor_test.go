package cmd

import (
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
		RunDoctor(mockEng, perfectChecks, mocks, false)
	})

	t.Run("Failing Health No Fix", func(t *testing.T) {
		RunDoctor(mockEng, failingChecks, mocks, false)
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
		RunDoctor(mockEng, failingChecks, fixMocks, true)

		if !lockRemoved {
			t.Error("Expected lock removal")
		}
		if !servicesReset {
			t.Error("Expected services reset")
		}
	})

	t.Run("Nil Engine", func(t *testing.T) {
		RunDoctor(nil, perfectChecks, mocks, false)
	})
}
