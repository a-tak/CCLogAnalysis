package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/a-tak/ccloganalysis/internal/api"
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

	// Create service
	service, err := api.NewDefaultSessionService(claudeDir)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	// Create handler and routes
	handler := api.NewHandler(service)
	router := handler.Routes()

	fmt.Printf("Claude Code Log Analysis Server\n")
	fmt.Printf("================================\n")
	fmt.Printf("Claude projects directory: %s\n", claudeDir)
	fmt.Printf("Server starting on http://localhost:%s\n", port)

	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal(err)
	}
}
