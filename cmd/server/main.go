package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/a-tak/ccloganalysis/internal/api"
	"github.com/a-tak/ccloganalysis/internal/db"
	"github.com/a-tak/ccloganalysis/internal/parser"
	"github.com/a-tak/ccloganalysis/internal/watcher"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Get Claude projects directory
	claudeDir := os.Getenv("CLAUDE_PROJECTS_DIR")
	if claudeDir == "" {
		var err error
		claudeDir, err = parser.GetDefaultClaudeDir()
		if err != nil {
			log.Fatalf("Failed to get Claude directory: %v", err)
		}
	}

	// Database mode (default)
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		// Default database path: same directory as executable
		exePath, err := os.Executable()
		if err != nil {
			log.Fatalf("Failed to get executable path: %v", err)
		}
		dbPath = filepath.Join(filepath.Dir(exePath), "ccloganalysis.db")
	}

	database, err := db.NewDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}
	defer database.Close()

	// Create parser for sync functionality
	p := parser.NewParser(claudeDir)
	service := api.NewDatabaseSessionService(database, p)

	// Initialize file watcher
	watcherConfig := watcher.LoadWatcherConfig()
	var fileWatcher *watcher.FileWatcher

	if watcherConfig.Enabled && p != nil {
		fileWatcher = watcher.NewFileWatcher(database, p, watcherConfig)
		if fileWatcher != nil {
			fileWatcher.Start()
			fmt.Printf("File watcher enabled (interval: %s, debounce: %s)\n",
				watcherConfig.Interval, watcherConfig.Debounce)
		}
	}

	fmt.Printf("Claude Code Log Analysis Server\n")
	fmt.Printf("================================\n")
	fmt.Printf("Claude projects directory: %s\n", claudeDir)
	fmt.Printf("Database path: %s\n", dbPath)
	fmt.Printf("Server starting on http://localhost:%s\n", port)

	// Create handler and routes
	handler := api.NewHandler(service)
	router := handler.Routes()

	// Setup graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Start HTTP server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		if err := http.ListenAndServe(":"+port, router); err != nil && err != http.ErrServerClosed {
			serverErrors <- err
		}
	}()

	// Wait for shutdown signal or server error
	select {
	case err := <-serverErrors:
		log.Fatalf("Server error: %v", err)
	case <-sigCh:
		fmt.Println("\nShutting down gracefully...")

		// Stop file watcher
		if fileWatcher != nil {
			fileWatcher.Stop()
		}

		// Close database
		database.Close()

		fmt.Println("Shutdown complete")
	}
}
