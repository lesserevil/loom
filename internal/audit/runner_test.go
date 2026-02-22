package audit

import (
	"context"
	"testing"
	"time"
)

func TestNewRunner_Defaults(t *testing.T) {
	mock := &mockBeadCreator{}
	runner := NewRunner("loom", "/tmp/test", 0, mock)

	if runner.projectID != "loom" {
		t.Errorf("projectID = %q, want %q", runner.projectID, "loom")
	}
	if runner.projectPath != "/tmp/test" {
		t.Errorf("projectPath = %q, want %q", runner.projectPath, "/tmp/test")
	}
	if runner.intervalMinutes != 30 {
		t.Errorf("intervalMinutes = %d, want 30 (default)", runner.intervalMinutes)
	}
	if runner.activity == nil {
		t.Error("expected non-nil activity")
	}
	if runner.beadCreator == nil {
		t.Error("expected non-nil beadCreator")
	}
}

func TestNewRunner_CustomInterval(t *testing.T) {
	runner := NewRunner("proj", ".", 15, nil)
	if runner.intervalMinutes != 15 {
		t.Errorf("intervalMinutes = %d, want 15", runner.intervalMinutes)
	}
}

func TestRunner_Stop(t *testing.T) {
	runner := NewRunner("loom", ".", 1, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		runner.Start(ctx)
		close(done)
	}()

	runner.Stop()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runner did not stop within 2s")
	}
}

func TestRunner_ContextCancel(t *testing.T) {
	runner := NewRunner("loom", ".", 1, nil)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		runner.Start(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runner did not stop on context cancel within 2s")
	}
}
