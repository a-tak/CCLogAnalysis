package parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Parser handles JSONL log file parsing
type Parser struct {
	claudeDir string
}

// SessionFileInfo holds session ID and file modification time
type SessionFileInfo struct {
	SessionID string
	ModTime   time.Time
}

// NewParser creates a new Parser instance
func NewParser(claudeDir string) *Parser {
	return &Parser{
		claudeDir: claudeDir,
	}
}

// GetDefaultClaudeDir returns the default Claude projects directory
func GetDefaultClaudeDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".claude", "projects"), nil
}

// ListProjects returns all project directories
func (p *Parser) ListProjects() ([]string, error) {
	entries, err := os.ReadDir(p.claudeDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read claude directory: %w", err)
	}

	var projects []string
	for _, entry := range entries {
		if entry.IsDir() {
			projects = append(projects, entry.Name())
		}
	}
	return projects, nil
}

// GetProjectDir returns the absolute path to a project directory
func (p *Parser) GetProjectDir(projectName string) (string, error) {
	projectDir := filepath.Join(p.claudeDir, projectName)

	// ディレクトリが存在するか確認
	info, err := os.Stat(projectDir)
	if err != nil {
		return "", fmt.Errorf("project directory does not exist: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("project path is not a directory: %s", projectDir)
	}

	return projectDir, nil
}

// ListSessions returns all session files in a project
func (p *Parser) ListSessions(projectName string) ([]string, error) {
	projectDir := filepath.Join(p.claudeDir, projectName)
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read project directory: %w", err)
	}

	var sessions []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".jsonl") {
			sessions = append(sessions, strings.TrimSuffix(entry.Name(), ".jsonl"))
		}
	}
	return sessions, nil
}

// ListSessionsWithModTime returns session IDs with file modification times
func (p *Parser) ListSessionsWithModTime(projectName string) ([]SessionFileInfo, error) {
	projectDir := filepath.Join(p.claudeDir, projectName)
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read project directory: %w", err)
	}

	var sessions []SessionFileInfo
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".jsonl") {
			info, err := entry.Info()
			if err != nil {
				return nil, fmt.Errorf("failed to get file info for %s: %w", entry.Name(), err)
			}

			sessions = append(sessions, SessionFileInfo{
				SessionID: strings.TrimSuffix(entry.Name(), ".jsonl"),
				ModTime:   info.ModTime(),
			})
		}
	}
	return sessions, nil
}

// ParseSession parses a single session file
func (p *Parser) ParseSession(projectName, sessionID string) (*Session, error) {
	filePath := filepath.Join(p.claudeDir, projectName, sessionID+".jsonl")
	return p.ParseFile(filePath)
}

// ParseFile parses a JSONL file and returns a Session
func (p *Parser) ParseFile(filePath string) (*Session, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	session := &Session{
		ModelUsage: make(map[string]TokenSummary),
	}

	scanner := bufio.NewScanner(file)
	// Increase buffer size for large lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024) // 10MB max line size

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if line == "" {
			continue
		}

		var entry LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			// Log warning but continue parsing
			fmt.Printf("Warning: failed to parse line %d: %v\n", lineNum, err)
			continue
		}

		// Set session info from first entry
		if session.ID == "" {
			session.ID = entry.SessionID
			session.ProjectPath = entry.Cwd
			session.GitBranch = entry.GitBranch
			session.StartTime = entry.Timestamp
		}
		session.EndTime = entry.Timestamp

		// Aggregate token usage for assistant messages
		if entry.Type == "assistant" && entry.Message != nil && entry.Message.Usage != nil {
			usage := entry.Message.Usage
			model := entry.Message.Model

			// Update total tokens
			session.TotalTokens.InputTokens += usage.InputTokens
			session.TotalTokens.OutputTokens += usage.OutputTokens
			session.TotalTokens.CacheCreationInputTokens += usage.CacheCreationInputTokens
			session.TotalTokens.CacheReadInputTokens += usage.CacheReadInputTokens

			// Update model-specific tokens
			modelSummary := session.ModelUsage[model]
			modelSummary.InputTokens += usage.InputTokens
			modelSummary.OutputTokens += usage.OutputTokens
			modelSummary.CacheCreationInputTokens += usage.CacheCreationInputTokens
			modelSummary.CacheReadInputTokens += usage.CacheReadInputTokens
			session.ModelUsage[model] = modelSummary

			// Extract tool calls
			for _, content := range entry.Message.Content {
				if content.Type == "tool_use" {
					session.ToolCalls = append(session.ToolCalls, ToolCall{
						Timestamp: entry.Timestamp,
						Name:      content.Name,
						Input:     content.Input,
					})
				}
			}
		}

		// Track tool results and errors
		if entry.Type == "user" && entry.Message != nil {
			for _, content := range entry.Message.Content {
				if content.Type == "tool_result" && content.IsError {
					session.ErrorCount++
				}
			}
		}

		session.Entries = append(session.Entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return session, nil
}

// GetProjectWorkingDirectory returns the actual working directory for a project
// by parsing available sessions and extracting the cwd field from the first session that has it
func (p *Parser) GetProjectWorkingDirectory(projectName string) (string, error) {
	// セッション一覧を取得
	sessions, err := p.ListSessions(projectName)
	if err != nil {
		return "", fmt.Errorf("failed to list sessions: %w", err)
	}

	// セッションが存在しない場合はエラー
	if len(sessions) == 0 {
		return "", fmt.Errorf("no sessions found for project")
	}

	// 複数のセッションをループで試す（cwdフィールドが見つかるまで）
	for _, sessionID := range sessions {
		session, err := p.ParseSession(projectName, sessionID)
		if err != nil {
			// パースエラーは警告のみで次のセッションを試す
			fmt.Printf("Warning: failed to parse session %s: %v\n", sessionID, err)
			continue
		}

		// ProjectPath（cwd）が見つかったら返す
		if session.ProjectPath != "" {
			return session.ProjectPath, nil
		}
	}

	// すべてのセッションでcwdが見つからなかった
	return "", fmt.Errorf("working directory not found in any session")
}

// DecodeProjectPath decodes an encoded project folder name
// Example: "C--Users-{username}--my-project" -> "C:/Users/{username}/my-project"
func DecodeProjectPath(encodedName string) string {
	// Replace double dash with path separator, single dash with nothing special
	// This is a simplified version - actual encoding may vary
	result := encodedName
	result = strings.ReplaceAll(result, "--", "/")
	return result
}
