package cmd

import (
	"bytes"   // Added for new tests
	"fmt"     // Added for new tests
	"strings" // Added for new tests
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

func TestRunCheck(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	// Mock CheckDatabase
	mock.CheckDatabaseFunc = func() error {
		return nil
	}

	var buf bytes.Buffer
	err := RunCheck(eng, &buf)
	if err != nil {
		t.Fatalf("RunCheck failed: %v", err)
	}

	if !strings.Contains(buf.String(), "database is consistent") {
		t.Errorf("Expected success message, got: %s", buf.String())
	}
}

func TestRunCheck_Failure(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	mock.CheckDatabaseFunc = func() error {
		return fmt.Errorf("missing dependencies")
	}

	var buf bytes.Buffer
	err := RunCheck(eng, &buf)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "database check failed") {
		t.Errorf("Expected error message 'database check failed', got: %v", err)
	}
}
