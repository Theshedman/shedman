package cmd

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
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
			fmt.Printf("FAILED (%v)\n", err)
		}

		// Override default repairs with engine based repairs where possible
		repairs := defaultRepairs
		if eng != nil && eng.GetOfficialBackend() != nil {
			repairs.RemoveLock = func() error {
				return eng.RepairLock()
			}
		}

		RunDoctor(eng, cmd.OutOrStdout(), defaultChecks, repairs, doctorFix)
	},
}

func init() {
	DoctorCmd.Flags().BoolVar(&doctorFix, "fix", false, "Attempt to automatically fix issues (e.g. remove lock file)")
}

// RunDoctor executes health checks
func RunDoctor(eng *core.Engine, w io.Writer, checks DoctorChecks, repairs DoctorRepairs, fix bool) {
	_, _ = fmt.Fprintln(w, "Running system health checks...")

	_, _ = fmt.Fprintln(w)

	hasIssues := false

	// Helper to print styled status
	printStatus := func(msg string, isError bool, isWarning bool) {
		if isError {
			fmt.Fprintf(w, "FAILED (%s)\n", msg)
		} else if isWarning {
			fmt.Fprintf(w, "WARNING (%s)\n", msg)
		} else {
			fmt.Fprintf(w, "OK (%s)\n", msg)
		}
	}
	// Simple OK
	printOK := func() {
		fmt.Fprintln(w, "OK")
	}

	// 1. Check Engine/Backend
	_, _ = fmt.Fprint(w, "Checking Package Backend... ")

	if eng == nil {
		printStatus("Engine Init", true, false)
		hasIssues = true
	} else {
		backend := eng.GetOfficialBackend()
		if backend != nil {
			printStatus(backend.Name(), false, false)
		} else {
			printStatus("No official backend detected", false, true)
			hasIssues = true
		}
	}

	// 2. Check Lock File
	_, _ = fmt.Fprint(w, "Checking Pacman Lock... ")

	if checks.CheckLockFile() {
		printStatus("Lock file exists: /var/lib/pacman/db.lck", true, false)
		hasIssues = true
		if fix {
			fmt.Fprintln(w, "  Attempting to remove lock file...")
			if err := repairs.RemoveLock(); err != nil {
				fmt.Fprintf(w, "  Failed to remove lock: %v\n", err)
			} else {
				fmt.Fprintln(w, "  Lock file removed.")
			}
		}
	} else {
		printOK()
	}

	// 3. Check Internet Connectivity
	_, _ = fmt.Fprint(w, "Checking Connectivity... ")

	if checks.CheckConnection() {
		printOK()
	} else {
		printStatus("Cannot reach archlinux.org", false, true)
		hasIssues = true
	}

	// 4. Check Disk Space (Root)
	_, _ = fmt.Fprint(w, "Checking Disk Space... ")

	freeGB := checks.CheckDiskSpace("/")
	if freeGB < 1.0 {
		printStatus(fmt.Sprintf("Only %.1fGB free on /", freeGB), true, false)
		hasIssues = true
	} else {
		printStatus(fmt.Sprintf("%.1fGB free", freeGB), false, false)
	}

	// 5. Check Failed Services
	_, _ = fmt.Fprint(w, "Checking Failed Services... ")

	failed := checks.CheckServices()
	if len(failed) > 0 {
		printStatus(fmt.Sprintf("%d failed units", len(failed)), false, true)
		for _, unit := range failed {
			fmt.Fprintf(w, "  - %s\n", unit)
		}
		hasIssues = true
		if fix {
			fmt.Fprintln(w, "  Attempting to reset failed services...")
			if err := repairs.ResetFailedServices(); err != nil {
				fmt.Fprintf(w, "  Failed to reset services: %v\n", err)
			} else {
				fmt.Fprintln(w, "  Services reset.")
			}
		}
	} else {
		printOK()
	}

	fmt.Fprintln(w)
	if hasIssues {
		if fix {
			fmt.Fprintln(w, "Doctor attempted to fix issues. Please re-run to verify.")
		} else {
			fmt.Fprintln(w, "Doctor found issues. Run with --fix to attempt repairs.")
		}
	} else {
		fmt.Fprintln(w, "Your system looks verify healthy!")
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
