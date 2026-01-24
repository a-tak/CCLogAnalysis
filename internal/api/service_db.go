package api

import (
	"fmt"
	"time"

	"github.com/a-tak/ccloganalysis/internal/db"
	"github.com/a-tak/ccloganalysis/internal/parser"
)

// DatabaseSessionService implements SessionService using SQLite database
type DatabaseSessionService struct {
	db     *db.DB
	parser *parser.Parser
}

// NewDatabaseSessionService creates a new DatabaseSessionService
// parser can be nil if Analyze functionality is not needed
func NewDatabaseSessionService(database *db.DB, p *parser.Parser) *DatabaseSessionService {
	service := &DatabaseSessionService{
		db:     database,
		parser: p,
	}

	// 初回起動時に自動同期
	if p != nil {
		service.autoSyncIfNeeded()
	}

	return service
}

// autoSyncIfNeeded checks if the database is empty and syncs if needed
func (s *DatabaseSessionService) autoSyncIfNeeded() {
	// プロジェクト数をチェック
	projects, err := s.db.ListProjects()
	if err != nil || len(projects) == 0 {
		// データベースが空の場合、自動的に同期
		fmt.Println("Database is empty. Starting initial sync...")
		result, err := db.SyncAll(s.db, s.parser)
		if err != nil {
			fmt.Printf("Warning: Auto-sync failed: %v\n", err)
			return
		}
		fmt.Printf("Initial sync completed: %d projects, %d sessions synced\n",
			result.ProjectsProcessed, result.SessionsSynced)
	}
}

// ListProjects returns all available projects from the database
func (s *DatabaseSessionService) ListProjects() ([]ProjectResponse, error) {
	projectRows, err := s.db.ListProjects()
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	projects := make([]ProjectResponse, 0, len(projectRows))
	for _, row := range projectRows {
		// セッション数を取得
		sessions, err := s.db.ListSessions(&row.ID, 1000, 0)
		sessionCount := 0
		if err == nil {
			sessionCount = len(sessions)
		}

		projects = append(projects, ProjectResponse{
			Name:         row.Name,
			DecodedPath:  row.DecodedPath,
			SessionCount: sessionCount,
		})
	}

	return projects, nil
}

// ListSessions returns all sessions for a project (or all projects if projectName is empty)
func (s *DatabaseSessionService) ListSessions(projectName string) ([]SessionSummary, error) {
	var projectID *int64
	var err error

	// プロジェクト名が指定されている場合、プロジェクトIDを取得
	if projectName != "" {
		project, err := s.db.GetProjectByName(projectName)
		if err != nil {
			// プロジェクトが存在しない場合は空のリストを返す
			return []SessionSummary{}, nil
		}
		projectID = &project.ID
	}

	// セッション一覧を取得（limit=1000, offset=0）
	sessionRows, err := s.db.ListSessions(projectID, 1000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	summaries := make([]SessionSummary, 0, len(sessionRows))
	for _, row := range sessionRows {
		// プロジェクト名を取得
		project, err := s.db.GetProjectByID(row.ProjectID)
		if err != nil {
			continue
		}

		totalTokens := row.TotalInputTokens + row.TotalOutputTokens

		summaries = append(summaries, SessionSummary{
			ID:          row.ID,
			ProjectName: project.Name,
			GitBranch:   row.GitBranch,
			StartTime:   row.StartTime,
			EndTime:     row.EndTime,
			TotalTokens: totalTokens,
			ErrorCount:  row.ErrorCount,
		})
	}

	return summaries, nil
}

// GetSession returns detailed session information from the database
func (s *DatabaseSessionService) GetSession(projectName, sessionID string) (*SessionDetailResponse, error) {
	// プロジェクトの存在確認
	project, err := s.db.GetProjectByName(projectName)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	// セッションを取得
	session, err := s.db.GetSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// プロジェクトパスを設定（セッションがどのプロジェクトに属するか確認するため）
	// 実際には、GetSessionで既にProjectPathが設定されているはず
	if session.ProjectPath == "" {
		session.ProjectPath = project.DecodedPath
	}

	// トークン集計を変換
	totalTokens := TokenSummaryResponse{
		InputTokens:              session.TotalTokens.InputTokens,
		OutputTokens:             session.TotalTokens.OutputTokens,
		CacheCreationInputTokens: session.TotalTokens.CacheCreationInputTokens,
		CacheReadInputTokens:     session.TotalTokens.CacheReadInputTokens,
		TotalTokens:              session.TotalTokens.InputTokens + session.TotalTokens.OutputTokens,
	}

	// モデル使用量を変換
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

	// ツール呼び出しを変換
	toolCalls := make([]ToolCallResponse, 0, len(session.ToolCalls))
	for _, tc := range session.ToolCalls {
		toolCalls = append(toolCalls, ToolCallResponse{
			Timestamp: tc.Timestamp,
			Name:      tc.Name,
			Input:     tc.Input,
			IsError:   tc.IsError,
		})
	}

	// メッセージを変換
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

	// Duration計算
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

// Analyze triggers log analysis and synchronization to database
func (s *DatabaseSessionService) Analyze(projectNames []string) (*AnalyzeResponse, error) {
	if s.parser == nil {
		return nil, fmt.Errorf("parser not configured for this service")
	}

	var result *db.SyncResult
	var err error

	if len(projectNames) == 0 {
		// 全プロジェクトを同期
		result, err = db.SyncAll(s.db, s.parser)
	} else if len(projectNames) == 1 {
		// 単一プロジェクトを同期
		result, err = db.SyncProject(s.db, s.parser, projectNames[0])
	} else {
		// 複数プロジェクトを同期（各プロジェクトを順次同期）
		result = &db.SyncResult{}
		for _, projectName := range projectNames {
			projectResult, projectErr := db.SyncProject(s.db, s.parser, projectName)
			if projectErr != nil {
				result.ErrorCount++
				continue
			}
			result.ProjectsProcessed += projectResult.ProjectsProcessed
			result.SessionsFound += projectResult.SessionsFound
			result.SessionsSynced += projectResult.SessionsSynced
			result.SessionsSkipped += projectResult.SessionsSkipped
			result.ErrorCount += projectResult.ErrorCount
		}
	}

	if err != nil {
		return &AnalyzeResponse{
			Status:  "error",
			Message: fmt.Sprintf("Sync failed: %v", err),
		}, err
	}

	return &AnalyzeResponse{
		Status:         "completed",
		SessionsFound:  result.SessionsFound,
		SessionsParsed: result.SessionsSynced,
		ErrorCount:     result.ErrorCount,
		Message:        fmt.Sprintf("Synced %d sessions from %d projects", result.SessionsSynced, result.ProjectsProcessed),
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
