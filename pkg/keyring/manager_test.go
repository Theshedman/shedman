package keyring

import (
	"context"
	"os/exec"
	"strings"
	"testing"
)

type mockExecutor struct {
	output func(name string, args ...string) ([]byte, error)
	calls  []string
}

func (m *mockExecutor) Output(name string, args ...string) ([]byte, error) {
	m.calls = append(m.calls, name+" "+strings.Join(args, " "))
	if m.output != nil {
		return m.output(name, args...)
	}
	return nil, nil
}

func (m *mockExecutor) Command(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

func (m *mockExecutor) CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}

func TestNew(t *testing.T) {
	path := "/tmp/keyring"
	m := New(path)
	if m == nil {
		t.Fatal("New returned nil")
	}
	if m.path != path {
		t.Errorf("Expected path %s, got %s", path, m.path)
	}
}

func TestManager_Operations(t *testing.T) {
	exec := &mockExecutor{
		output: func(name string, args ...string) ([]byte, error) {
			if name == "gpg" && containsArg(args, "--list-keys") {
				return []byte("pub:u:2048:1:ABCDEF1234567890:1700000000:::::::\n" +
					"fpr:::::::::1234567890ABCDEF1234567890ABCDEF12345678:\n" +
					"uid:u:::::::John Doe <john@example.com>:\n"), nil
			}
			return []byte(""), nil
		},
	}
	m := NewWithExecutor("/tmp/keyring", exec)
	m.keyservers = []string{"keyserver.test"}

	// Test List
	keys, err := m.List()
	if err != nil {
		t.Errorf("List returned error: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("Expected 1 key, got %d", len(keys))
	}
	if keys[0].ID != "1234567890ABCDEF1234567890ABCDEF12345678" {
		t.Errorf("Unexpected key ID: %s", keys[0].ID)
	}
	if keys[0].Name != "John Doe" || keys[0].Email != "john@example.com" {
		t.Errorf("Unexpected key identity: %+v", keys[0])
	}

	// Test Add
	if err := m.Add("test-id"); err != nil {
		t.Errorf("Add returned error: %v", err)
	}
	if !callContains(exec.calls, "--keyserver keyserver.test") {
		t.Errorf("Expected Add to use configured keyserver, calls: %v", exec.calls)
	}

	// Test Remove
	if err := m.Remove("test-id"); err != nil {
		t.Errorf("Remove returned error: %v", err)
	}
}

func containsArg(args []string, target string) bool {
	for _, arg := range args {
		if arg == target {
			return true
		}
	}
	return false
}

func callContains(calls []string, target string) bool {
	for _, call := range calls {
		if strings.Contains(call, target) {
			return true
		}
	}
	return false
}
