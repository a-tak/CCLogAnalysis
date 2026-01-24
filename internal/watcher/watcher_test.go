package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/a-tak/ccloganalysis/internal/db"
	"github.com/a-tak/ccloganalysis/internal/parser"
)

func TestNewFileWatcher(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	p := parser.NewParser(setupTestProjectsDir(t))
	config := WatcherConfig{
		Enabled:  true,
		Interval: 10 * time.Second,
		Debounce: 5 * time.Second,
	}

	watcher := NewFileWatcher(database, p, config)

	if watcher == nil {
		t.Fatal("Expected NewFileWatcher to return non-nil watcher")
	}
	if watcher.db != database {
		t.Error("Expected watcher.db to be set")
	}
	if watcher.parser != p {
		t.Error("Expected watcher.parser to be set")
	}
	if watcher.interval != config.Interval {
		t.Errorf("Expected interval to be %v, got %v", config.Interval, watcher.interval)
	}
	if watcher.debounce != config.Debounce {
		t.Errorf("Expected debounce to be %v, got %v", config.Debounce, watcher.debounce)
	}
}

func TestNewFileWatcher_NilDB(t *testing.T) {
	p := parser.NewParser(setupTestProjectsDir(t))
	config := WatcherConfig{
		Enabled:  true,
		Interval: 10 * time.Second,
		Debounce: 5 * time.Second,
	}

	watcher := NewFileWatcher(nil, p, config)

	if watcher != nil {
		t.Error("Expected NewFileWatcher to return nil for nil database")
	}
}

func TestNewFileWatcher_NilParser(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	config := WatcherConfig{
		Enabled:  true,
		Interval: 10 * time.Second,
		Debounce: 5 * time.Second,
	}

	watcher := NewFileWatcher(database, nil, config)

	if watcher != nil {
		t.Error("Expected NewFileWatcher to return nil for nil parser")
	}
}

func TestFileWatcher_Start(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	p := parser.NewParser(setupTestProjectsDir(t))
	config := WatcherConfig{
		Enabled:  true,
		Interval: 100 * time.Millisecond, // Short interval for testing
		Debounce: 50 * time.Millisecond,
	}

	watcher := NewFileWatcher(database, p, config)
	watcher.Start()

	// Give it a moment to start
	time.Sleep(50 * time.Millisecond)

	// Verify it's running (channels should be initialized)
	if watcher.stopCh == nil {
		t.Error("Expected stopCh to be initialized after Start")
	}
	if watcher.doneCh == nil {
		t.Error("Expected doneCh to be initialized after Start")
	}

	watcher.Stop()
}

func TestFileWatcher_Stop(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	p := parser.NewParser(setupTestProjectsDir(t))
	config := WatcherConfig{
		Enabled:  true,
		Interval: 100 * time.Millisecond,
		Debounce: 50 * time.Millisecond,
	}

	watcher := NewFileWatcher(database, p, config)
	watcher.Start()

	time.Sleep(50 * time.Millisecond)

	watcher.Stop()

	// Verify goroutine has stopped (doneCh should be closed)
	select {
	case <-watcher.doneCh:
		// Good - channel is closed
	case <-time.After(1 * time.Second):
		t.Error("Expected watcher to stop within 1 second")
	}
}

func TestFileWatcher_StartIdempotent(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	p := parser.NewParser(setupTestProjectsDir(t))
	config := WatcherConfig{
		Enabled:  true,
		Interval: 100 * time.Millisecond,
		Debounce: 50 * time.Millisecond,
	}

	watcher := NewFileWatcher(database, p, config)

	// Start multiple times
	watcher.Start()
	watcher.Start()
	watcher.Start()

	time.Sleep(50 * time.Millisecond)

	// Should still work
	watcher.Stop()

	select {
	case <-watcher.doneCh:
		// Good
	case <-time.After(1 * time.Second):
		t.Error("Expected watcher to stop")
	}
}

func TestFileWatcher_StopIdempotent(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	p := parser.NewParser(setupTestProjectsDir(t))
	config := WatcherConfig{
		Enabled:  true,
		Interval: 100 * time.Millisecond,
		Debounce: 50 * time.Millisecond,
	}

	watcher := NewFileWatcher(database, p, config)
	watcher.Start()

	time.Sleep(50 * time.Millisecond)

	// Stop multiple times (should not panic)
	watcher.Stop()
	watcher.Stop()
	watcher.Stop()
}

// Helper functions

func setupTestDB(t *testing.T) (*db.DB, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	database, err := db.NewDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	cleanup := func() {
		database.Close()
		os.RemoveAll(tmpDir)
	}

	return database, cleanup
}

func setupTestProjectsDir(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "test-project")
	err := os.MkdirAll(projectDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test project dir: %v", err)
	}

	return tmpDir
}

// Sprint 2: Polling and sync tests

func TestFileWatcher_TriggerSync(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	projectsDir := setupTestProjectsDir(t)
	p := parser.NewParser(projectsDir)

	// Create a test session file
	createTestSession(t, projectsDir, "test-project", "session-1")

	config := WatcherConfig{
		Enabled:  true,
		Interval: 100 * time.Millisecond,
		Debounce: 50 * time.Millisecond,
	}

	watcher := NewFileWatcher(database, p, config)

	// Manually trigger sync
	err := watcher.triggerSync()
	if err != nil {
		t.Errorf("Expected triggerSync to succeed, got error: %v", err)
	}

	// Verify session was synced to database
	projects, err := database.ListProjects()
	if err != nil {
		t.Fatalf("Failed to list projects: %v", err)
	}

	if len(projects) == 0 {
		t.Error("Expected at least one project to be synced")
	}
}

func TestFileWatcher_ShouldSync_Debounce(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	p := parser.NewParser(setupTestProjectsDir(t))
	config := WatcherConfig{
		Enabled:  true,
		Interval: 100 * time.Millisecond,
		Debounce: 200 * time.Millisecond,
	}

	watcher := NewFileWatcher(database, p, config)

	// First sync should be allowed (lastSync is zero)
	if !watcher.shouldSync() {
		t.Error("Expected first shouldSync to return true")
	}

	// Update lastSync to now
	watcher.mu.Lock()
	watcher.lastSync = time.Now()
	watcher.mu.Unlock()

	// Immediate second check should be blocked by debounce
	if watcher.shouldSync() {
		t.Error("Expected shouldSync to return false (debounce)")
	}

	// Wait for debounce period
	time.Sleep(250 * time.Millisecond)

	// Should be allowed now
	if !watcher.shouldSync() {
		t.Error("Expected shouldSync to return true after debounce period")
	}
}

func TestFileWatcher_WatchLoop_PeriodicExecution(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	projectsDir := setupTestProjectsDir(t)
	p := parser.NewParser(projectsDir)

	// Create initial session
	createTestSession(t, projectsDir, "test-project", "session-1")

	config := WatcherConfig{
		Enabled:  true,
		Interval: 200 * time.Millisecond,
		Debounce: 50 * time.Millisecond,
	}

	watcher := NewFileWatcher(database, p, config)
	watcher.Start()
	defer watcher.Stop()

	// Wait for at least 2 polling cycles
	time.Sleep(500 * time.Millisecond)

	// Add a new session during watch
	createTestSession(t, projectsDir, "test-project", "session-2")

	// Wait for another polling cycle
	time.Sleep(300 * time.Millisecond)

	// Verify both sessions were synced
	projects, err := database.ListProjects()
	if err != nil {
		t.Fatalf("Failed to list projects: %v", err)
	}

	if len(projects) == 0 {
		t.Error("Expected at least one project")
	}

	// Get sessions for the project
	sessions, err := database.ListSessions(&projects[0].ID, 100, 0)
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	// Should have synced both sessions
	if len(sessions) < 2 {
		t.Errorf("Expected at least 2 sessions, got %d", len(sessions))
	}
}

// Helper to create test session files
func createTestSession(t *testing.T, projectsDir, projectName, sessionID string) {
	t.Helper()

	projectDir := filepath.Join(projectsDir, projectName)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	sessionFile := filepath.Join(projectDir, sessionID+".jsonl")
	// Create minimal valid Claude Code log format
	content := `{"sessionId":"` + sessionID + `","type":"user","message":{"role":"user","content":"test"},"uuid":"` + sessionID + `-uuid-1","timestamp":"2026-01-24T10:00:00Z"}
{"sessionId":"` + sessionID + `","type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"response"}]},"uuid":"` + sessionID + `-uuid-2","timestamp":"2026-01-24T10:00:01Z"}
`
	if err := os.WriteFile(sessionFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write session file: %v", err)
	}
}
