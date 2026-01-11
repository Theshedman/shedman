package signals_test

import (
	"testing"
	"time"

	"github.com/theshedman/shedman/internal/signals"
)

func TestSignalHandler_AddHandler(t *testing.T) {
	h := signals.NewSignalHandler()
	defer h.Reset()

	called := false
	h.AddHandler(func() {
		called = true
	})

	h.RunCleanup()

	if !called {
		t.Error("Handler should have been called")
	}
}

func TestSignalHandler_LIFO(t *testing.T) {
	h := signals.NewSignalHandler()
	defer h.Reset()

	var order []int

	h.AddHandler(func() { order = append(order, 1) })
	h.AddHandler(func() { order = append(order, 2) })
	h.AddHandler(func() { order = append(order, 3) })

	h.RunCleanup()

	if len(order) != 3 {
		t.Fatalf("Expected 3 handlers, got %d", len(order))
	}

	// Should be LIFO order: 3, 2, 1
	if order[0] != 3 || order[1] != 2 || order[2] != 1 {
		t.Errorf("Expected LIFO order [3,2,1], got %v", order)
	}
}

func TestSignalHandler_LIFOWithManyHandlers(t *testing.T) {
	h := signals.NewSignalHandler()
	defer h.Reset()

	const numHandlers = 100
	var order []int

	for i := 0; i < numHandlers; i++ {
		n := i
		h.AddHandler(func() { order = append(order, n) })
	}

	h.RunCleanup()

	if len(order) != numHandlers {
		t.Fatalf("Expected %d handlers, got %d", numHandlers, len(order))
	}

	// Verify LIFO order
	for i := 0; i < numHandlers; i++ {
		expected := numHandlers - 1 - i
		if order[i] != expected {
			t.Errorf("At index %d: expected %d, got %d", i, expected, order[i])
		}
	}
}

func TestSignalHandler_PanicRecovery(t *testing.T) {
	h := signals.NewSignalHandler()
	defer h.Reset()

	calledBefore := false
	calledAfter := false

	// Handler added first runs last (LIFO)
	h.AddHandler(func() {
		calledBefore = true
	})
	h.AddHandler(func() {
		panic("test panic")
	})
	h.AddHandler(func() {
		calledAfter = true
	})

	// Should not panic and should run all handlers
	h.RunCleanup()

	if !calledAfter {
		t.Error("Handler after panic should be called (runs first in LIFO)")
	}
	if !calledBefore {
		t.Error("Handler before panic should be called (runs last in LIFO)")
	}
}

func TestSignalHandler_Reset(t *testing.T) {
	h := signals.NewSignalHandler()

	called := false
	h.AddHandler(func() {
		called = true
	})

	h.Reset()
	h.RunCleanup()

	if called {
		t.Error("Handler should not be called after reset")
	}
}

func TestSignalHandler_MultipleSetup(t *testing.T) {
	h := signals.NewSignalHandler()
	defer h.Reset()

	// First setup should succeed
	h.Setup()

	// Second setup should be a no-op (no panic)
	h.Setup()
}

func TestSignalHandler_SetupAndReset(t *testing.T) {
	h := signals.NewSignalHandler()

	h.Setup()

	// Give goroutine time to start
	time.Sleep(10 * time.Millisecond)

	// Reset should clean up the goroutine
	h.Reset()

	// Should be able to setup again after reset
	h.Setup()
	h.Reset()
}

func TestSignalHandler_RunCleanupClearsHandlers(t *testing.T) {
	h := signals.NewSignalHandler()
	defer h.Reset()

	callCount := 0
	h.AddHandler(func() {
		callCount++
	})

	h.RunCleanup()
	h.RunCleanup() // Second call should do nothing

	if callCount != 1 {
		t.Errorf("Expected handler to be called once, got %d", callCount)
	}
}

func TestPackageLevelFunctions(t *testing.T) {
	// Reset to clean state
	signals.ResetHandlers()
	defer signals.ResetHandlers()

	called := false
	signals.AddCleanupHandler(func() {
		called = true
	})

	signals.RunCleanup()

	if !called {
		t.Error("Package-level handler should have been called")
	}
}

func TestPackageLevelFunctions_Independent(t *testing.T) {
	// Ensure clean state
	signals.ResetHandlers()
	defer signals.ResetHandlers()

	// Verify handlers from previous tests don't leak
	count := 0
	signals.AddCleanupHandler(func() {
		count++
	})

	signals.RunCleanup()

	if count != 1 {
		t.Errorf("Expected 1 call, got %d (handler may have leaked from previous test)", count)
	}
}
