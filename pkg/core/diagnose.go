package core

import (
	"fmt"
)

// Diagnose runs all health checks and returns a structured report
func Diagnose(eng *Engine, checks DoctorChecks) SystemReport {
	report := SystemReport{}

	// Helper to add item
	addItem := func(name string, success bool, failMsg string, warn bool) {
		status := DiagnoseStatusOk
		msg := "OK"
		if !success {
			if warn {
				status = DiagnoseStatusWarn
				msg = fmt.Sprintf("WARNING (%s)", failMsg)
			} else {
				status = DiagnoseStatusFail
				msg = fmt.Sprintf("FAILED (%s)", failMsg)
			}
		}
		item := DiagnoseItem{
			Name:    name,
			Status:  status,
			Message: msg,
		}
		report.Items = append(report.Items, item)
	}

	// 1. Check Engine
	if eng == nil {
		addItem("Engine Init", false, "Engine is nil", false)
	} else {
		backend := eng.GetOfficialBackend()
		if backend != nil {
			addItem(fmt.Sprintf("Backend: %s", backend.Name()), true, "", false)
		} else {
			addItem("Backend Detection", false, "No official backend detected", true)
		}
	}

	// 2. Lock File
	isLocked := checks.CheckLockFile()
	addItem("Pacman Lock", !isLocked, "Lock file exists", false)

	// 3. Connectivity
	isConnected := checks.CheckConnection()
	addItem("Connectivity", isConnected, "Cannot reach archlinux.org", true)

	// 4. Disk Space
	freeGB := checks.CheckDiskSpace("/")
	addItem("Disk Space", freeGB >= 1.0, fmt.Sprintf("Only %.1fGB free", freeGB), false)

	// 5. Failed Services
	failed := checks.CheckServices()
	if len(failed) > 0 {
		addItem("System Services", false, fmt.Sprintf("%d failed units", len(failed)), true)
		// Add individual services as info items if needed, or just summary
	} else {
		addItem("System Services", true, "", false)
	}

	return report
}
