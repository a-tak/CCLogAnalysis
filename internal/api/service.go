package api

import (
	"fmt"
	"time"

	"github.com/a-tak/ccloganalysis/internal/parser"
)

// SessionService defines the interface for session operations
type SessionService interface {
	ListProjects() ([]ProjectResponse, error)
	ListSessions(projectName string) ([]SessionSummary, error)
	GetSession(projectName, sessionID string) (*SessionDetailResponse, error)
	Analyze(projectNames []string) (*AnalyzeResponse, error)
}

// DefaultSessionService implements SessionService using the parser
type DefaultSessionService struct {
	parser *parser.Parser
}

// NewDefaultSessionService creates a new DefaultSessionService
func NewDefaultSessionService(claudeDir string) (*DefaultSessionService, error) {
	p := parser.NewParser(claudeDir)
	return &DefaultSessionService{parser: p}, nil
}

// ListProjects returns all available projects
func (s *DefaultSessionService) ListProjects() ([]ProjectResponse, error) {
	projectNames, err := s.parser.ListProjects()
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	projects := make([]ProjectResponse, 0, len(projectNames))
	for _, name := range projectNames {
		sessions, err := s.parser.ListSessions(name)
		sessionCount := 0
		if err == nil {
			sessionCount = len(sessions)
		}

		projects = append(projects, ProjectResponse{
			Name:         name,
			DecodedPath:  parser.DecodeProjectPath(name),
			SessionCount: sessionCount,
		})
	}

	return projects, nil
}

// ListSessions returns all sessions for a project
func (s *DefaultSessionService) ListSessions(projectName string) ([]SessionSummary, error) {
	var sessionIDs []string
	var err error

	if projectName == "" {
		// List sessions from all projects
		projectNames, err := s.parser.ListProjects()
		if err != nil {
			return nil, fmt.Errorf("failed to list projects: %w", err)
		}
		for _, pName := range projectNames {
			sessions, err := s.parser.ListSessions(pName)
			if err != nil {
				continue
			}
			for _, sid := range sessions {
				sessionIDs = append(sessionIDs, pName+"/"+sid)
			}
		}
	} else {
		sessionIDs, err = s.parser.ListSessions(projectName)
		if err != nil {
			return nil, fmt.Errorf("failed to list sessions: %w", err)
		}
		// Prepend project name
		for i, sid := range sessionIDs {
			sessionIDs[i] = projectName + "/" + sid
		}
	}

	summaries := make([]SessionSummary, 0, len(sessionIDs))
	for _, fullID := range sessionIDs {
		// Parse project/session format
		var pName, sID string
		for i := len(fullID) - 1; i >= 0; i-- {
			if fullID[i] == '/' {
				pName = fullID[:i]
				sID = fullID[i+1:]
				break
			}
		}

		session, err := s.parser.ParseSession(pName, sID)
		if err != nil {
			continue
		}

		totalTokens := session.TotalTokens.InputTokens + session.TotalTokens.OutputTokens

		summaries = append(summaries, SessionSummary{
			ID:          sID,
			ProjectName: pName,
			GitBranch:   session.GitBranch,
			StartTime:   session.StartTime,
			EndTime:     session.EndTime,
			TotalTokens: totalTokens,
			ErrorCount:  session.ErrorCount,
		})
	}

	return summaries, nil
}

// GetSession returns detailed session information
func (s *DefaultSessionService) GetSession(projectName, sessionID string) (*SessionDetailResponse, error) {
	session, err := s.parser.ParseSession(projectName, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse session: %w", err)
	}

	// Convert token summary
	totalTokens := TokenSummaryResponse{
		InputTokens:              session.TotalTokens.InputTokens,
		OutputTokens:             session.TotalTokens.OutputTokens,
		CacheCreationInputTokens: session.TotalTokens.CacheCreationInputTokens,
		CacheReadInputTokens:     session.TotalTokens.CacheReadInputTokens,
		TotalTokens:              session.TotalTokens.InputTokens + session.TotalTokens.OutputTokens,
	}

	// Convert model usage
	modelUsage := make([]ModelUsageResponse, 0, len(session.ModelUsage))
	for model, usage := range session.ModelUsage {
		modelUsage = append(modelUsage, ModelUsageResponse{
			Model: model,
			Tokens: TokenSummaryResponse{
				InputTokens:              usage.InputTokens,
				OutputTokens:             usage.OutputTokens,
				CacheCreationInputTokens: usage.CacheCreationInputTokens,
				CacheReadInputTokens:     usage.CacheReadInputTokens,
				TotalTokens:              usage.InputTokens + usage.OutputTokens,
			},
		})
	}

	// Convert tool calls
	toolCalls := make([]ToolCallResponse, 0, len(session.ToolCalls))
	for _, tc := range session.ToolCalls {
		toolCalls = append(toolCalls, ToolCallResponse{
			Timestamp: tc.Timestamp,
			Name:      tc.Name,
			Input:     tc.Input,
			IsError:   tc.IsError,
		})
	}

	// Convert messages
	messages := make([]MessageResponse, 0, len(session.Entries))
	for _, entry := range session.Entries {
		if entry.Type == "user" || entry.Type == "assistant" {
			var content interface{}
			var model string
			if entry.Message != nil {
				content = entry.Message.Content
				model = entry.Message.Model
			}
			messages = append(messages, MessageResponse{
				Type:      entry.Type,
				Timestamp: entry.Timestamp,
				Model:     model,
				Content:   content,
			})
		}
	}

	// Calculate duration
	duration := session.EndTime.Sub(session.StartTime)

	return &SessionDetailResponse{
		ID:          session.ID,
		ProjectName: projectName,
		ProjectPath: session.ProjectPath,
		GitBranch:   session.GitBranch,
		StartTime:   session.StartTime,
		EndTime:     session.EndTime,
		Duration:    formatDuration(duration),
		TotalTokens: totalTokens,
		ModelUsage:  modelUsage,
		ToolCalls:   toolCalls,
		Messages:    messages,
		ErrorCount:  session.ErrorCount,
	}, nil
}

// Analyze triggers log analysis for specified projects
func (s *DefaultSessionService) Analyze(projectNames []string) (*AnalyzeResponse, error) {
	var projects []string
	var err error

	if len(projectNames) == 0 {
		projects, err = s.parser.ListProjects()
		if err != nil {
			return nil, fmt.Errorf("failed to list projects: %w", err)
		}
	} else {
		projects = projectNames
	}

	sessionsFound := 0
	sessionsParsed := 0
	errorCount := 0

	for _, pName := range projects {
		sessions, err := s.parser.ListSessions(pName)
		if err != nil {
			errorCount++
			continue
		}
		sessionsFound += len(sessions)

		for _, sID := range sessions {
			_, err := s.parser.ParseSession(pName, sID)
			if err != nil {
				errorCount++
				continue
			}
			sessionsParsed++
		}
	}

	return &AnalyzeResponse{
		Status:         "completed",
		SessionsFound:  sessionsFound,
		SessionsParsed: sessionsParsed,
		ErrorCount:     errorCount,
	}, nil
}

// formatDuration formats a duration as a human-readable string
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}
