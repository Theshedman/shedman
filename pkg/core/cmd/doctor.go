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
	CheckOrphans    func() ([]string, error)
	CheckDatabase   func() error
	CheckConflicts  func() ([]core.FileConflict, error)
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

	// Additional checks: orphans, database integrity, file conflicts
	extraFailed := false
	runCheck := func(name string, status core.DiagnoseStatus, msg string) {
		switch status {
		case core.DiagnoseStatusFail:
			extraFailed = true
			_, _ = fmt.Fprintf(w, "FAILED (%s): %s\n", name, msg)
		case core.DiagnoseStatusWarn:
			_, _ = fmt.Fprintf(w, "WARNING (%s): %s\n", name, msg)
		default:
			_, _ = fmt.Fprintf(w, "OK (%s)\n", name)
		}
	}

	// Orphans
	if eng != nil {
		orphans, err := checkOrphans(eng, checks)
		if err != nil {
			runCheck("Orphan Packages", core.DiagnoseStatusFail, err.Error())
		} else if len(orphans) > 0 {
			runCheck("Orphan Packages", core.DiagnoseStatusWarn, fmt.Sprintf("%d packages", len(orphans)))
		} else {
			runCheck("Orphan Packages", core.DiagnoseStatusOk, "")
		}
	}

	// Database integrity / missing deps
	if eng != nil {
		if err := checkDatabase(eng, checks); err != nil {
			runCheck("Database Integrity", core.DiagnoseStatusFail, err.Error())
		} else {
			runCheck("Database Integrity", core.DiagnoseStatusOk, "")
		}
	}

	// File conflicts
	if eng != nil {
		conflicts, err := checkConflicts(eng, checks)
		if err != nil {
			runCheck("File Conflicts", core.DiagnoseStatusFail, err.Error())
		} else if len(conflicts) > 0 {
			runCheck("File Conflicts", core.DiagnoseStatusWarn, fmt.Sprintf("%d conflicts", len(conflicts)))
		} else {
			runCheck("File Conflicts", core.DiagnoseStatusOk, "")
		}
	}

	_, _ = fmt.Fprintln(w)

	if extraFailed {
		hasIssues = true
	}

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

func checkOrphans(eng *core.Engine, checks DoctorChecks) ([]string, error) {
	if checks.CheckOrphans != nil {
		return checks.CheckOrphans()
	}
	return eng.ListOrphans()
}

func checkDatabase(eng *core.Engine, checks DoctorChecks) error {
	if checks.CheckDatabase != nil {
		return checks.CheckDatabase()
	}
	return eng.CheckDatabase()
}

func checkConflicts(eng *core.Engine, checks DoctorChecks) ([]core.FileConflict, error) {
	if checks.CheckConflicts != nil {
		return checks.CheckConflicts()
	}

	backend := eng.GetOfficialBackend()
	if backend == nil {
		return nil, nil
	}

	packageFiles, err := core.GetInstalledPackageFilesWithBackend(backend)
	if err != nil {
		return nil, err
	}

	checker := core.NewFileConflictChecker()
	for pkg, files := range packageFiles {
		checker.RegisterPackageFiles(pkg, files)
	}

	return checker.CheckConflicts(), nil
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
