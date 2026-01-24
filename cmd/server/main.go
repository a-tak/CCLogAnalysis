package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/a-tak/ccloganalysis/internal/api"
	"github.com/a-tak/ccloganalysis/internal/db"
	"github.com/a-tak/ccloganalysis/internal/parser"
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

	// Check if database mode is enabled
	useDB := os.Getenv("USE_DB")
	var service api.SessionService
	var err error

	if useDB == "true" || useDB == "1" {
		// Database mode
		dbPath := os.Getenv("DB_PATH")
		if dbPath == "" {
			// Default database path: same directory as Claude projects
			dbPath = filepath.Join(filepath.Dir(claudeDir), "ccloganalysis.db")
		}

		database, err := db.NewDB(dbPath)
		if err != nil {
			log.Fatalf("Failed to create database: %v", err)
		}
		defer database.Close()

		// Create parser for sync functionality
		p := parser.NewParser(claudeDir)
		service = api.NewDatabaseSessionService(database, p)

		fmt.Printf("Claude Code Log Analysis Server (Database Mode)\n")
		fmt.Printf("================================================\n")
		fmt.Printf("Claude projects directory: %s\n", claudeDir)
		fmt.Printf("Database path: %s\n", dbPath)
		fmt.Printf("Server starting on http://localhost:%s\n", port)
	} else {
		// File-based mode (default)
		service, err = api.NewDefaultSessionService(claudeDir)
		if err != nil {
			log.Fatalf("Failed to create service: %v", err)
		}

		fmt.Printf("Claude Code Log Analysis Server (File-based Mode)\n")
		fmt.Printf("==================================================\n")
		fmt.Printf("Claude projects directory: %s\n", claudeDir)
		fmt.Printf("Server starting on http://localhost:%s\n", port)
	}

	// Create handler and routes
	handler := api.NewHandler(service)
	router := handler.Routes()

	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal(err)
	}
}
