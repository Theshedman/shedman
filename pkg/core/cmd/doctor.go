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

	// Define core check adapter
	coreChecks := core.DoctorChecks{
		CheckConnection: checks.CheckConnection,
		CheckDiskSpace:  checks.CheckDiskSpace,
		CheckServices:   checks.CheckServices,
		CheckLockFile:   checks.CheckLockFile,
	}

	report := core.Diagnose(eng, coreChecks)
	hasIssues := !report.IsHealthy()

	// Convert and Print Report
	for _, item := range report.Items {
		switch item.Status {
		case core.DiagnoseStatusFail:
			_, _ = fmt.Fprintf(w, "FAILED (%s)\n", item.Name)
			if item.Name == "Pacman Lock" && fix {
				if err := repairs.RemoveLock(); err != nil {
					_, _ = fmt.Fprintf(w, "  Failed to remove lock: %v\n", err)
				} else {
					_, _ = fmt.Fprintln(w, "  Lock file removed.")
				}
			}
		case core.DiagnoseStatusWarn:
			_, _ = fmt.Fprintf(w, "WARNING (%s): %s\n", item.Name, item.Message)
			if item.Name == "System Services" && fix {
				if err := repairs.ResetFailedServices(); err != nil {
					_, _ = fmt.Fprintf(w, "  Failed to reset services: %v\n", err)
				} else {
					_, _ = fmt.Fprintln(w, "  Services reset.")
				}
			}
		case core.DiagnoseStatusOk:
			_, _ = fmt.Fprintf(w, "OK (%s)\n", item.Name)
		}
	}

	_, _ = fmt.Fprintln(w)

	if hasIssues {
		if fix {
			_, _ = fmt.Fprintln(w, "Doctor attempted to fix issues. Please re-run to verify.")
		} else {
			_, _ = fmt.Fprintln(w, "Doctor found issues. Run with --fix to attempt repairs.")
		}
	} else {
		_, _ = fmt.Fprintln(w, "Your system looks verify healthy!")
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
