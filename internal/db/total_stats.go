package db

import (
	"database/sql"
	"fmt"
	"time"
)

// TotalStats represents total statistics across all projects
type TotalStats struct {
	TotalGroups              int       `json:"totalGroups"`
	TotalProjects            int       `json:"totalProjects"`
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

// GetTotalStats retrieves overall statistics across all projects
func (db *DB) GetTotalStats() (*TotalStats, error) {
	query := `
		SELECT
			(SELECT COUNT(DISTINCT id) FROM project_groups) as total_groups,
			COUNT(DISTINCT p.id) as total_projects,
			COUNT(s.id) as total_sessions,
			COALESCE(SUM(s.total_input_tokens), 0) as total_input_tokens,
			COALESCE(SUM(s.total_output_tokens), 0) as total_output_tokens,
			COALESCE(SUM(s.total_cache_creation_tokens), 0) as total_cache_creation_tokens,
			COALESCE(SUM(s.total_cache_read_tokens), 0) as total_cache_read_tokens,
			COALESCE(AVG(s.total_input_tokens + s.total_output_tokens), 0) as avg_tokens,
			MIN(s.start_time) as first_session,
			MAX(s.end_time) as last_session,
			CAST(SUM(CASE WHEN s.error_count > 0 THEN 1 ELSE 0 END) AS REAL) / NULLIF(COUNT(s.id), 0) as error_rate
		FROM projects p
		LEFT JOIN sessions s ON p.id = s.project_id
	`

	var stats TotalStats
	var firstSessionStr, lastSessionStr sql.NullString
	var errorRate sql.NullFloat64

	err := db.conn.QueryRow(query).Scan(
		&stats.TotalGroups,
		&stats.TotalProjects,
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
		return nil, fmt.Errorf("failed to query total stats: %w", err)
	}

	// NULL値の処理と日時パース
	if stats.TotalSessions > 0 {
		if firstSessionStr.Valid {
			stats.FirstSession, err = parseDateTime(firstSessionStr.String)
			if err != nil {
				return nil, fmt.Errorf("failed to parse first session time: %w", err)
			}
		} else {
			// セッションが存在するがfirst_sessionがNULLの場合は異常
			return nil, fmt.Errorf("sessions exist but first_session is null: data integrity issue")
		}
		if lastSessionStr.Valid {
			stats.LastSession, err = parseDateTime(lastSessionStr.String)
			if err != nil {
				return nil, fmt.Errorf("failed to parse last session time: %w", err)
			}
		} else {
			// セッションが存在するがlast_sessionがNULLの場合は異常
			return nil, fmt.Errorf("sessions exist but last_session is null: data integrity issue")
		}
	}
	// セッションが0件の場合、FirstSessionとLastSessionはゼロ値のままで問題なし
	if errorRate.Valid {
		stats.ErrorRate = errorRate.Float64
	}

	return &stats, nil
}

// GetTotalTimeSeriesStats retrieves time-series statistics across all projects
// period can be "day", "week", or "month"
// limit specifies the maximum number of periods to return (default: 30)
func (db *DB) GetTotalTimeSeriesStats(period string, limit int) ([]TimeSeriesStats, error) {
	if limit <= 0 {
		limit = 30
	}

	// 期間の検証
	if period != "day" && period != "week" && period != "month" {
		return nil, fmt.Errorf("invalid period: %s (must be day, week, or month)", period)
	}

	// 全プロジェクトのセッションを取得
	query := `
		SELECT
			id, project_id, git_branch, start_time, end_time, duration_seconds,
			total_input_tokens, total_output_tokens,
			total_cache_creation_tokens, total_cache_read_tokens,
			error_count, first_user_message, created_at, updated_at
		FROM sessions
		WHERE start_time > '0001-01-02'
		ORDER BY start_time DESC
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
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
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}

		s.StartTime, _ = time.Parse(time.RFC3339Nano, startTimeStr)
		s.EndTime, _ = time.Parse(time.RFC3339Nano, endTimeStr)
		s.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAtStr)
		s.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAtStr)
		if firstUserMsg.Valid {
			s.FirstUserMessage = firstUserMsg.String
		}

		sessions = append(sessions, s)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sessions: %w", err)
	}

	// セッションを日付ごとに展開して集約（project_stats.goのヘルパー関数を再利用）
	return aggregateSessionsByPeriod(sessions, period, limit), nil
}

// DailyGroupStats represents statistics for a single group on a specific date
type DailyGroupStats struct {
	GroupID                  int64   `json:"groupId"`
	GroupName                string  `json:"groupName"`
	SessionCount             int     `json:"sessionCount"`
	TotalInputTokens         int     `json:"totalInputTokens"`
	TotalOutputTokens        int     `json:"totalOutputTokens"`
	TotalCacheCreationTokens int     `json:"totalCacheCreationTokens"`
	TotalCacheReadTokens     int     `json:"totalCacheReadTokens"`
}

// GetDailyGroupStats retrieves group-wise statistics for a specific date
func (db *DB) GetDailyGroupStats(date string) ([]DailyGroupStats, error) {
	query := `
		SELECT
			pg.id as group_id,
			pg.name as group_name,
			COUNT(s.id) as session_count,
			COALESCE(SUM(s.total_input_tokens), 0) as total_input_tokens,
			COALESCE(SUM(s.total_output_tokens), 0) as total_output_tokens,
			COALESCE(SUM(s.total_cache_creation_tokens), 0) as total_cache_creation_tokens,
			COALESCE(SUM(s.total_cache_read_tokens), 0) as total_cache_read_tokens
		FROM project_groups pg
		INNER JOIN project_group_mappings pgm ON pg.id = pgm.group_id
		INNER JOIN projects p ON pgm.project_id = p.id
		INNER JOIN sessions s ON p.id = s.project_id
		WHERE DATE(s.start_time) <= ?
		  AND DATE(s.end_time) >= ?
		GROUP BY pg.id, pg.name
		HAVING session_count > 0
		ORDER BY (total_input_tokens + total_output_tokens) DESC
	`

	rows, err := db.conn.Query(query, date, date)
	if err != nil {
		return nil, fmt.Errorf("failed to query daily group stats: %w", err)
	}
	defer rows.Close()

	var stats []DailyGroupStats
	for rows.Next() {
		var s DailyGroupStats
		err := rows.Scan(
			&s.GroupID,
			&s.GroupName,
			&s.SessionCount,
			&s.TotalInputTokens,
			&s.TotalOutputTokens,
			&s.TotalCacheCreationTokens,
			&s.TotalCacheReadTokens,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan daily group stats: %w", err)
		}
		stats = append(stats, s)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating daily group stats: %w", err)
	}

	return stats, nil
}
