package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/a-tak/ccloganalysis/internal/api"
	"github.com/a-tak/ccloganalysis/internal/db"
	"github.com/a-tak/ccloganalysis/internal/parser"
	"github.com/a-tak/ccloganalysis/internal/scanner"
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

	// Create scan manager
	scanManager := scanner.NewScanManager(database, p)

	// Sync project groups from existing data before starting scan
	// This allows groups to be displayed immediately on server startup
	if err := database.SyncProjectGroups(); err != nil {
		log.Printf("Warning: Failed to sync project groups: %v", err)
	} else {
		fmt.Println("Project groups synchronized from existing data")
	}

	// Start initial sync asynchronously (unless SKIP_INITIAL_SYNC is set)
	skipInitialSync := os.Getenv("SKIP_INITIAL_SYNC") != ""
	if !skipInitialSync {
		if err := scanManager.StartInitialScan(context.Background()); err != nil {
			log.Printf("Warning: Failed to start initial scan: %v", err)
		}
		fmt.Println("Initial sync started in background...")

		// Wait for initial sync to complete before starting file watcher
		// This prevents concurrent sync operations that could cause excessive logging
		fmt.Println("Waiting for initial sync to complete...")
		scanManager.WaitForInitialScan()
		fmt.Println("Initial sync completed")
	} else {
		fmt.Println("Skipping initial sync (SKIP_INITIAL_SYNC is set)")
	}

	// Initialize file watcher (after initial sync completes)
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
	handler := api.NewHandler(service, scanManager)
	router := handler.Routes()

	// Setup HTTP server with graceful shutdown support
	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Setup graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Start HTTP server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrors <- err
		}
	}()

	// Wait for shutdown signal or server error
	select {
	case err := <-serverErrors:
		log.Fatalf("Server error: %v", err)
	case <-sigCh:
		fmt.Println("\nShutting down gracefully...")

		// Create shutdown context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Shutdown HTTP server gracefully
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}

		// Stop scan manager
		if scanManager != nil {
			scanManager.Stop()
		}

		// Stop file watcher
		if fileWatcher != nil {
			fileWatcher.Stop()
		}

		// Close database
		database.Close()

		fmt.Println("Shutdown complete")
	}
}
