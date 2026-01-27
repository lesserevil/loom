package hotreload

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors file changes and triggers reload events
type Watcher struct {
	fsWatcher *fsnotify.Watcher
	listeners []chan FileChangeEvent
	mu        sync.RWMutex
	patterns  []string
	ctx       context.Context
	cancel    context.CancelFunc
}

// FileChangeEvent represents a file change event
type FileChangeEvent struct {
	Path      string
	Operation string // "created", "modified", "deleted"
	Timestamp time.Time
}

// NewWatcher creates a new file watcher
func NewWatcher(patterns []string) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	w := &Watcher{
		fsWatcher: fsWatcher,
		listeners: make([]chan FileChangeEvent, 0),
		patterns:  patterns,
		ctx:       ctx,
		cancel:    cancel,
	}

	return w, nil
}

// Watch starts watching the specified directory
func (w *Watcher) Watch(dir string) error {
	// Add directory and subdirectories
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Watch directory
			if err := w.fsWatcher.Add(path); err != nil {
				log.Printf("[HotReload] Failed to watch %s: %v", path, err)
			} else {
				log.Printf("[HotReload] Watching %s", path)
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	// Start event loop
	go w.eventLoop()

	return nil
}

// Subscribe returns a channel that receives file change events
func (w *Watcher) Subscribe() chan FileChangeEvent {
	w.mu.Lock()
	defer w.mu.Unlock()

	ch := make(chan FileChangeEvent, 10)
	w.listeners = append(w.listeners, ch)
	return ch
}

// Unsubscribe removes a listener channel
func (w *Watcher) Unsubscribe(ch chan FileChangeEvent) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for i, listener := range w.listeners {
		if listener == ch {
			w.listeners = append(w.listeners[:i], w.listeners[i+1:]...)
			close(ch)
			break
		}
	}
}

// eventLoop processes file system events
func (w *Watcher) eventLoop() {
	debounceTimer := time.NewTimer(0)
	<-debounceTimer.C // Drain initial timer

	var pendingEvents []fsnotify.Event
	var mu sync.Mutex

	for {
		select {
		case <-w.ctx.Done():
			return

		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}

			// Filter by patterns
			if !w.matchesPattern(event.Name) {
				continue
			}

			// Debounce rapid changes (editors often write multiple times)
			mu.Lock()
			pendingEvents = append(pendingEvents, event)
			mu.Unlock()

			debounceTimer.Reset(100 * time.Millisecond)

		case <-debounceTimer.C:
			mu.Lock()
			events := pendingEvents
			pendingEvents = nil
			mu.Unlock()

			// Process debounced events
			for _, event := range events {
				w.notifyListeners(event)
			}

		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			log.Printf("[HotReload] Watcher error: %v", err)
		}
	}
}

// matchesPattern checks if a file matches any of the watch patterns
func (w *Watcher) matchesPattern(path string) bool {
	if len(w.patterns) == 0 {
		return true // No patterns = watch everything
	}

	for _, pattern := range w.patterns {
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err != nil {
			continue
		}
		if matched {
			return true
		}
	}
	return false
}

// notifyListeners sends event to all subscribed listeners
func (w *Watcher) notifyListeners(event fsnotify.Event) {
	changeEvent := FileChangeEvent{
		Path:      event.Name,
		Operation: getOperation(event.Op),
		Timestamp: time.Now(),
	}

	log.Printf("[HotReload] File changed: %s (%s)", changeEvent.Path, changeEvent.Operation)

	w.mu.RLock()
	defer w.mu.RUnlock()

	for _, listener := range w.listeners {
		select {
		case listener <- changeEvent:
		default:
			// Channel full, skip this listener
		}
	}
}

// getOperation converts fsnotify.Op to string
func getOperation(op fsnotify.Op) string {
	switch {
	case op&fsnotify.Create == fsnotify.Create:
		return "created"
	case op&fsnotify.Write == fsnotify.Write:
		return "modified"
	case op&fsnotify.Remove == fsnotify.Remove:
		return "deleted"
	case op&fsnotify.Rename == fsnotify.Rename:
		return "renamed"
	case op&fsnotify.Chmod == fsnotify.Chmod:
		return "chmod"
	default:
		return "unknown"
	}
}

// Close stops the watcher and closes all listeners
func (w *Watcher) Close() error {
	w.cancel()

	w.mu.Lock()
	for _, listener := range w.listeners {
		close(listener)
	}
	w.listeners = nil
	w.mu.Unlock()

	return w.fsWatcher.Close()
}
