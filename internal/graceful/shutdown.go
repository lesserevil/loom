package graceful

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// ShutdownManager handles graceful shutdown of the application.
type ShutdownManager struct {
	shutdownTimeout time.Duration
	callbacks       []ShutdownCallback
	mu              sync.Mutex
	shutdownOnce    sync.Once
}

// ShutdownCallback is a function called during shutdown.
type ShutdownCallback func(ctx context.Context) error

// NewShutdownManager creates a new shutdown manager.
func NewShutdownManager(timeout time.Duration) *ShutdownManager {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &ShutdownManager{
		shutdownTimeout: timeout,
		callbacks:       make([]ShutdownCallback, 0),
	}
}

// RegisterCallback registers a function to be called during shutdown.
// Callbacks are called in reverse order of registration (LIFO).
func (sm *ShutdownManager) RegisterCallback(cb ShutdownCallback) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.callbacks = append(sm.callbacks, cb)
}

// WaitForShutdown blocks until a shutdown signal is received.
// It then executes all registered callbacks in reverse order.
func (sm *ShutdownManager) WaitForShutdown() error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	log.Printf("[INFO] Received signal: %v, starting graceful shutdown...", sig)

	return sm.Shutdown()
}

// Shutdown performs graceful shutdown.
func (sm *ShutdownManager) Shutdown() error {
	var shutdownErr error

	sm.shutdownOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), sm.shutdownTimeout)
		defer cancel()

		sm.mu.Lock()
		callbacks := make([]ShutdownCallback, len(sm.callbacks))
		copy(callbacks, sm.callbacks)
		sm.mu.Unlock()

		// Execute callbacks in reverse order (LIFO)
		for i := len(callbacks) - 1; i >= 0; i-- {
			log.Printf("[INFO] Executing shutdown callback %d/%d", len(callbacks)-i, len(callbacks))

			if err := callbacks[i](ctx); err != nil {
				log.Printf("[ERROR] Shutdown callback failed: %v", err)
				if shutdownErr == nil {
					shutdownErr = err
				}
			}
		}

		log.Println("[INFO] Graceful shutdown complete")
	})

	return shutdownErr
}

// StartupManager handles graceful startup with health gates.
type StartupManager struct {
	readyGates []ReadyGate
	mu         sync.RWMutex
	ready      bool
}

// ReadyGate is a function that checks if a component is ready.
type ReadyGate struct {
	Name    string
	Check   func(ctx context.Context) error
	Timeout time.Duration
}

// NewStartupManager creates a new startup manager.
func NewStartupManager() *StartupManager {
	return &StartupManager{
		readyGates: make([]ReadyGate, 0),
		ready:      false,
	}
}

// RegisterGate registers a readiness gate.
func (sm *StartupManager) RegisterGate(name string, check func(ctx context.Context) error, timeout time.Duration) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if timeout == 0 {
		timeout = 30 * time.Second
	}

	sm.readyGates = append(sm.readyGates, ReadyGate{
		Name:    name,
		Check:   check,
		Timeout: timeout,
	})
}

// WaitUntilReady waits for all readiness gates to pass.
// Returns an error if any gate fails or times out.
func (sm *StartupManager) WaitUntilReady(ctx context.Context) error {
	log.Println("[INFO] Starting readiness checks...")

	sm.mu.RLock()
	gates := make([]ReadyGate, len(sm.readyGates))
	copy(gates, sm.readyGates)
	sm.mu.RUnlock()

	for i, gate := range gates {
		log.Printf("[INFO] Checking readiness gate %d/%d: %s", i+1, len(gates), gate.Name)

		gateCtx, cancel := context.WithTimeout(ctx, gate.Timeout)
		defer cancel()

		start := time.Now()
		err := gate.Check(gateCtx)
		duration := time.Since(start)

		if err != nil {
			return fmt.Errorf("readiness gate '%s' failed after %v: %w", gate.Name, duration, err)
		}

		log.Printf("[INFO] Readiness gate '%s' passed in %v", gate.Name, duration)
	}

	sm.mu.Lock()
	sm.ready = true
	sm.mu.Unlock()

	log.Println("[INFO] All readiness gates passed - application is ready!")
	return nil
}

// IsReady returns true if all readiness gates have passed.
func (sm *StartupManager) IsReady() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.ready
}

// WaitWithTimeout waits for the manager to be ready or timeout.
func (sm *StartupManager) WaitWithTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return sm.WaitUntilReady(ctx)
}
