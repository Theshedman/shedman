package snapshot

import (
	"context"
	"os/exec"
	"testing"

	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/pkg/executor"
)

func TestSnapperBackend_Create(t *testing.T) {
	cfg := config.Default()

	mockExec := &executor.MockExecutor{
		// CommandFunc handles execution of 'sudo' commands
		CommandFunc: func(name string, args ...string) *exec.Cmd {
			if name != "sudo" || len(args) == 0 {
				return exec.Command("true")
			}

			// Handle 'sudo snapper' commands
			if args[0] == "snapper" {
				subCmd := args[1]
				// Handle specific snpaper subcommands
				switch {
				// 'list-configs': Mock CSV output for config detection
				case len(args) > 2 && args[1] == "--csvout" && args[2] == "list-configs":
					return exec.Command("echo", "config,subvolume\nroot,/\n")

				// 'create': Return success (exit 0)
				case subCmd == "create":
					return exec.Command("true")
				}
			}
			return exec.Command("true")
		},
		// OutputFunc handles direct output calls if any (none expected for Create flow requiring mocking)
		OutputFunc: func(name string, args ...string) ([]byte, error) {
			return nil, nil
		},
	}

	backend := NewSnapperBackend(cfg, mockExec)

	snap, err := backend.Create(context.Background(), "test snapshot", CreateOptions{Type: "single"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if snap.ID == "" || len(snap.ID) != 15 {
		t.Errorf("Expected valid timestamp ID, got '%s'", snap.ID)
	}
	if snap.Backend != "snapper" {
		t.Errorf("Expected backend 'snapper', got '%s'", snap.Backend)
	}
}

func TestSnapperBackend_DryRun(t *testing.T) {
	cfg := config.Default()
	mockExec := &executor.MockExecutor{
		OutputFunc: func(name string, args ...string) ([]byte, error) {
			// Allow read-only config detection
			if len(args) > 0 && args[1] == "list-configs" {
				return []byte("config,subvolume\nroot,/\n"), nil
			}
			t.Errorf("Unexpected command execution in dry-run: %s %v", name, args)
			return nil, nil
		},
	}
	backend := NewSnapperBackend(cfg, mockExec)

	snap, err := backend.Create(context.Background(), "dry run test", CreateOptions{DryRun: true})
	if err != nil {
		t.Fatalf("DryRun Create failed: %v", err)
	}
	if snap.ID != "dry-run" {
		t.Errorf("Expected dry-run ID, got '%s'", snap.ID)
	}
}

func TestSnapperBackend_List_Local(t *testing.T) {
	cfg := config.Default()

	mockExec := &executor.MockExecutor{
		CommandFunc: func(name string, args ...string) *exec.Cmd {
			cmdName, cmdArgs := normalizeSnapperCmd(name, args)
			if cmdName != "snapper" {
				return exec.Command("true")
			}
			if hasArgs(cmdArgs, "--csvout", "list-configs") {
				return exec.Command("echo", "config,subvolume\nroot,/\n")
			}
			if hasArgs(cmdArgs, "--csvout", "--iso", "list") {
				return exec.Command("echo", "number,type,date,used-space,description,userdata\n"+
					"1,single,2023-01-01 00:00:00,1.0 MiB,Test,shedman_id=20230101-000000\n")
			}
			return exec.Command("true")
		},
	}

	backend := NewSnapperBackend(cfg, mockExec)
	snaps, err := backend.List(context.Background(), ListOptions{})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(snaps) != 1 {
		t.Fatalf("Expected 1 snapshot, got %d", len(snaps))
	}
	if snaps[0].ID != "20230101-000000" {
		t.Errorf("Unexpected snapshot ID: %s", snaps[0].ID)
	}
}

func hasArgs(args []string, items ...string) bool {
	for i := 0; i <= len(args)-len(items); i++ {
		match := true
		for j, item := range items {
			if args[i+j] != item {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func normalizeSnapperCmd(name string, args []string) (string, []string) {
	if name == "sudo" && len(args) >= 2 && args[1] == "snapper" {
		return "snapper", args[2:]
	}
	return name, args
}
