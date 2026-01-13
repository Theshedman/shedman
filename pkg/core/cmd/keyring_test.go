package cmd

import (
	"fmt"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

func TestRunKeyring(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	tests := []struct {
		name      string
		action    string
		arg       string // keyID or path
		mockError error
		wantError bool
		wantList  []string // Only for list action
	}{
		{
			name:      "Init Success",
			action:    "init",
			wantError: false,
		},
		{
			name:      "Init Fail",
			action:    "init",
			mockError: fmt.Errorf("init failed"),
			wantError: true,
		},
		{
			name:      "Refresh Success",
			action:    "refresh",
			wantError: false,
		},
		{
			name:      "List Success",
			action:    "list",
			wantList:  []string{"key1", "key2"},
			wantError: false,
		},
		{
			name:      "Add Success",
			action:    "add",
			arg:       "123456",
			wantError: false,
		},
		{
			name:      "Add Fail",
			action:    "add",
			arg:       "123456",
			mockError: fmt.Errorf("add failed"),
			wantError: true,
		},
		{
			name:      "Remove Success",
			action:    "remove",
			arg:       "123456",
			wantError: false,
		},
		{
			name:      "Import Success",
			action:    "import",
			arg:       "/tmp/key.gpg",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mocks
			mock.InitKeyringFunc = func() error { return tt.mockError }
			mock.RefreshKeysFunc = func() error { return tt.mockError }
			mock.ListKeysFunc = func() ([]string, error) { return tt.wantList, tt.mockError }
			mock.AddKeyFunc = func(id string) error {
				if id != tt.arg {
					return fmt.Errorf("wrong arg: got %s want %s", id, tt.arg)
				}
				return tt.mockError
			}
			mock.RemoveKeyFunc = func(id string) error {
				if id != tt.arg {
					return fmt.Errorf("wrong arg: got %s want %s", id, tt.arg)
				}
				return tt.mockError
			}
			mock.ImportKeyFunc = func(path string) error {
				if path != tt.arg {
					return fmt.Errorf("wrong arg: got %s want %s", path, tt.arg)
				}
				return tt.mockError
			}

			var err error
			switch tt.action {
			case "init":
				err = RunKeyringInit(eng)
			case "refresh":
				err = RunKeyringRefresh(eng)
			case "list":
				err = RunKeyringList(eng)
			case "add":
				err = RunKeyringAdd(eng, tt.arg)
			case "remove":
				err = RunKeyringRemove(eng, tt.arg)
			case "import":
				err = RunKeyringImport(eng, tt.arg)
			}

			if (err != nil) != tt.wantError {
				t.Errorf("Action %s error = %v, wantError %v", tt.action, err, tt.wantError)
			}
		})
	}
}
