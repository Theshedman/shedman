package snapshot

import (
	"fmt"
	"strings"
	"testing"

	"github.com/theshedman/shedman/pkg/executor"
)

func TestSystemdScheduler_Status_Show(t *testing.T) {
	mock := &executor.MockExecutor{
		OutputFunc: func(name string, args ...string) ([]byte, error) {
			if hasArg(args, "show") {
				return []byte(
					"UnitFileState=enabled\n" +
						"ActiveState=active\n" +
						"NextElapseUSecRealtime=1700000000000000\n" +
						"LastTriggerUSecRealtime=1690000000000000\n" +
						"OnCalendar=daily\n",
				), nil
			}
			return nil, fmt.Errorf("unexpected command: %s %v", name, args)
		},
	}

	s := NewSystemdScheduler(mock)
	status, err := s.Status()
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if !status.Enabled {
		t.Error("Expected status.Enabled to be true")
	}
	if status.Frequency != "daily" {
		t.Errorf("Expected Frequency 'daily', got %s", status.Frequency)
	}
	if status.NextRun.IsZero() || status.LastRun.IsZero() {
		t.Errorf("Expected NextRun and LastRun to be populated, got %+v", status)
	}
	if status.NextRun.Unix() != 1700000000 {
		t.Errorf("Unexpected NextRun Unix time: %d", status.NextRun.Unix())
	}
	if status.LastRun.Unix() != 1690000000 {
		t.Errorf("Unexpected LastRun Unix time: %d", status.LastRun.Unix())
	}
}

func TestSystemdScheduler_Status_Fallback(t *testing.T) {
	mock := &executor.MockExecutor{
		OutputFunc: func(name string, args ...string) ([]byte, error) {
			if hasArg(args, "show") {
				return nil, fmt.Errorf("show failed")
			}
			if hasArg(args, "is-active") {
				return []byte("active\n"), nil
			}
			return nil, fmt.Errorf("unexpected command: %s %v", name, args)
		},
	}

	s := NewSystemdScheduler(mock)
	status, err := s.Status()
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if !status.Enabled {
		t.Error("Expected status.Enabled to be true in fallback path")
	}
	if !status.NextRun.IsZero() || !status.LastRun.IsZero() {
		t.Errorf("Expected empty run times on fallback, got %+v", status)
	}
	if status.Frequency != "" {
		t.Errorf("Expected empty frequency on fallback, got %s", status.Frequency)
	}
}

func hasArg(args []string, value string) bool {
	for _, arg := range args {
		if strings.TrimSpace(arg) == value {
			return true
		}
	}
	return false
}
