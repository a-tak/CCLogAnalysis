package api

import (
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"github.com/a-tak/ccloganalysis/internal/db"
	"github.com/a-tak/ccloganalysis/internal/logger"
	"github.com/a-tak/ccloganalysis/internal/parser"
)

// DatabaseSessionService implements SessionService using SQLite database
type DatabaseSessionService struct {
	db        *db.DB
	parser    *parser.Parser
	syncError error
	logger    *logger.Logger
}

// NewDatabaseSessionService creates a new DatabaseSessionService
// parser can be nil if Analyze functionality is not needed
func NewDatabaseSessionService(database *db.DB, p *parser.Parser) *DatabaseSessionService {
	service := &DatabaseSessionService{
		db:     database,
		parser: p,
		logger: logger.New(),
	}

	// Note: Initial sync is now handled by ScanManager in main.go

	return service
}

// getProjectDisplayName returns the display name for a project
// Falls back to the encoded name if working directory cannot be retrieved
func (s *DatabaseSessionService) getProjectDisplayName(projectID int64, fallbackName string) string {
	if cwd, err := s.db.GetProjectWorkingDirectory(projectID); err == nil {
		return filepath.Base(cwd)
	}
	return fallbackName
}

// getGroupDisplayName returns the display name for a project group
// Priority: 1. gitRoot base name, 2. first project's cwd base name, 3. fallback name
func (s *DatabaseSessionService) getGroupDisplayName(gitRoot *string, projectRows []*db.ProjectRow, fallbackName string) string {
	// git_root から displayName を取得
	if gitRoot != nil && *gitRoot != "" {
		return filepath.Base(*gitRoot)
	}
	// git_root がない場合、グループに属するプロジェクトの cwd から取得を試みる
	if len(projectRows) > 0 {
		if cwd, err := s.db.GetProjectWorkingDirectory(projectRows[0].ID); err == nil {
			return filepath.Base(cwd)
		}
	}
	return fallbackName
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
			DisplayName:  s.getProjectDisplayName(row.ID, row.Name),
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
			// プロジェクトが存在しない場合は警告を出力して空のリストを返す
			s.logger.WarnWithContext("Project not found in database", map[string]interface{}{
				"project": projectName,
			})
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
			ID:               row.ID,
			ProjectName:      project.Name,
			GitBranch:        row.GitBranch,
			StartTime:        row.StartTime,
			EndTime:          row.EndTime,
			TotalTokens:      totalTokens,
			ErrorCount:       row.ErrorCount,
			FirstUserMessage: row.FirstUserMessage,
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
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", projectName, projectErr))
				continue
			}
			result.ProjectsProcessed += projectResult.ProjectsProcessed
			result.SessionsFound += projectResult.SessionsFound
			result.SessionsSynced += projectResult.SessionsSynced
			result.SessionsSkipped += projectResult.SessionsSkipped
			result.ErrorCount += projectResult.ErrorCount
			result.Errors = append(result.Errors, projectResult.Errors...)
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

// GetProjectStats returns project-level statistics
func (s *DatabaseSessionService) GetProjectStats(projectName string) (*ProjectStatsResponse, error) {
	// プロジェクトの存在確認
	project, err := s.db.GetProjectByName(projectName)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	// プロジェクト統計を取得
	stats, err := s.db.GetProjectStats(project.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project stats: %w", err)
	}

	return &ProjectStatsResponse{
		TotalSessions:            stats.TotalSessions,
		TotalInputTokens:         stats.TotalInputTokens,
		TotalOutputTokens:        stats.TotalOutputTokens,
		TotalCacheCreationTokens: stats.TotalCacheCreationTokens,
		TotalCacheReadTokens:     stats.TotalCacheReadTokens,
		TotalTokens:              stats.TotalInputTokens + stats.TotalOutputTokens,
		AvgTokens:                stats.AvgTokens,
		FirstSession:             stats.FirstSession,
		LastSession:              stats.LastSession,
		ErrorRate:                stats.ErrorRate,
	}, nil
}

// GetProjectTimeline returns time-series statistics for a project
func (s *DatabaseSessionService) GetProjectTimeline(projectName, period string, limit int) (*TimeSeriesResponse, error) {
	// プロジェクトの存在確認
	project, err := s.db.GetProjectByName(projectName)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	// periodのデフォルト値
	if period == "" {
		period = "day"
	}

	// limitのデフォルト値
	if limit <= 0 {
		limit = 30
	}

	// 時系列統計を取得
	timeSeriesStats, err := s.db.GetTimeSeriesStats(project.ID, period, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get timeline stats: %w", err)
	}

	return &TimeSeriesResponse{
		Period: period,
		Data:   convertToTimeSeriesDataPoints(timeSeriesStats),
	}, nil
}

// ListProjectGroups returns all project groups
func (s *DatabaseSessionService) ListProjectGroups() ([]ProjectGroupResponse, error) {
	// 1. 全グループを取得
	groupRows, err := s.db.ListProjectGroups()
	if err != nil {
		return nil, fmt.Errorf("failed to list project groups: %w", err)
	}

	// 2. 除外対象のグループIDを取得（ワークツリーグループのメンバー）
	hiddenGroupIDs, err := s.db.GetStandaloneGroupsInWorktreeGroups()
	if err != nil {
		return nil, fmt.Errorf("failed to get hidden group ids: %w", err)
	}

	// 3. 除外対象のグループIDを取得（名前にworktreeを含む）
	worktreeNameGroupIDs, err := s.db.GetStandaloneGroupsWithWorktreeName()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree name group ids: %w", err)
	}

	// 4. 除外対象をmapに変換（O(1)検索用）
	hiddenMap := make(map[int64]bool)
	for _, id := range hiddenGroupIDs {
		hiddenMap[id] = true
	}
	for _, id := range worktreeNameGroupIDs {
		hiddenMap[id] = true
	}

	// 5. フィルタリング
	groups := make([]ProjectGroupResponse, 0, len(groupRows))
	for _, row := range groupRows {
		// 除外対象はスキップ
		if hiddenMap[row.ID] {
			continue
		}

		projects, err := s.db.GetProjectsByGroupID(row.ID)
		if err != nil {
			s.logger.WarnWithContext("Failed to get projects for group", map[string]interface{}{
				"group_id": row.ID,
				"error":    err.Error(),
			})
			projects = []*db.ProjectRow{}
		}
		displayName := s.getGroupDisplayName(row.GitRoot, projects, row.Name)

		groups = append(groups, ProjectGroupResponse{
			ID:          row.ID,
			Name:        row.Name,
			DisplayName: displayName,
			GitRoot:     row.GitRoot,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		})
	}

	// 6. ソート: git_rootが設定されているグループを先頭に、その後は名前順
	sort.Slice(groups, func(i, j int) bool {
		// git_rootの有無で優先順位を決定
		iHasGitRoot := groups[i].GitRoot != nil
		jHasGitRoot := groups[j].GitRoot != nil

		if iHasGitRoot != jHasGitRoot {
			// git_rootがあるグループを先頭に
			return iHasGitRoot
		}

		// git_rootの有無が同じ場合は、名前順でソート
		return groups[i].Name < groups[j].Name
	})

	return groups, nil
}

// GetProjectGroup returns detailed project group information with member projects
func (s *DatabaseSessionService) GetProjectGroup(groupID int64) (*ProjectGroupDetailResponse, error) {
	// グループ基本情報を取得
	group, err := s.db.GetProjectGroupByID(groupID)
	if err != nil {
		return nil, fmt.Errorf("group not found: %w", err)
	}

	// グループ内のプロジェクトを取得
	projectRows, err := s.db.GetProjectsByGroupID(groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get projects in group: %w", err)
	}

	// プロジェクトレスポンスに変換
	projects := make([]ProjectResponse, 0, len(projectRows))
	for _, row := range projectRows {
		// セッション数を取得
		sessionCount, err := s.db.CountSessions(row.ID)
		if err != nil {
			s.logger.WarnWithContext("Failed to count sessions for project", map[string]interface{}{
				"project": row.Name,
				"error":   err.Error(),
			})
			sessionCount = 0
		}

		projects = append(projects, ProjectResponse{
			Name:         row.Name,
			DecodedPath:  row.DecodedPath,
			DisplayName:  s.getProjectDisplayName(row.ID, row.Name),
			SessionCount: sessionCount,
		})
	}

	displayName := s.getGroupDisplayName(group.GitRoot, projectRows, group.Name)

	return &ProjectGroupDetailResponse{
		ID:          group.ID,
		Name:        group.Name,
		DisplayName: displayName,
		GitRoot:     group.GitRoot,
		CreatedAt:   group.CreatedAt,
		UpdatedAt:   group.UpdatedAt,
		Projects:    projects,
	}, nil
}

// GetProjectGroupStats returns statistics for a project group
func (s *DatabaseSessionService) GetProjectGroupStats(groupID int64) (*ProjectGroupStatsResponse, error) {
	// グループの存在確認
	_, err := s.db.GetProjectGroupByID(groupID)
	if err != nil {
		return nil, fmt.Errorf("group not found: %w", err)
	}

	// グループ統計を取得
	stats, err := s.db.GetGroupStats(groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group stats: %w", err)
	}

	return &ProjectGroupStatsResponse{
		TotalProjects:            stats.TotalProjects,
		TotalSessions:            stats.TotalSessions,
		TotalInputTokens:         stats.TotalInputTokens,
		TotalOutputTokens:        stats.TotalOutputTokens,
		TotalCacheCreationTokens: stats.TotalCacheCreationTokens,
		TotalCacheReadTokens:     stats.TotalCacheReadTokens,
		AvgTokens:                stats.AvgTokens,
		FirstSession:             stats.FirstSession,
		LastSession:              stats.LastSession,
		ErrorRate:                stats.ErrorRate,
	}, nil
}

// GetProjectGroupTimeline returns time-series statistics for a project group
func (s *DatabaseSessionService) GetProjectGroupTimeline(groupID int64, period string, limit int) (*TimeSeriesResponse, error) {
	// グループの存在確認
	_, err := s.db.GetProjectGroupByID(groupID)
	if err != nil {
		return nil, fmt.Errorf("group not found: %w", err)
	}

	// periodのデフォルト値
	if period == "" {
		period = "day"
	}

	// limitのデフォルト値
	if limit <= 0 {
		limit = 30
	}

	// 時系列統計を取得
	timeSeriesStats, err := s.db.GetGroupTimeSeriesStats(groupID, period, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get group timeline stats: %w", err)
	}

	return &TimeSeriesResponse{
		Period: period,
		Data:   convertToTimeSeriesDataPoints(timeSeriesStats),
	}, nil
}

// GetTotalStats returns total statistics across all projects
func (s *DatabaseSessionService) GetTotalStats() (*TotalStatsResponse, error) {
	stats, err := s.db.GetTotalStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get total stats: %w", err)
	}

	return &TotalStatsResponse{
		TotalGroups:              stats.TotalGroups,
		TotalProjects:            stats.TotalProjects,
		TotalSessions:            stats.TotalSessions,
		TotalInputTokens:         stats.TotalInputTokens,
		TotalOutputTokens:        stats.TotalOutputTokens,
		TotalCacheCreationTokens: stats.TotalCacheCreationTokens,
		TotalCacheReadTokens:     stats.TotalCacheReadTokens,
		TotalTokens:              stats.TotalInputTokens + stats.TotalOutputTokens,
		AvgTokens:                stats.AvgTokens,
		FirstSession:             stats.FirstSession,
		LastSession:              stats.LastSession,
		ErrorRate:                stats.ErrorRate,
	}, nil
}

// GetTotalTimeline returns time-series statistics across all projects
func (s *DatabaseSessionService) GetTotalTimeline(period string, limit int) (*TimeSeriesResponse, error) {
	// periodのデフォルト値
	if period == "" {
		period = "day"
	}

	// limitのデフォルト値
	if limit <= 0 {
		limit = 30
	}

	// 時系列統計を取得
	timeSeriesStats, err := s.db.GetTotalTimeSeriesStats(period, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get total timeline stats: %w", err)
	}

	return &TimeSeriesResponse{
		Period: period,
		Data:   convertToTimeSeriesDataPoints(timeSeriesStats),
	}, nil
}

// GetDailyStats returns group-wise statistics for a specific date
func (s *DatabaseSessionService) GetDailyStats(date string) (*DailyStatsResponse, error) {
	stats, err := s.db.GetDailyGroupStats(date)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily stats: %w", err)
	}

	groups := make([]DailyGroupStatsResponse, 0, len(stats))
	for _, g := range stats {
		groups = append(groups, DailyGroupStatsResponse{
			GroupID:                  g.GroupID,
			GroupName:                g.GroupName,
			SessionCount:             g.SessionCount,
			TotalInputTokens:         g.TotalInputTokens,
			TotalOutputTokens:        g.TotalOutputTokens,
			TotalCacheCreationTokens: g.TotalCacheCreationTokens,
			TotalCacheReadTokens:     g.TotalCacheReadTokens,
			TotalTokens:              g.TotalInputTokens + g.TotalOutputTokens,
		})
	}

	return &DailyStatsResponse{
		Date:   date,
		Groups: groups,
	}, nil
}

// GetGroupDailyStats retrieves project-wise statistics for a group on a specific date
func (s *DatabaseSessionService) GetGroupDailyStats(groupID int64, date string) (*GroupDailyStatsResponse, error) {
	// DB層からプロジェクト別統計を取得
	stats, err := s.db.GetGroupDailyProjectStats(groupID, date)
	if err != nil {
		return nil, fmt.Errorf("failed to get group daily project stats: %w", err)
	}

	// APIレスポンスに変換
	projects := make([]DailyProjectStatsResponse, 0, len(stats))
	for _, p := range stats {
		projects = append(projects, DailyProjectStatsResponse{
			ProjectID:                p.ProjectID,
			ProjectName:              p.ProjectName,
			SessionCount:             p.SessionCount,
			TotalInputTokens:         p.TotalInputTokens,
			TotalOutputTokens:        p.TotalOutputTokens,
			TotalCacheCreationTokens: p.TotalCacheCreationTokens,
			TotalCacheReadTokens:     p.TotalCacheReadTokens,
			TotalTokens:              p.TotalTokens,
		})
	}

	return &GroupDailyStatsResponse{
		Date:     date,
		Projects: projects,
	}, nil
}

// GetProjectDailyStats retrieves session-wise statistics for a project on a specific date
func (s *DatabaseSessionService) GetProjectDailyStats(projectName string, date string) (*ProjectDailyStatsResponse, error) {
	// プロジェクト名からプロジェクトIDを取得
	project, err := s.db.GetProjectByName(projectName)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// DB層からセッション一覧を取得
	sessionRows, err := s.db.GetProjectDailySessions(project.ID, date)
	if err != nil {
		return nil, fmt.Errorf("failed to get project daily sessions: %w", err)
	}

	// APIレスポンスに変換
	sessions := make([]DailySessionResponse, 0, len(sessionRows))
	for _, sr := range sessionRows {
		duration := sr.EndTime.Sub(sr.StartTime)
		sessions = append(sessions, DailySessionResponse{
			ID:                       sr.ID,
			GitBranch:                sr.GitBranch,
			StartTime:                sr.StartTime,
			EndTime:                  sr.EndTime,
			Duration:                 formatDuration(duration),
			TotalInputTokens:         sr.TotalInputTokens,
			TotalOutputTokens:        sr.TotalOutputTokens,
			TotalCacheCreationTokens: sr.TotalCacheCreationTokens,
			TotalCacheReadTokens:     sr.TotalCacheReadTokens,
			TotalTokens:              sr.TotalInputTokens + sr.TotalOutputTokens + sr.TotalCacheCreationTokens + sr.TotalCacheReadTokens,
			ErrorCount:               sr.ErrorCount,
			FirstUserMessage:         sr.FirstUserMessage,
		})
	}

	return &ProjectDailyStatsResponse{
		Date:     date,
		Sessions: sessions,
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

// convertToTimeSeriesDataPoints converts db.TimeSeriesStats to api.TimeSeriesDataPoint
func convertToTimeSeriesDataPoints(stats []db.TimeSeriesStats) []TimeSeriesDataPoint {
	dataPoints := make([]TimeSeriesDataPoint, 0, len(stats))
	for _, ts := range stats {
		dataPoints = append(dataPoints, TimeSeriesDataPoint{
			PeriodStart:              ts.PeriodStart,
			PeriodEnd:                ts.PeriodEnd,
			SessionCount:             ts.SessionCount,
			TotalInputTokens:         ts.TotalInputTokens,
			TotalOutputTokens:        ts.TotalOutputTokens,
			TotalCacheCreationTokens: ts.TotalCacheCreationTokens,
			TotalCacheReadTokens:     ts.TotalCacheReadTokens,
			TotalTokens:              ts.TotalInputTokens + ts.TotalOutputTokens,
		})
	}
	return dataPoints
}
