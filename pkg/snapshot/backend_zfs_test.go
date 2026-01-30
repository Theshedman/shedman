package snapshot

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theshedman/shedman/pkg/executor"
)

func TestZFSBackend_List_ParsesSnapshots(t *testing.T) {
	mock := &executor.MockExecutor{
		OutputFunc: func(name string, args ...string) ([]byte, error) {
			cmdName, cmdArgs := stripSudo(name, args)
			if cmdName == "zfs" && hasArgZFS(cmdArgs, "name,mountpoint") {
				return []byte("tank/ROOT/default\t/\n"), nil
			}
			if cmdName == "zfs" && hasArgZFS(cmdArgs, "name,creation,used,com.shedman:description,com.shedman:type,com.shedman:id") {
				return []byte(
					"tank/ROOT/default@20240101-000000\t1704067200\t1048576\tdesc\tpre\t20240101-000000\n" +
						"tank/ROOT/default/home@20240101-000000\t1704067200\t2048\t-\t-\t20240101-000000\n" +
						"tank/ROOT/default@20240102-000000\t1704153600\t1024\tdesc2\tpost\t20240102-000000\n",
				), nil
			}
			return nil, nil
		},
		CommandFunc: func(name string, args ...string) *exec.Cmd {
			return exec.Command("true")
		},
	}

	backend := NewZFSBackend(nil, mock)
	backend.SetRootDataset("tank/ROOT/default")

	snaps, err := backend.List(context.Background(), ListOptions{})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(snaps) != 2 {
		t.Fatalf("Expected 2 snapshots, got %d", len(snaps))
	}
	if snaps[0].ID != "20240102-000000" && snaps[1].ID != "20240102-000000" {
		t.Errorf("Expected snapshot 20240102-000000 to be present")
	}
	for _, snap := range snaps {
		if snap.ID == "20240101-000000" {
			if snap.Size != 1048576+2048 {
				t.Errorf("Unexpected aggregated size: %d", snap.Size)
			}
			if snap.Description != "desc" || snap.Type != "pre" {
				t.Errorf("Unexpected metadata: %+v", snap)
			}
		}
	}
}

func TestZFSBackend_Diff(t *testing.T) {
	var diffDatasets []string
	mock := &executor.MockExecutor{
		OutputFunc: func(name string, args ...string) ([]byte, error) {
			cmdName, cmdArgs := stripSudo(name, args)
			if cmdName == "zfs" && len(cmdArgs) > 0 && cmdArgs[0] == "diff" {
				if len(cmdArgs) >= 3 {
					diffDatasets = append(diffDatasets, strings.SplitN(cmdArgs[2], "@", 2)[0])
				}
				return []byte("+ /etc/new\n- /etc/old\nM /etc/mod\n"), nil
			}
			if cmdName == "zfs" && hasArgZFS(cmdArgs, "name,com.shedman:id") {
				return []byte(
					"tank/ROOT/default@20240101-000000\t20240101-000000\n" +
						"tank/ROOT/default/home@20240101-000000\t20240101-000000\n" +
						"tank/ROOT/default@20240102-000000\t20240102-000000\n" +
						"tank/ROOT/default/home@20240102-000000\t20240102-000000\n",
				), nil
			}
			return nil, nil
		},
		CommandFunc: func(name string, args ...string) *exec.Cmd {
			return exec.Command("true")
		},
	}

	backend := NewZFSBackend(nil, mock)
	backend.SetRootDataset("tank/ROOT/default")

	res, err := backend.Diff(context.Background(), "20240101-000000", "20240102-000000")
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}
	if len(res.Added) != 2 {
		t.Errorf("Unexpected added files: %v", res.Added)
	}
	for _, path := range res.Added {
		if path != "/etc/new" {
			t.Errorf("Unexpected added file: %s", path)
		}
	}
	if len(res.Removed) != 2 {
		t.Errorf("Unexpected removed files: %v", res.Removed)
	}
	for _, path := range res.Removed {
		if path != "/etc/old" {
			t.Errorf("Unexpected removed file: %s", path)
		}
	}
	if len(res.Modified) != 2 {
		t.Errorf("Unexpected modified files: %v", res.Modified)
	}
	for _, path := range res.Modified {
		if path != "/etc/mod" {
			t.Errorf("Unexpected modified file: %s", path)
		}
	}
	if len(diffDatasets) != 2 {
		t.Fatalf("Expected diff to run for 2 datasets, got %d", len(diffDatasets))
	}
	want := map[string]bool{
		"tank/ROOT/default":      false,
		"tank/ROOT/default/home": false,
	}
	for _, dataset := range diffDatasets {
		if _, ok := want[dataset]; ok {
			want[dataset] = true
		}
	}
	for dataset, ok := range want {
		if !ok {
			t.Errorf("Expected diff for dataset %s", dataset)
		}
	}
}

func TestZFSBackend_Delete_UsesTopLevelSnapshots(t *testing.T) {
	var commands [][]string
	mock := &executor.MockExecutor{
		OutputFunc: func(name string, args ...string) ([]byte, error) {
			cmdName, cmdArgs := stripSudo(name, args)
			if cmdName == "zfs" && hasArgZFS(cmdArgs, "name,com.shedman:id") {
				return []byte(
					"tank/ROOT/default@20240101-000000\t20240101-000000\n" +
						"tank/ROOT/default/home@20240101-000000\t20240101-000000\n" +
						"tank/home@20240101-000000\t20240101-000000\n",
				), nil
			}
			return nil, nil
		},
		CommandFunc: func(name string, args ...string) *exec.Cmd {
			cmdName, cmdArgs := stripSudo(name, args)
			commands = append(commands, append([]string{cmdName}, cmdArgs...))
			return exec.Command("true")
		},
	}

	backend := NewZFSBackend(nil, mock)

	if err := backend.Delete(context.Background(), "20240101-000000"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	expected := map[string]bool{
		"tank/ROOT/default@20240101-000000": false,
		"tank/home@20240101-000000":         false,
	}
	for _, cmd := range commands {
		if len(cmd) >= 4 && cmd[0] == "zfs" && cmd[1] == "destroy" && cmd[2] == "-r" {
			if _, ok := expected[cmd[3]]; ok {
				expected[cmd[3]] = true
			}
		}
	}
	for snap, ok := range expected {
		if !ok {
			t.Errorf("Expected destroy for %s", snap)
		}
	}
}

func TestZFSBackend_Restore_UsesTopLevelSnapshots(t *testing.T) {
	var commands [][]string
	mock := &executor.MockExecutor{
		OutputFunc: func(name string, args ...string) ([]byte, error) {
			cmdName, cmdArgs := stripSudo(name, args)
			if cmdName == "zfs" && hasArgZFS(cmdArgs, "name,com.shedman:id") {
				return []byte(
					"tank/ROOT/default@20240101-000000\t20240101-000000\n" +
						"tank/ROOT/default/home@20240101-000000\t20240101-000000\n" +
						"tank/home@20240101-000000\t20240101-000000\n",
				), nil
			}
			return nil, nil
		},
		CommandFunc: func(name string, args ...string) *exec.Cmd {
			cmdName, cmdArgs := stripSudo(name, args)
			commands = append(commands, append([]string{cmdName}, cmdArgs...))
			return exec.Command("true")
		},
	}

	backend := NewZFSBackend(nil, mock)

	if err := backend.Restore(context.Background(), "20240101-000000", RestoreOptions{}); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}

	expected := map[string]bool{
		"tank/ROOT/default@20240101-000000": false,
		"tank/home@20240101-000000":         false,
	}
	for _, cmd := range commands {
		if len(cmd) >= 4 && cmd[0] == "zfs" && cmd[1] == "rollback" && cmd[2] == "-r" {
			if _, ok := expected[cmd[3]]; ok {
				expected[cmd[3]] = true
			}
		}
	}
	for snap, ok := range expected {
		if !ok {
			t.Errorf("Expected rollback for %s", snap)
		}
	}
}

func TestZFSBackend_Push_Local_MultiDataset(t *testing.T) {
	tempDir := t.TempDir()
	mock := &executor.MockExecutor{
		OutputFunc: func(name string, args ...string) ([]byte, error) {
			cmdName, cmdArgs := stripSudo(name, args)
			if cmdName == "zfs" && hasArgZFS(cmdArgs, "name,com.shedman:id") {
				return []byte(
					"tank/ROOT/default@20240101-000000\t20240101-000000\n" +
						"tank/ROOT/default/home@20240101-000000\t20240101-000000\n" +
						"tank/home@20240101-000000\t20240101-000000\n",
				), nil
			}
			return nil, nil
		},
		CommandFunc: func(name string, args ...string) *exec.Cmd {
			return exec.Command("true")
		},
	}

	backend := NewZFSBackend(nil, mock)
	target := RemoteTarget{Type: StrategyLocal, Path: tempDir}

	if err := backend.Push(context.Background(), "20240101-000000", target, RemoteOptions{}); err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	expected := []string{
		filepath.Join(tempDir, zfsStreamFilename("tank/ROOT/default", "20240101-000000")),
		filepath.Join(tempDir, zfsStreamFilename("tank/home", "20240101-000000")),
	}
	for _, path := range expected {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("Expected stream file %s: %v", path, err)
		}
	}
	child := filepath.Join(tempDir, zfsStreamFilename("tank/ROOT/default/home", "20240101-000000"))
	if _, err := os.Stat(child); err == nil {
		t.Errorf("Did not expect stream file for child dataset: %s", child)
	}
}

func TestZFSBackend_Pull_Local_MultiDataset(t *testing.T) {
	tempDir := t.TempDir()
	id := "20240101-000000"
	files := []string{
		filepath.Join(tempDir, zfsStreamFilename("tank/ROOT/default", id)),
		filepath.Join(tempDir, zfsStreamFilename("tank/home", id)),
	}
	for _, path := range files {
		if err := os.WriteFile(path, []byte(""), 0o600); err != nil {
			t.Fatalf("Failed to create stream file: %v", err)
		}
	}

	var commands [][]string
	mock := &executor.MockExecutor{
		CommandFunc: func(name string, args ...string) *exec.Cmd {
			cmdName, cmdArgs := stripSudo(name, args)
			commands = append(commands, append([]string{cmdName}, cmdArgs...))
			return exec.Command("true")
		},
	}

	backend := NewZFSBackend(nil, mock)
	backend.SetRootDataset("tank/ROOT/default")

	source := RemoteTarget{Type: StrategyLocal, Path: tempDir}
	if err := backend.Pull(context.Background(), id, source, RemoteOptions{}); err != nil {
		t.Fatalf("Pull failed: %v", err)
	}

	expected := map[string]bool{
		"tank/ROOT/default": false,
		"tank/home":         false,
	}
	for _, cmd := range commands {
		if len(cmd) >= 4 && cmd[0] == "zfs" && cmd[1] == "receive" && cmd[2] == "-F" {
			if _, ok := expected[cmd[3]]; ok {
				expected[cmd[3]] = true
			}
		}
	}
	for dataset, ok := range expected {
		if !ok {
			t.Errorf("Expected receive for dataset %s", dataset)
		}
	}
}

func TestZFSBackend_ResolveSnapshotName(t *testing.T) {
	backend := NewZFSBackend(nil, &executor.MockExecutor{})
	backend.SetRootDataset("tank/ROOT/default")

	snap, err := backend.resolveSnapshotName("20240101-000000")
	if err != nil {
		t.Fatalf("resolveSnapshotName failed: %v", err)
	}
	if snap != "tank/ROOT/default@20240101-000000" {
		t.Errorf("Unexpected snapshot name: %s", snap)
	}

	snap, err = backend.resolveSnapshotName("tank/ROOT/default@20240101-000000")
	if err != nil {
		t.Fatalf("resolveSnapshotName failed: %v", err)
	}
	if snap != "tank/ROOT/default@20240101-000000" {
		t.Errorf("Unexpected snapshot name: %s", snap)
	}
}

func stripSudo(name string, args []string) (string, []string) {
	if name == "sudo" && len(args) >= 2 && args[0] == "-n" && args[1] == "zfs" {
		return "zfs", args[2:]
	}
	return name, args
}

func hasArgZFS(args []string, target string) bool {
	for _, arg := range args {
		if strings.TrimSpace(arg) == target {
			return true
		}
	}
	return false
}
