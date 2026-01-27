package hotreload

import (
	"fmt"
	"log"
)

// Manager coordinates file watching and WebSocket server
type Manager struct {
	watcher *Watcher
	server  *Server
	enabled bool
}

// NewManager creates a new hot-reload manager
func NewManager(enabled bool, watchDirs []string, patterns []string) (*Manager, error) {
	if !enabled {
		log.Println("[HotReload] Disabled")
		return &Manager{enabled: false}, nil
	}

	// Create watcher
	watcher, err := NewWatcher(patterns)
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	// Watch directories
	for _, dir := range watchDirs {
		if err := watcher.Watch(dir); err != nil {
			log.Printf("[HotReload] Failed to watch %s: %v", dir, err)
		}
	}

	// Create WebSocket server
	server := NewServer(watcher)

	log.Printf("[HotReload] Enabled - watching %d directories", len(watchDirs))

	return &Manager{
		watcher: watcher,
		server:  server,
		enabled: true,
	}, nil
}

// GetServer returns the WebSocket server (for route registration)
func (m *Manager) GetServer() *Server {
	if !m.enabled {
		return nil
	}
	return m.server
}

// IsEnabled returns whether hot-reload is enabled
func (m *Manager) IsEnabled() bool {
	return m.enabled
}

// Close shuts down the hot-reload manager
func (m *Manager) Close() error {
	if !m.enabled {
		return nil
	}

	if m.server != nil {
		if err := m.server.Close(); err != nil {
			log.Printf("[HotReload] Error closing server: %v", err)
		}
	}

	if m.watcher != nil {
		if err := m.watcher.Close(); err != nil {
			return fmt.Errorf("failed to close watcher: %w", err)
		}
	}

	log.Println("[HotReload] Shut down")
	return nil
}
