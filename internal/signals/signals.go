package signals

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/theshedman/shedman/internal/output"
)

// SignalHandler manages cleanup handlers and signal interception
type SignalHandler struct {
	handlers []func()
	mu       sync.Mutex
	captured bool
	stopCh   chan struct{}
}

// defaultHandler is the singleton instance for package-level functions
var defaultHandler = &SignalHandler{}

// NewSignalHandler creates a new signal handler instance
func NewSignalHandler() *SignalHandler {
	return &SignalHandler{}
}

// Setup listens for SIGINT/SIGTERM and runs cleanup handlers
func (h *SignalHandler) Setup() {
	h.mu.Lock()
	if h.captured {
		h.mu.Unlock()
		return
	}
	h.captured = true
	h.stopCh = make(chan struct{})
	h.mu.Unlock()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		select {
		case <-c:
			output.Warning("\nReceived interrupt signal. Cleaning up...")
			h.RunCleanup()
			os.Exit(1)
		case <-h.stopCh:
			signal.Stop(c)
			return
		}
	}()
}

// AddHandler registers a function to run on interrupt
func (h *SignalHandler) AddHandler(handler func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.handlers = append(h.handlers, handler)
}

// RunCleanup executes all registered cleanup handlers in reverse order (LIFO)
func (h *SignalHandler) RunCleanup() {
	h.mu.Lock()
	handlers := make([]func(), len(h.handlers))
	copy(handlers, h.handlers)
	h.handlers = nil
	h.mu.Unlock()

	for i := len(handlers) - 1; i >= 0; i-- {
		func() {
			defer func() {
				if r := recover(); r != nil {
					output.Error("Panic during cleanup: %v", r)
				}
			}()
			handlers[i]()
		}()
	}
}

// Reset clears all handlers and stops signal listening (for testing)
func (h *SignalHandler) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.handlers = nil

	if h.captured && h.stopCh != nil {
		close(h.stopCh)
		h.stopCh = nil
	}
	h.captured = false
}

// Package-level functions for production use

// SetupSignalHandler listens for SIGINT/SIGTERM and runs cleanup handlers
func SetupSignalHandler() {
	defaultHandler.Setup()
}

// AddCleanupHandler registers a function to run on interrupt
func AddCleanupHandler(handler func()) {
	defaultHandler.AddHandler(handler)
}

// RunCleanup executes all registered cleanup handlers
func RunCleanup() {
	defaultHandler.RunCleanup()
}

// ResetHandlers clears all handlers and stops listening (for testing only)
func ResetHandlers() {
	defaultHandler.Reset()
}
