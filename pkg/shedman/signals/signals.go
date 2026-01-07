package signals

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/theshedman/shedman/pkg/shedman/output"
)

var (
	cleanupHandlers []func()
	mu              sync.Mutex
	captured        bool
)

// SetupSignalHandler listens for SIGINT/SIGTERM and runs cleanup handlers
func SetupSignalHandler() {
	mu.Lock()
	if captured {
		mu.Unlock()
		return
	}
	captured = true
	mu.Unlock()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		output.Warning("\nReceived interrupt signal. Cleaning up...")
		RunCleanup()
		os.Exit(1)
	}()
}

// AddCleanupHandler registers a function to run on interrupt
func AddCleanupHandler(handler func()) {
	mu.Lock()
	defer mu.Unlock()
	cleanupHandlers = append(cleanupHandlers, handler)
}

// RunCleanup executes all registered cleanup handlers
func RunCleanup() {
	mu.Lock()
	defer mu.Unlock()

	// Run in reverse order (LIFO)
	for i := len(cleanupHandlers) - 1; i >= 0; i-- {
		// Recover from panics in handlers to ensure all run
		func() {
			defer func() {
				if r := recover(); r != nil {
					output.Error("Panic during cleanup: %v", r)
				}
			}()
			cleanupHandlers[i]()
		}()
	}

	cleanupHandlers = nil
}
