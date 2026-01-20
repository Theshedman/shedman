package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiagnose(t *testing.T) {
	// Mock Engine
	eng := &Engine{}

	checks := DoctorChecks{
		CheckConnection: func() bool { return true },
		CheckDiskSpace:  func(p string) float64 { return 50.0 },
		CheckServices:   func() []string { return nil },
		CheckLockFile:   func() bool { return false },
	}

	report := Diagnose(eng, checks)

	assert.NotEmpty(t, report.Items, "Report should have items")
	assert.True(t, report.IsHealthy(), "System should be healthy")
}

func TestDiagnose_Failing(t *testing.T) {
	checks := DoctorChecks{
		CheckConnection: func() bool { return false }, // Failed
		CheckDiskSpace:  func(p string) float64 { return 50.0 },
		CheckServices:   func() []string { return []string{"failed.service"} }, // Failed
		CheckLockFile:   func() bool { return true },                           // Failed
	}

	report := Diagnose(nil, checks)

	assert.False(t, report.IsHealthy(), "System should be unhealthy")
	// Verify failed items
	assert.Equal(t, DiagnoseStatusFail, report.Items[0].Status) // Connection
}
