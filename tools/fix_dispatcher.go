//go:build ignore

package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	data, err := os.ReadFile("internal/dispatch/dispatcher.go")
	if err != nil {
		fmt.Println("Error reading file:", err)
		os.Exit(1)
	}
	content := string(data)

	// Check if already modified
	if strings.Contains(content, "SetLifecycleContext") {
		fmt.Println("File already contains SetLifecycleContext")
		os.Exit(0)
	}

	// Add setter methods after SetReadinessMode
	marker := "// processCommitQueue processes commit requests sequentially"
	if !strings.Contains(content, marker) {
		fmt.Println("Marker not found:", marker)
		os.Exit(1)
	}

	newMethods := `// SetLifecycleContext sets the dispatcher's lifecycle context for graceful shutdown.
// Task goroutines derive their context from this, enabling cancellation propagation
// when Loom is shutting down.
func (d *Dispatcher) SetLifecycleContext(ctx context.Context) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.lifecycleCtx = ctx
}

// SetTaskTimeout sets the maximum duration for a single task execution.
// If not set, defaults to 30 minutes.
func (d *Dispatcher) SetTaskTimeout(timeout time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.taskTimeout = timeout
}

// DefaultTaskTimeout is the default maximum duration for task execution.
const DefaultTaskTimeout = 30 * time.Minute

`

	content = strings.Replace(content, marker, newMethods+marker, 1)

	// Fix the goroutine context - find "taskCtx := ctx" and replace it
	oldCtx := "\t\ttaskCtx := ctx"
	if !strings.Contains(content, oldCtx) {
		fmt.Println("taskCtx pattern not found")
		os.Exit(1)
	}

	newCtx := `		// Use lifecycle context with task timeout instead of request context.
		// This enables graceful shutdown cancellation and prevents goroutine leaks.
		d.mu.RLock()
		baseCtx := d.lifecycleCtx
		timeout := d.taskTimeout
		d.mu.RUnlock()
		if baseCtx == nil {
			baseCtx = context.Background()
		}
		if timeout == 0 {
			timeout = DefaultTaskTimeout
		}
		taskCtx, cancel := context.WithTimeout(baseCtx, timeout)
		defer cancel()`

	content = strings.Replace(content, oldCtx, newCtx, 1)

	err = os.WriteFile("internal/dispatch/dispatcher.go", []byte(content), 0644)
	if err != nil {
		fmt.Println("Error writing file:", err)
		os.Exit(1)
	}
	fmt.Println("File modified successfully")
}
