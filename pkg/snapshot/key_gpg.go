package snapshot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/theshedman/shedman/pkg/executor"
)

// GPGKeyManager implements KeyManager using GPG
type GPGKeyManager struct {
	exec executor.Executor
}

// NewGPGKeyManager creates a new key manager
func NewGPGKeyManager(exec executor.Executor) *GPGKeyManager {
	return &GPGKeyManager{exec: exec}
}

// Generate generates a new key
func (k *GPGKeyManager) Generate(desc string) (string, error) {
	// Use quick-generate-key for automation
	// desc is used as "Name"
	userID := desc
	if userID == "" {
		userID = "ShedOS-Snapshot-Key"
	}

	// gpg --batch --passphrase '' --quick-generate-key "ID" default default
	// This generates a key without a passphrase for automation purposes (snapshot enc).
	args := []string{"--batch", "--passphrase", "", "--quick-generate-key", userID, "default", "default"}

	_, err := k.exec.Output("gpg", args...)
	if err != nil {
		return "", fmt.Errorf("gpg generation failed: %w", err)
	}

	// Let's try to find the key we just made.
	keys, err := k.List()
	if err != nil {
		return "", nil // ID lookup failed after creation
	}

	// Simple heuristic: find latest key matching description
	for _, key := range keys {
		if strings.Contains(key.Description, userID) {
			return key.ID, nil
		}
	}

	return "", fmt.Errorf("key generated but unable to retrieve ID")
}

// Export exports a key to file
func (k *GPGKeyManager) Export(id string, path string) error {
	// --export-secret-keys --armor --output file id
	_, err := k.exec.Output("gpg", "--export-secret-keys", "--armor", "--output", path, id)
	return err
}

// Import imports a key from file
func (k *GPGKeyManager) Import(path string) error {
	_, err := k.exec.Output("gpg", "--import", path)
	return err
}

// List lists keys
func (k *GPGKeyManager) List() ([]Key, error) {
	// gpg --list-secret-keys --with-colons
	// Format: sec:u:2048:1:FINGERPRINT:DATE::::::UID:...
	output, err := k.exec.Output("gpg", "--list-secret-keys", "--with-colons")
	if err != nil {
		return nil, err
	}

	var keys []Key
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		parts := strings.Split(line, ":")
		if len(parts) < 10 {
			continue
		}

		recordType := parts[0]

		if recordType == "sec" {
			// New secret key
			id := parts[4]
			// Date is unix timestamp
			var createdTS time.Time
			if ts, err := strconv.ParseInt(parts[5], 10, 64); err == nil {
				createdTS = time.Unix(ts, 0)
			}

			keys = append(keys, Key{
				ID:      id,
				Created: createdTS,
			})
		} else if recordType == "fpr" && len(keys) > 0 {
			// Fingerprint record, usually follows sec
			keys[len(keys)-1].Fingerprint = parts[9]
			keys[len(keys)-1].ID = parts[9] // Use full fingerprint as ID
		} else if recordType == "uid" && len(keys) > 0 {
			// User ID record
			keys[len(keys)-1].Description = parts[9]
		}
	}

	return keys, nil
}

// Delete deletes a key
func (k *GPGKeyManager) Delete(id string) error {
	// --delete-secret-key --batch --yes id
	_, err := k.exec.Output("gpg", "--delete-secret-key", "--batch", "--yes", id)
	return err
}
