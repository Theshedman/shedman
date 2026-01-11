package log

import (
	"os"
	"testing"
	"time"
)

func TestManager_List(t *testing.T) {
	// Create temp log file
	tmpLog, err := os.CreateTemp("", "pacman.log")
	if err != nil {
		t.Fatalf("Failed to create temp log: %v", err)
	}
	defer os.Remove(tmpLog.Name())

	content := `[2023-11-20T10:00:00+0000] [ALPM] transaction started
[2023-11-20T10:00:01+0000] [ALPM] installed go (2:1.21.4-1)
[2023-11-20T10:00:02+0000] [ALPM] transaction completed
[2023-11-21T11:00:00+0000] [ALPM] transaction started
[2023-11-21T11:00:01+0000] [ALPM] removed vim (9.0-1)
[2023-11-21T11:00:02+0000] [ALPM] transaction completed
`
	if _, err := tmpLog.WriteString(content); err != nil {
		t.Fatalf("Failed to write log content: %v", err)
	}
	tmpLog.Close()

	mgr := New(tmpLog.Name())
	txs, err := mgr.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(txs) != 2 {
		t.Errorf("Expected 2 transactions, got %d", len(txs))
	}

	if len(txs[0].Packages) != 1 || txs[0].Packages[0] != "go (2:1.21.4-1)" {
		t.Errorf("Unexpected first transaction packages: %v", txs[0].Packages)
	}

	if txs[0].Action != "installed" {
		t.Errorf("Expected action 'installed', got '%s'", txs[0].Action)
	}

	if txs[1].Action != "removed" {
		t.Errorf("Expected action 'removed', got '%s'", txs[1].Action)
	}
}

func TestManager_ParsesDate(t *testing.T) {
	tmpLog, _ := os.CreateTemp("", "pacman.log")
	defer os.Remove(tmpLog.Name())

	content := `[2023-11-20T10:00:00+0000] [ALPM] transaction started
[2023-11-20T10:00:02+0000] [ALPM] transaction completed
`
	tmpLog.WriteString(content)
	tmpLog.Close()

	mgr := New(tmpLog.Name())
	txs, _ := mgr.List()

	if len(txs) != 1 {
		t.Fatalf("Expected 1 transaction")
	}

	expectedTime, _ := time.Parse("2006-01-02T15:04:05-0700", "2023-11-20T10:00:00+0000")
	if !txs[0].Timestamp.Equal(expectedTime) {
		t.Errorf("Timestamp mismatch. Got %v, want %v", txs[0].Timestamp, expectedTime)
	}
}
