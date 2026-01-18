package restic

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

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

// Init initializes a new repository
func (m *Manager) Init(remote string) error {
	args := []string{"-r", "rclone:" + remote, "init"}

	// Init doesn't strictly need root if called by user, but if called by root, we should de-escalate rclone.
	cmd, cleanup, err := m.prepareCommand(context.Background(), "restic", args, false)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("restic init failed: %s: %w", string(out), err)
	}
	return nil
}

// Backup creates a backup
func (m *Manager) Backup(remote, path string, tags []string) error {
	args := []string{"-r", "rclone:" + remote, "backup", path}
	for _, tag := range tags {
		args = append(args, "--tag", tag)
	}

	// Backup reads source files, so it likely needs root if backing up /.snapshots
	cmd, cleanup, err := m.prepareCommand(context.Background(), "restic", args, true)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("restic backup failed: %s: %w", string(out), err)
	}
	return nil
}

// Restore restores a snapshot
func (m *Manager) Restore(remote, snapshotID, target string) error {
	args := []string{"-r", "rclone:" + remote, "restore", snapshotID, "--target", target}

	// Restore writes files, likely needs root for /.snapshots
	cmd, cleanup, err := m.prepareCommand(context.Background(), "restic", args, true)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("restic restore failed: %s: %w", string(out), err)
	}
	return nil
}

// List lists snapshots
func (m *Manager) List(remote string) ([]Snapshot, error) {
	args := []string{"-r", "rclone:" + remote, "snapshots", "--json"}

	// List doesn't need root
	cmd, cleanup, err := m.prepareCommand(context.Background(), "restic", args, false)
	if err != nil {
		return nil, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("restic list failed: %w", err)
	}

	var snaps []Snapshot
	if err := json.Unmarshal(out, &snaps); err != nil {
		return nil, fmt.Errorf("failed to parse restic output: %w", err)
	}
	return snaps, nil
}

// FindByTags finds snapshots matching all provided tags
func (m *Manager) FindByTags(remote string, tags []string) ([]Snapshot, error) {
	all, err := m.List(remote)
	if err != nil {
		return nil, err
	}

	var matches []Snapshot
	for _, s := range all {
		match := true
		// Filter by all required tags
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

// Helper to determine current user
func getCurrentUser() string {
	if u := os.Getenv("SUDO_USER"); u != "" {
		return u
	}
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	// Fallback
	return "root"
}

// prepareCommand handles privilege escalation and rclone isolation
func (m *Manager) prepareCommand(ctx context.Context, name string, args []string, requireRoot bool) (*exec.Cmd, func(), error) {
	var cleanup func()

	useSudo := requireRoot && os.Geteuid() != 0

	user := getCurrentUser()
	if (useSudo || os.Geteuid() == 0) && user != "root" && user != "" {
		tmpScript, err := os.CreateTemp("", "shedman-rclone-wrapper-*.sh")
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create rclone wrapper: %w", err)
		}

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
		// Use -E to preserve RESTIC_PASSWORD env var
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
