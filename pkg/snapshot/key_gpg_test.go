package snapshot

import (
	"reflect"
	"strings"
	"testing"
)

// GPGMockExecutor for testing
type GPGMockExecutor struct {
	Calls []executionCall
	// Simple map for output: cmd -> output (not context aware by default)
	// Or handler function
	Handler   func(cmd string, args ...string) ([]byte, error)
	OutputMap map[string]MockOutput
}

type executionCall struct {
	Cmd  string
	Args []string
}

type MockOutput struct {
	Output []byte
	Err    error
}

func (m *GPGMockExecutor) Output(cmd string, args ...string) ([]byte, error) {
	m.Calls = append(m.Calls, executionCall{Cmd: cmd, Args: args})

	if m.Handler != nil {
		return m.Handler(cmd, args...)
	}

	if out, ok := m.OutputMap[cmd]; ok {
		return out.Output, out.Err
	}
	// Fallback for known subcommands if needed or return empty
	return []byte{}, nil
}

func (m *GPGMockExecutor) Run(cmd string, args ...string) error {
	m.Calls = append(m.Calls, executionCall{Cmd: cmd, Args: args})
	if m.Handler != nil {
		_, err := m.Handler(cmd, args...)
		return err
	}
	return nil
}

func (m *GPGMockExecutor) LookPath(file string) (string, error) {
	return "/bin/" + file, nil
}

// Helper for test assertions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.Contains(s, item) {
			return true
		}
	}
	return false
}

func TestGPGKeyManager_Generate(t *testing.T) {
	mockExec := &GPGMockExecutor{
		OutputMap: map[string]MockOutput{
			"gpg": {Output: []byte(""), Err: nil},
		},
	}

	// Complex mock to handle sequential calls (Generate then List)
	mockExec.Handler = func(cmd string, args ...string) ([]byte, error) {
		if cmd == "gpg" {
			// Check if it's list command
			isList := false
			for _, arg := range args {
				if arg == "--list-secret-keys" {
					isList = true
					break
				}
			}

			if isList {
				return []byte(`sec:u:2048:1:KEYID123:1234567890::::::UID:Test Key <test@example.com>
fpr:::::::::FINGERPRINT123:
uid:::::::::Test Key <test@example.com>:
`), nil
			}
		}
		return []byte(""), nil
	}

	km := NewGPGKeyManager(mockExec)
	id, err := km.Generate("Test Key")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if id != "FINGERPRINT123" {
		t.Errorf("Expected ID FINGERPRINT123, got %s", id)
	}

	// Verify Generate command was called
	foundGen := false
	for _, call := range mockExec.Calls {
		if call.Cmd == "gpg" && contains(call.Args, "--quick-generate-key") {
			foundGen = true
			if !contains(call.Args, "Test Key") {
				t.Errorf("Expected user ID 'Test Key' in args, got %v", call.Args)
			}
		}
	}
	if !foundGen {
		t.Error("gpg --quick-generate-key was not called")
	}
}

func TestGPGKeyManager_List(t *testing.T) {
	mockExec := &GPGMockExecutor{
		OutputMap: map[string]MockOutput{
			"gpg": {
				Output: []byte(`sec:u:2048:1:KEYID1:1234567890::::::UID:
fpr:::::::::FINGERPRINT1:
uid:::::::::Key 1:
sec:u:2048:1:KEYID2:1234567890::::::UID:
fpr:::::::::FINGERPRINT2:
uid:::::::::Key 2:
`),
			},
		},
	}

	km := NewGPGKeyManager(mockExec)
	keys, err := km.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(keys) != 2 {
		t.Fatalf("Expected 2 keys, got %d", len(keys))
	}

	if keys[0].ID != "FINGERPRINT1" || keys[0].Description != "Key 1" {
		t.Errorf("Unexpected key 1 data: %+v", keys[0])
	}
	if keys[1].ID != "FINGERPRINT2" || keys[1].Description != "Key 2" {
		t.Errorf("Unexpected key 2 data: %+v", keys[1])
	}
}

func TestGPGKeyManager_Export(t *testing.T) {
	mockExec := &GPGMockExecutor{}
	km := NewGPGKeyManager(mockExec)

	err := km.Export("key1", "key1.asc")
	if err != nil {
		t.Errorf("Export failed: %v", err)
	}

	expectedArgs := []string{"--export-secret-keys", "--armor", "--output", "key1.asc", "key1"}
	if len(mockExec.Calls) != 1 {
		t.Fatalf("Expected 1 call, got %d", len(mockExec.Calls))
	}

	if !reflect.DeepEqual(mockExec.Calls[0].Args, expectedArgs) {
		t.Errorf("Args mismatch. Want %v, got %v", expectedArgs, mockExec.Calls[0].Args)
	}
}

func TestGPGKeyManager_Delete(t *testing.T) {
	mockExec := &GPGMockExecutor{}
	km := NewGPGKeyManager(mockExec)

	err := km.Delete("key1")
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}

	expectedArgs := []string{"--delete-secret-key", "--batch", "--yes", "key1"}
	if !reflect.DeepEqual(mockExec.Calls[0].Args, expectedArgs) {
		t.Errorf("Args mismatch. Want %v, got %v", expectedArgs, mockExec.Calls[0].Args)
	}
}
