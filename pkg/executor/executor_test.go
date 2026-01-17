package executor

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRealExecutor_Output(t *testing.T) {
	exec := &RealExecutor{}

	// Test success case
	out, err := exec.Output("echo", "hello")
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}
	if strings.TrimSpace(string(out)) != "hello" {
		t.Errorf("Expected 'hello', got '%s'", out)
	}

	// Test failure case
	_, err = exec.Output("false")
	if err == nil {
		t.Error("Expected error from 'false', got nil")
	}
}

func TestRealExecutor_CommandContext(t *testing.T) {
	exec := &RealExecutor{}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Test command respecting context timeout
	// "sleep 0.5" should be killed by 100ms timeout
	cmd := exec.CommandContext(ctx, "sleep", "0.5")
	err := cmd.Run()

	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
	if ctx.Err() != context.DeadlineExceeded {
		t.Errorf("Expected DeadlineExceeded, got %v", ctx.Err())
	}
}
