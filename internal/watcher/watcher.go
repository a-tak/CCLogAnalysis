package watcher

import (
	"fmt"
	"sync"
	"time"

	"github.com/a-tak/ccloganalysis/internal/db"
	"github.com/a-tak/ccloganalysis/internal/parser"
)

// FileWatcher watches filesystem for new JSONL files
type FileWatcher struct {
	db       *db.DB
	parser   *parser.Parser
	interval time.Duration
	debounce time.Duration

	stopCh   chan struct{}
	doneCh   chan struct{}
	lastSync time.Time
	mu       sync.Mutex
	running  bool
}

// NewFileWatcher creates a new file watcher
func NewFileWatcher(database *db.DB, p *parser.Parser, config WatcherConfig) *FileWatcher {
	if database == nil || p == nil {
		return nil
	}

	return &FileWatcher{
		db:       database,
		parser:   p,
		interval: config.Interval,
		debounce: config.Debounce,
	}
}

// Start starts the file watcher
func (w *FileWatcher) Start() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.running {
		// Already running
		return
	}

	w.stopCh = make(chan struct{})
	w.doneCh = make(chan struct{})
	w.running = true
	w.lastSync = time.Time{} // Zero time to allow initial sync

	go w.watchLoop()
}

// Stop stops the file watcher
func (w *FileWatcher) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.mu.Unlock()

	close(w.stopCh)

	// Wait for goroutine to finish with timeout
	select {
	case <-w.doneCh:
		// Goroutine finished normally
	case <-time.After(2 * time.Second):
		// Timeout - goroutine will exit on next ticker or stopCh
	}

	w.mu.Lock()
	w.running = false
	w.mu.Unlock()
}

// watchLoop is the main polling loop (runs in goroutine)
func (w *FileWatcher) watchLoop() {
	defer close(w.doneCh)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			// Polling tick - check for new files and sync
			if w.shouldSync() {
				_ = w.triggerSync() // Error is handled at upper layer
			}
		}
	}
}

// shouldSync checks if enough time has passed since last sync (debounce)
func (w *FileWatcher) shouldSync() bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	if now.Sub(w.lastSync) < w.debounce {
		return false
	}

	return true
}

// triggerSync executes incremental sync and updates lastSync time
func (w *FileWatcher) triggerSync() error {
	// DEBUGレベルでのみ表示（ログスパム削減のため）

	_, err := db.SyncIncremental(w.db, w.parser)
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	// Update last sync time
	w.mu.Lock()
	w.lastSync = time.Now()
	w.mu.Unlock()

	return nil
}
