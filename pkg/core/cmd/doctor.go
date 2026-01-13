package cmd

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/output"
	"github.com/theshedman/shedman/pkg/core"
)

// DoctorChecks holds check functions for testing
type DoctorChecks struct {
	CheckConnection func() bool
	CheckDiskSpace  func(path string) float64
	CheckServices   func() []string
	CheckLockFile   func() bool
}

// Real checks
var defaultChecks = DoctorChecks{
	CheckConnection: checkConnection,
	CheckDiskSpace:  checkDiskSpace,
	CheckServices:   checkFailedServices,
	CheckLockFile: func() bool {
		_, err := os.Stat("/var/lib/pacman/db.lck")
		return err == nil
	},
}

// DoctorRepairs holds functions to fix issues
type DoctorRepairs struct {
	RemoveLock          func() error
	ResetFailedServices func() error
}

var doctorFix bool

// Real repair actions
var defaultRepairs = DoctorRepairs{
	RemoveLock: func() error {
		return exec.Command("sudo", "rm", "-f", "/var/lib/pacman/db.lck").Run()
	},
	ResetFailedServices: func() error {
		return exec.Command("sudo", "systemctl", "reset-failed").Run()
	},
}

// DoctorCmd represents the doctor command
var DoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system health and shedman status",
	Run: func(cmd *cobra.Command, args []string) {
		eng, err := NewEngineWithConfig(nil)
		if err != nil {
			output.Error("FAILED (%v)", err)
		}

		// Override default repairs with engine based repairs where possible
		repairs := defaultRepairs
		if eng != nil && eng.GetOfficialBackend() != nil {
			repairs.RemoveLock = func() error {
				return eng.RepairLock()
			}
		}

		RunDoctor(eng, defaultChecks, repairs, doctorFix)
	},
}

func init() {
	DoctorCmd.Flags().BoolVar(&doctorFix, "fix", false, "Attempt to automatically fix issues (e.g. remove lock file)")
}

// RunDoctor executes health checks
func RunDoctor(eng *core.Engine, checks DoctorChecks, repairs DoctorRepairs, fix bool) {
	output.Info("Running system health checks...\n")
	hasIssues := false

	// 1. Check Engine/Backend
	fmt.Print("Checking Package Backend... ")
	if eng == nil {
		output.Error("FAILED (Engine Init)")
		hasIssues = true
	} else {
		backend := eng.GetOfficialBackend()
		if backend != nil {
			output.Success("OK (%s)", backend.Name())
		} else {
			output.Warning("WARNING (No official backend detected)")
			hasIssues = true
		}
	}

	// 2. Check Lock File
	fmt.Print("Checking Pacman Lock... ")
	if checks.CheckLockFile() {
		output.Error("FAILED (Lock file exists: /var/lib/pacman/db.lck)")
		hasIssues = true
		if fix {
			output.Info("  Attempting to remove lock file...")
			if err := repairs.RemoveLock(); err != nil {
				output.Error("  Failed to remove lock: %v", err)
			} else {
				output.Success("  Lock file removed.")
			}
		}
	} else {
		output.Success("OK")
	}

	// 3. Check Internet Connectivity
	fmt.Print("Checking Connectivity... ")
	if checks.CheckConnection() {
		output.Success("OK")
	} else {
		output.Warning("WARNING (Cannot reach archlinux.org)")
		hasIssues = true
	}

	// 4. Check Disk Space (Root)
	fmt.Print("Checking Disk Space... ")
	freeGB := checks.CheckDiskSpace("/")
	if freeGB < 1.0 {
		output.Error("FAILED (Only %.1fGB free on /)", freeGB)
		hasIssues = true
	} else {
		output.Success("OK (%.1fGB free)", freeGB)
	}

	// 5. Check Failed Services
	fmt.Print("Checking Failed Services... ")
	failed := checks.CheckServices()
	if len(failed) > 0 {
		output.Warning("WARNING (%d failed units)", len(failed))
		for _, unit := range failed {
			fmt.Printf("  - %s\n", unit)
		}
		hasIssues = true
		if fix {
			output.Info("  Attempting to reset failed services...")
			if err := repairs.ResetFailedServices(); err != nil {
				output.Error("  Failed to reset services: %v", err)
			} else {
				output.Success("  Services reset.")
			}
		}
	} else {
		output.Success("OK")
	}

	fmt.Println()
	if hasIssues {
		if fix {
			output.Warning("Doctor attempted to fix issues. Please re-run to verify.")
		} else {
			output.Warning("Doctor found issues. Run with --fix to attempt repairs.")
		}
	} else {
		output.Success("Your system looks verify healthy!")
	}
}

func checkConnection() bool {
	timeout := 2 * time.Second
	_, err := net.DialTimeout("tcp", "archlinux.org:443", timeout)
	return err == nil
}

func checkDiskSpace(path string) float64 {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0
	}
	// Available blocks * size / 1024^3
	return float64(stat.Bavail*uint64(stat.Bsize)) / (1024 * 1024 * 1024)
}

func checkFailedServices() []string {
	out, err := exec.Command("systemctl", "list-units", "--state=failed", "--no-legend", "--plain").Output()
	if err != nil {
		return nil
	}
	var services []string
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) > 0 {
				services = append(services, parts[0])
			}
		}
	}
	return services
}
