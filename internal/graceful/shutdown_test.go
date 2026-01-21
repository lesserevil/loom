package graceful

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestShutdownManager(t *testing.T) {
	sm := NewShutdownManager(5 * time.Second)

	// Track callback execution order
	var executed []int

	// Register callbacks
	sm.RegisterCallback(func(ctx context.Context) error {
		executed = append(executed, 1)
		return nil
	})

	sm.RegisterCallback(func(ctx context.Context) error {
		executed = append(executed, 2)
		return nil
	})

	sm.RegisterCallback(func(ctx context.Context) error {
		executed = append(executed, 3)
		return nil
	})

	// Execute shutdown
	err := sm.Shutdown()
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// Verify callbacks executed in reverse order (LIFO)
	if len(executed) != 3 {
		t.Fatalf("Expected 3 callbacks, got %d", len(executed))
	}

	if executed[0] != 3 || executed[1] != 2 || executed[2] != 1 {
		t.Errorf("Expected callbacks in reverse order [3,2,1], got %v", executed)
	}

	// Second shutdown should be no-op
	err = sm.Shutdown()
	if err != nil {
		t.Fatalf("Second shutdown should succeed: %v", err)
	}

	// Callbacks should not execute again
	if len(executed) != 3 {
		t.Errorf("Callbacks should only execute once, got %d executions", len(executed))
	}
}

func TestShutdownManagerWithError(t *testing.T) {
	sm := NewShutdownManager(5 * time.Second)

	expectedErr := errors.New("shutdown error")

	sm.RegisterCallback(func(ctx context.Context) error {
		return nil
	})

	sm.RegisterCallback(func(ctx context.Context) error {
		return expectedErr
	})

	sm.RegisterCallback(func(ctx context.Context) error {
		return nil
	})

	// Execute shutdown
	err := sm.Shutdown()
	if err == nil {
		t.Fatal("Expected shutdown to return error")
	}

	// Should return first error encountered
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestStartupManager(t *testing.T) {
	sm := NewStartupManager()

	// Initially not ready
	if sm.IsReady() {
		t.Error("Expected IsReady=false initially")
	}

	// Register gates
	sm.RegisterGate("database", func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	}, 1*time.Second)

	sm.RegisterGate("cache", func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	}, 1*time.Second)

	// Wait for readiness
	ctx := context.Background()
	err := sm.WaitUntilReady(ctx)
	if err != nil {
		t.Fatalf("WaitUntilReady failed: %v", err)
	}

	// Should be ready now
	if !sm.IsReady() {
		t.Error("Expected IsReady=true after waiting")
	}
}

func TestStartupManagerTimeout(t *testing.T) {
	sm := NewStartupManager()

	// Register gate that takes too long
	sm.RegisterGate("slow", func(ctx context.Context) error {
		select {
		case <-time.After(5 * time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}, 100*time.Millisecond)

	// Wait with short timeout
	err := sm.WaitWithTimeout(200 * time.Millisecond)
	if err == nil {
		t.Fatal("Expected timeout error")
	}

	// Should not be ready
	if sm.IsReady() {
		t.Error("Expected IsReady=false after timeout")
	}
}

func TestStartupManagerFailure(t *testing.T) {
	sm := NewStartupManager()

	expectedErr := errors.New("gate check failed")

	sm.RegisterGate("database", func(ctx context.Context) error {
		return nil
	}, 1*time.Second)

	sm.RegisterGate("failing", func(ctx context.Context) error {
		return expectedErr
	}, 1*time.Second)

	// Wait for readiness
	ctx := context.Background()
	err := sm.WaitUntilReady(ctx)
	if err == nil {
		t.Fatal("Expected error from failing gate")
	}

	// Should not be ready
	if sm.IsReady() {
		t.Error("Expected IsReady=false after failure")
	}
}

func TestDefaultTimeout(t *testing.T) {
	sm := NewShutdownManager(0)
	if sm.shutdownTimeout != 30*time.Second {
		t.Errorf("Expected default timeout 30s, got %v", sm.shutdownTimeout)
	}
}

func TestGateDefaultTimeout(t *testing.T) {
	sm := NewStartupManager()
	sm.RegisterGate("test", func(ctx context.Context) error {
		return nil
	}, 0)

	// Check that default timeout was applied
	if len(sm.readyGates) != 1 {
		t.Fatalf("Expected 1 gate, got %d", len(sm.readyGates))
	}

	if sm.readyGates[0].Timeout != 30*time.Second {
		t.Errorf("Expected default timeout 30s, got %v", sm.readyGates[0].Timeout)
	}
}
