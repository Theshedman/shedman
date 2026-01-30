package keyring

import (
	"fmt"
	"os"
	"strings"

	"github.com/theshedman/shedman/pkg/executor"
)

// Manager handles GPG keyring operations
type Manager struct {
	path       string
	exec       executor.Executor
	keyservers []string
}

// New creates a new keyring manager
func New(path string) *Manager {
	return &Manager{
		path:       path,
		exec:       &executor.RealExecutor{},
		keyservers: []string{"keyserver.ubuntu.com", "keys.openpgp.org"},
	}
}

// NewWithExecutor creates a new keyring manager with a custom executor (for testing)
func NewWithExecutor(path string, exec executor.Executor) *Manager {
	if exec == nil {
		exec = &executor.RealExecutor{}
	}
	return &Manager{
		path:       path,
		exec:       exec,
		keyservers: []string{"keyserver.ubuntu.com", "keys.openpgp.org"},
	}
}

// Key represents a GPG key
type Key struct {
	ID    string
	Name  string
	Email string
}

// List lists keys
func (m *Manager) List() ([]Key, error) {
	args := m.gpgArgs("--with-colons", "--list-keys")
	out, err := m.exec.Output("gpg", args...)
	if err != nil {
		return nil, err
	}

	var keys []Key
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		parts := strings.Split(line, ":")
		if len(parts) == 0 {
			continue
		}
		switch parts[0] {
		case "pub":
			if len(parts) > 4 {
				keys = append(keys, Key{ID: parts[4]})
			}
		case "fpr":
			if len(keys) > 0 && len(parts) > 9 {
				keys[len(keys)-1].ID = parts[9]
			}
		case "uid":
			if len(keys) > 0 {
				uidField := ""
				if len(parts) > 9 {
					uidField = parts[9]
				} else if len(parts) > 0 {
					uidField = parts[len(parts)-1]
				}
				if strings.TrimSpace(uidField) == "" {
					for i := len(parts) - 1; i >= 0; i-- {
						if strings.TrimSpace(parts[i]) != "" {
							uidField = parts[i]
							break
						}
					}
				}
				name, email := parseUID(uidField)
				keys[len(keys)-1].Name = name
				keys[len(keys)-1].Email = email
			}
		}
	}

	return keys, nil
}

// Add adds a key
func (m *Manager) Add(id string) error {
	if id == "" {
		return fmt.Errorf("key id is required")
	}

	for _, ks := range m.keyservers {
		args := m.gpgArgs("--keyserver", ks, "--recv-keys", id)
		if _, err := m.exec.Output("gpg", args...); err == nil {
			return nil
		}
	}

	return fmt.Errorf("failed to fetch key %s from keyservers", id)
}

// Remove removes a key
func (m *Manager) Remove(id string) error {
	if id == "" {
		return fmt.Errorf("key id is required")
	}

	args := m.gpgArgs("--batch", "--yes", "--delete-keys", id)
	_, err := m.exec.Output("gpg", args...)
	return err
}

func (m *Manager) gpgArgs(args ...string) []string {
	if m.path == "" {
		return args
	}

	_ = os.MkdirAll(m.path, 0700)
	return append([]string{"--homedir", m.path}, args...)
}

func parseUID(uid string) (string, string) {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return "", ""
	}

	if lt := strings.LastIndex(uid, "<"); lt != -1 {
		if gt := strings.LastIndex(uid, ">"); gt > lt {
			name := strings.TrimSpace(uid[:lt])
			email := strings.TrimSpace(uid[lt+1 : gt])
			return name, email
		}
	}

	return uid, ""
}
