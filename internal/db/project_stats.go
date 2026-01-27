package db

import (
	"database/sql"
	"fmt"
	"time"
)

// parseDateTime parses SQLite datetime string in various formats
func parseDateTime(s string) (time.Time, error) {
	// 試すフォーマット一覧（優先度順）
	formats := []string{
		time.RFC3339Nano,                       // 2006-01-02T15:04:05.999999999Z07:00
		time.RFC3339,                           // 2006-01-02T15:04:05Z07:00
		"2006-01-02 15:04:05.999999999 -0700 MST", // 2026-01-10 06:13:10.028 +0000 UTC (Go time.Time.String() format)
		"2006-01-02 15:04:05.999999 -0700 MST", // 2026-01-10 06:13:10.123456 +0000 UTC
		"2006-01-02 15:04:05.999 -0700 MST",    // 2026-01-10 06:13:10.028 +0000 UTC
		"2006-01-02 15:04:05 -0700 MST",        // 2026-01-10 06:13:10 +0000 UTC
		"2006-01-02 15:04:05.999999-07:00",     // 2026-01-20 10:00:00.123456+00:00
		"2006-01-02 15:04:05-07:00",            // 2026-01-20 10:00:00+00:00
		"2006-01-02 15:04:05",                  // 2026-01-20 10:00:00
		"2006-01-02",                           // 2026-01-20
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unsupported datetime format: %s", s)
}

// ProjectStats represents project-level statistics
type ProjectStats struct {
	TotalSessions            int       `json:"totalSessions"`
	TotalInputTokens         int       `json:"totalInputTokens"`
	TotalOutputTokens        int       `json:"totalOutputTokens"`
	TotalCacheCreationTokens int       `json:"totalCacheCreationTokens"`
	TotalCacheReadTokens     int       `json:"totalCacheReadTokens"`
	AvgTokens                float64   `json:"avgTokens"`
	FirstSession             time.Time `json:"firstSession"`
	LastSession              time.Time `json:"lastSession"`
	ErrorRate                float64   `json:"errorRate"`
}

// BranchStats represents statistics per branch
type BranchStats struct {
	Branch                   string    `json:"branch"`
	SessionCount             int       `json:"sessionCount"`
	TotalInputTokens         int       `json:"totalInputTokens"`
	TotalOutputTokens        int       `json:"totalOutputTokens"`
	TotalCacheCreationTokens int       `json:"totalCacheCreationTokens"`
	TotalCacheReadTokens     int       `json:"totalCacheReadTokens"`
	LastActivity             time.Time `json:"lastActivity"`
}

// TimeSeriesStats represents time-series statistics
type TimeSeriesStats struct {
	PeriodStart              time.Time `json:"periodStart"`
	PeriodEnd                time.Time `json:"periodEnd"`
	SessionCount             int       `json:"sessionCount"`
	TotalInputTokens         int       `json:"totalInputTokens"`
	TotalOutputTokens        int       `json:"totalOutputTokens"`
	TotalCacheCreationTokens int       `json:"totalCacheCreationTokens"`
	TotalCacheReadTokens     int       `json:"totalCacheReadTokens"`
}

// GetProjectStats retrieves overall statistics for a project
func (db *DB) GetProjectStats(projectID int64) (*ProjectStats, error) {
	query := `
		SELECT
			COUNT(*) as total_sessions,
			COALESCE(SUM(total_input_tokens), 0) as total_input_tokens,
			COALESCE(SUM(total_output_tokens), 0) as total_output_tokens,
			COALESCE(SUM(total_cache_creation_tokens), 0) as total_cache_creation_tokens,
			COALESCE(SUM(total_cache_read_tokens), 0) as total_cache_read_tokens,
			COALESCE(AVG(total_input_tokens + total_output_tokens), 0) as avg_tokens,
			MIN(start_time) as first_session,
			MAX(end_time) as last_session,
			CAST(SUM(CASE WHEN error_count > 0 THEN 1 ELSE 0 END) AS REAL) / NULLIF(COUNT(*), 0) as error_rate
		FROM sessions
		WHERE project_id = ?
	`

	var stats ProjectStats
	var firstSessionStr, lastSessionStr sql.NullString
	var errorRate sql.NullFloat64

	err := db.conn.QueryRow(query, projectID).Scan(
		&stats.TotalSessions,
		&stats.TotalInputTokens,
		&stats.TotalOutputTokens,
		&stats.TotalCacheCreationTokens,
		&stats.TotalCacheReadTokens,
		&stats.AvgTokens,
		&firstSessionStr,
		&lastSessionStr,
		&errorRate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query project stats: %w", err)
	}

	// NULL値の処理と日時パース
	// セッション数が0の場合、日時のパースはスキップ
	if stats.TotalSessions > 0 {
		if firstSessionStr.Valid {
			stats.FirstSession, err = parseDateTime(firstSessionStr.String)
			if err != nil {
				return nil, fmt.Errorf("failed to parse first session time: %w", err)
			}
		}
		if lastSessionStr.Valid {
			stats.LastSession, err = parseDateTime(lastSessionStr.String)
			if err != nil {
				return nil, fmt.Errorf("failed to parse last session time: %w", err)
			}
		}
	}
	if errorRate.Valid {
		stats.ErrorRate = errorRate.Float64
	}

	return &stats, nil
}

// GetBranchStats retrieves statistics per branch for a project
func (db *DB) GetBranchStats(projectID int64) ([]BranchStats, error) {
	query := `
		SELECT
			git_branch,
			COUNT(*) as session_count,
			COALESCE(SUM(total_input_tokens), 0) as total_input_tokens,
			COALESCE(SUM(total_output_tokens), 0) as total_output_tokens,
			COALESCE(SUM(total_cache_creation_tokens), 0) as total_cache_creation_tokens,
			COALESCE(SUM(total_cache_read_tokens), 0) as total_cache_read_tokens,
			MAX(end_time) as last_activity
		FROM sessions
		WHERE project_id = ?
		GROUP BY git_branch
		ORDER BY session_count DESC, git_branch
	`

	rows, err := db.conn.Query(query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to query branch stats: %w", err)
	}
	defer rows.Close()

	var branchStats []BranchStats
	for rows.Next() {
		var stats BranchStats
		var lastActivityStr string

		err := rows.Scan(
			&stats.Branch,
			&stats.SessionCount,
			&stats.TotalInputTokens,
			&stats.TotalOutputTokens,
			&stats.TotalCacheCreationTokens,
			&stats.TotalCacheReadTokens,
			&lastActivityStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan branch stats: %w", err)
		}

		// 日時パース
		stats.LastActivity, err = parseDateTime(lastActivityStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse last activity time: %w", err)
		}

		branchStats = append(branchStats, stats)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating branch stats: %w", err)
	}

	return branchStats, nil
}

// GetTimeSeriesStats retrieves time-series statistics for a project
// period can be "day", "week", or "month"
// limit specifies the maximum number of periods to return (default: 30)
func (db *DB) GetTimeSeriesStats(projectID int64, period string, limit int) ([]TimeSeriesStats, error) {
	if limit <= 0 {
		limit = 30
	}

	// 期間ごとのグループ化SQL
	var dateFormat string
	switch period {
	case "day":
		dateFormat = "%Y-%m-%d"
	case "week":
		// SQLiteのweek開始日は日曜日
		dateFormat = "%Y-%W"
	case "month":
		dateFormat = "%Y-%m"
	default:
		return nil, fmt.Errorf("invalid period: %s (must be day, week, or month)", period)
	}

	query := fmt.Sprintf(`
		SELECT
			STRFTIME('%s', start_time) as period_group,
			MIN(DATE(start_time)) as period_start,
			MAX(DATE(start_time)) as period_end,
			COUNT(*) as session_count,
			COALESCE(SUM(total_input_tokens), 0) as total_input_tokens,
			COALESCE(SUM(total_output_tokens), 0) as total_output_tokens,
			COALESCE(SUM(total_cache_creation_tokens), 0) as total_cache_creation_tokens,
			COALESCE(SUM(total_cache_read_tokens), 0) as total_cache_read_tokens
		FROM sessions
		WHERE project_id = ?
		GROUP BY period_group
		ORDER BY period_start DESC
		LIMIT ?
	`, dateFormat)

	rows, err := db.conn.Query(query, projectID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query time series stats: %w", err)
	}
	defer rows.Close()

	var timeSeriesStats []TimeSeriesStats
	for rows.Next() {
		var stats TimeSeriesStats
		var periodGroup sql.NullString
		var periodStartStr, periodEndStr sql.NullString

		err := rows.Scan(
			&periodGroup,
			&periodStartStr,
			&periodEndStr,
			&stats.SessionCount,
			&stats.TotalInputTokens,
			&stats.TotalOutputTokens,
			&stats.TotalCacheCreationTokens,
			&stats.TotalCacheReadTokens,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan time series stats: %w", err)
		}

		// NULL値をスキップ
		if !periodStartStr.Valid || !periodEndStr.Valid {
			continue
		}

		// 日付文字列をtime.Timeに変換
		stats.PeriodStart, err = parseDateTime(periodStartStr.String)
		if err != nil {
			return nil, fmt.Errorf("failed to parse period start: %w", err)
		}
		stats.PeriodEnd, err = parseDateTime(periodEndStr.String)
		if err != nil {
			return nil, fmt.Errorf("failed to parse period end: %w", err)
		}

		timeSeriesStats = append(timeSeriesStats, stats)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating time series stats: %w", err)
	}

	return timeSeriesStats, nil
}

// GetProjectDailySessions retrieves sessions for a project on a specific date
// date should be in "YYYY-MM-DD" format
func (db *DB) GetProjectDailySessions(projectID int64, date string) ([]SessionRow, error) {
	query := `
		SELECT
			id, project_id, git_branch, start_time, end_time, duration_seconds,
			total_input_tokens, total_output_tokens,
			total_cache_creation_tokens, total_cache_read_tokens,
			error_count, first_user_message, created_at, updated_at
		FROM sessions
		WHERE project_id = ? AND DATE(start_time) = ?
		ORDER BY start_time DESC
	`

	rows, err := db.conn.Query(query, projectID, date)
	if err != nil {
		return nil, fmt.Errorf("failed to query project daily sessions: %w", err)
	}
	defer rows.Close()

	var sessions []SessionRow
	for rows.Next() {
		var s SessionRow
		var startTimeStr, endTimeStr, createdAtStr, updatedAtStr string
		var firstUserMsg sql.NullString

		err := rows.Scan(
			&s.ID,
			&s.ProjectID,
			&s.GitBranch,
			&startTimeStr,
			&endTimeStr,
			&s.DurationSeconds,
			&s.TotalInputTokens,
			&s.TotalOutputTokens,
			&s.TotalCacheCreationTokens,
			&s.TotalCacheReadTokens,
			&s.ErrorCount,
			&firstUserMsg,
			&createdAtStr,
			&updatedAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session row: %w", err)
		}

		// 日時パース
		s.StartTime, err = parseDateTime(startTimeStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse start time: %w", err)
		}
		s.EndTime, err = parseDateTime(endTimeStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse end time: %w", err)
		}
		s.CreatedAt, err = parseDateTime(createdAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse created at: %w", err)
		}
		s.UpdatedAt, err = parseDateTime(updatedAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse updated at: %w", err)
		}

		// FirstUserMessageのNULL処理
		if firstUserMsg.Valid {
			s.FirstUserMessage = firstUserMsg.String
		}

		sessions = append(sessions, s)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating session rows: %w", err)
	}

	return sessions, nil
}
