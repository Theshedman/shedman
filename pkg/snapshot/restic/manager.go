package restic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/theshedman/shedman/internal/util"
	"github.com/theshedman/shedman/pkg/executor"
)

// Manager wraps restic commands
type Manager struct {
	exec     executor.Executor
	password string
}

// NewManager creates a new restic manager
func NewManager(exec executor.Executor, password string) *Manager {
	return &Manager{
		exec:     exec,
		password: password,
	}
}

// Snapshot represents a restic snapshot
type Snapshot struct {
	ID       string    `json:"id"`
	Time     time.Time `json:"time"`
	Tree     string    `json:"tree"`
	Paths    []string  `json:"paths"`
	Hostname string    `json:"hostname"`
	Username string    `json:"username"`
	Tags     []string  `json:"tags"`
	ShortID  string    `json:"short_id"`
}

// resolveRepo determines the correct prefix for the backend
func (m *Manager) resolveRepo(remote string) string {
	if strings.HasPrefix(remote, "/") || strings.HasPrefix(remote, ".") {
		return remote
	}

	// Supported restic backend schemes
	schemes := []string{
		"s3:", "sftp:", "rest:", "b2:", "azure:", "gs:", "swift:", "rclone:",
	}

	for _, scheme := range schemes {
		if strings.HasPrefix(remote, scheme) {
			return remote
		}
	}

	// If no scheme is present, assume it is an rclone remote alias
	return "rclone:" + remote
}

// Init initializes a new repository
func (m *Manager) Init(ctx context.Context, remote string, w io.Writer) error {
	repo := m.resolveRepo(remote)
	args := []string{"-r", repo, "init"}

	cmd, cleanup, err := m.prepareCommand(ctx, "restic", args, false)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	cmd.Stdout = w
	cmd.Stderr = w

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("restic init failed: %w", err)
	}
	return nil
}

// Backup creates a backup
func (m *Manager) Backup(ctx context.Context, remote, path string, tags []string, stdout, stderr io.Writer) error {
	repo := m.resolveRepo(remote)
	args := []string{"-r", repo, "backup", path}
	for _, tag := range tags {
		args = append(args, "--tag", tag)
	}

	cmd, cleanup, err := m.prepareCommand(ctx, "restic", args, true)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	if stdout != nil {
		cmd.Stdout = stdout
	}
	if stderr != nil {
		cmd.Stderr = stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("restic backup failed: %w", err)
	}
	return nil
}

// Restore restores a snapshot
func (m *Manager) Restore(ctx context.Context, remote, snapshotID, target string, stdout, stderr io.Writer) error {
	repo := m.resolveRepo(remote)
	args := []string{"-r", repo, "restore", snapshotID, "--target", target}

	cmd, cleanup, err := m.prepareCommand(ctx, "restic", args, true)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	if stdout != nil {
		cmd.Stdout = stdout
	}
	if stderr != nil {
		cmd.Stderr = stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("restic restore failed: %w", err)
	}
	return nil
}

// Check verifies the integrity of a repository.
func (m *Manager) Check(ctx context.Context, remote string, w io.Writer) error {
	repo := m.resolveRepo(remote)
	args := []string{"-r", repo, "check"}

	cmd, cleanup, err := m.prepareCommand(ctx, "restic", args, true)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	if w != nil {
		cmd.Stdout = w
		cmd.Stderr = w
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("restic check failed: %w", err)
	}
	return nil
}

// List lists snapshots
func (m *Manager) List(ctx context.Context, remote string) ([]Snapshot, error) {
	repo := m.resolveRepo(remote)
	args := []string{"-r", repo, "snapshots", "--json"}

	cmd, cleanup, err := m.prepareCommand(ctx, "restic", args, false)
	if err != nil {
		return nil, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	// Capture output for List as we need to parse it
	out, err := cmd.Output()
	if err != nil {
		// If exit code is 1, it might be an empty repo or error, checking output might be helpful
		// JSON output is expected on stdout
		return nil, fmt.Errorf("failed to parse snapshots JSON: %w", err)
	}

	var snaps []Snapshot
	if len(out) == 0 {
		return []Snapshot{}, nil
	}
	if err := json.Unmarshal(out, &snaps); err != nil {
		return nil, fmt.Errorf("failed to parse restic output: %w", err)
	}
	return snaps, nil
}

// FindByTags finds snapshots matching all provided tags
func (m *Manager) FindByTags(ctx context.Context, remote string, tags []string) ([]Snapshot, error) {
	all, err := m.List(ctx, remote)
	if err != nil {
		return nil, err
	}

	var matches []Snapshot
	for _, s := range all {
		match := true
		for _, requiredTag := range tags {
			found := false
			for _, t := range s.Tags {
				if t == requiredTag {
					found = true
					break
				}
			}
			if !found {
				match = false
				break
			}
		}
		if match {
			matches = append(matches, s)
		}
	}
	return matches, nil
}

// getCurrentUser returns the current user, preferring SUDO_USER if set.
// Returns an error if the username fails validation (security check against injection).
func getCurrentUser() (string, error) {
	if u := os.Getenv("SUDO_USER"); u != "" {
		if err := util.ValidateUsername(u); err != nil {
			return "", fmt.Errorf("invalid SUDO_USER: %w", err)
		}
		return u, nil
	}
	if u := os.Getenv("USER"); u != "" {
		if err := util.ValidateUsername(u); err != nil {
			return "", fmt.Errorf("invalid USER: %w", err)
		}
		return u, nil
	}
	return "root", nil
}

// prepareCommand handles privilege escalation and rclone isolation
func (m *Manager) prepareCommand(ctx context.Context, name string, args []string, requireRoot bool) (*exec.Cmd, func(), error) {
	var cleanup func()

	useSudo := requireRoot && os.Geteuid() != 0

	user, err := getCurrentUser()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get current user: %w", err)
	}

	if (useSudo || os.Geteuid() == 0) && user != "root" && user != "" {
		tmpScript, err := os.CreateTemp("", "shedman-rclone-wrapper-*.sh")
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create rclone wrapper: %w", err)
		}

		// Username is validated by getCurrentUser(), safe to use in script
		scriptContent := fmt.Sprintf("#!/bin/sh\nexec sudo -u %s rclone \"$@\"\n", user)
		if _, err := tmpScript.WriteString(scriptContent); err != nil {
			_ = tmpScript.Close()
			_ = os.Remove(tmpScript.Name())
			return nil, nil, fmt.Errorf("failed to write rclone wrapper: %w", err)
		}
		_ = tmpScript.Close()

		if err := os.Chmod(tmpScript.Name(), 0755); err != nil {
			_ = os.Remove(tmpScript.Name())
			return nil, nil, fmt.Errorf("failed to chmod rclone wrapper: %w", err)
		}

		args = append([]string{"-o", "rclone.program=" + tmpScript.Name()}, args...)

		cleanup = func() {
			_ = os.Remove(tmpScript.Name())
		}
	}

	var cmd *exec.Cmd
	if useSudo {
		fullArgs := append([]string{"-E", "restic"}, args...)
		if ctx != nil {
			cmd = m.exec.CommandContext(ctx, "sudo", fullArgs...)
		} else {
			cmd = m.exec.Command("sudo", fullArgs...)
		}
	} else {
		if ctx != nil {
			cmd = m.exec.CommandContext(ctx, "restic", args...)
		} else {
			cmd = m.exec.Command("restic", args...)
		}
	}

	m.setEnv(cmd)
	return cmd, cleanup, nil
}

func (m *Manager) setEnv(cmd *exec.Cmd) {
	cmd.Env = append(os.Environ(), "RESTIC_PASSWORD="+m.password)
}
