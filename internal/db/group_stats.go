package db

import (
	"database/sql"
	"fmt"
	"time"
)

// GroupStats represents group-level statistics
type GroupStats struct {
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

// DailyProjectStats represents project-level statistics for a specific date
type DailyProjectStats struct {
	ProjectID                int64  `json:"projectId"`
	ProjectName              string `json:"projectName"`
	SessionCount             int    `json:"sessionCount"`
	TotalInputTokens         int    `json:"totalInputTokens"`
	TotalOutputTokens        int    `json:"totalOutputTokens"`
	TotalCacheCreationTokens int    `json:"totalCacheCreationTokens"`
	TotalCacheReadTokens     int    `json:"totalCacheReadTokens"`
	TotalTokens              int    `json:"totalTokens"`
}

// GetGroupStats retrieves overall statistics for a project group
func (db *DB) GetGroupStats(groupID int64) (*GroupStats, error) {
	query := `
		SELECT
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
		FROM project_group_mappings pgm
		INNER JOIN projects p ON pgm.project_id = p.id
		LEFT JOIN sessions s ON p.id = s.project_id
		WHERE pgm.group_id = ?
	`

	var stats GroupStats
	var firstSessionStr, lastSessionStr sql.NullString
	var errorRate sql.NullFloat64

	err := db.conn.QueryRow(query, groupID).Scan(
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
		return nil, fmt.Errorf("failed to query group stats: %w", err)
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

// GetGroupTimeSeriesStats retrieves time-series statistics for a project group
// period can be "day", "week", or "month"
// limit specifies the maximum number of periods to return (default: 30)
func (db *DB) GetGroupTimeSeriesStats(groupID int64, period string, limit int) ([]TimeSeriesStats, error) {
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
			STRFTIME('%s', s.start_time) as period_group,
			MIN(DATE(s.start_time)) as period_start,
			MAX(DATE(s.start_time)) as period_end,
			COUNT(*) as session_count,
			COALESCE(SUM(s.total_input_tokens), 0) as total_input_tokens,
			COALESCE(SUM(s.total_output_tokens), 0) as total_output_tokens,
			COALESCE(SUM(s.total_cache_creation_tokens), 0) as total_cache_creation_tokens,
			COALESCE(SUM(s.total_cache_read_tokens), 0) as total_cache_read_tokens
		FROM project_group_mappings pgm
		INNER JOIN projects p ON pgm.project_id = p.id
		INNER JOIN sessions s ON p.id = s.project_id
		WHERE pgm.group_id = ? AND s.start_time > '0001-01-02'  -- SQLiteの最小日付より後のデータのみを対象
		GROUP BY period_group
		ORDER BY period_start ASC
		LIMIT ?
	`, dateFormat)

	rows, err := db.conn.Query(query, groupID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query group time series stats: %w", err)
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
			return nil, fmt.Errorf("failed to scan group time series stats: %w", err)
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
		return nil, fmt.Errorf("error iterating group time series stats: %w", err)
	}

	return timeSeriesStats, nil
}

// GetGroupDailyProjectStats retrieves project-wise statistics for a group on a specific date
// date should be in "YYYY-MM-DD" format
func (db *DB) GetGroupDailyProjectStats(groupID int64, date string) ([]DailyProjectStats, error) {
	query := `
		SELECT
			p.id as project_id,
			p.name as project_name,
			COUNT(s.id) as session_count,
			COALESCE(SUM(s.total_input_tokens), 0) as total_input_tokens,
			COALESCE(SUM(s.total_output_tokens), 0) as total_output_tokens,
			COALESCE(SUM(s.total_cache_creation_tokens), 0) as total_cache_creation_tokens,
			COALESCE(SUM(s.total_cache_read_tokens), 0) as total_cache_read_tokens
		FROM project_group_mappings pgm
		INNER JOIN projects p ON pgm.project_id = p.id
		INNER JOIN sessions s ON p.id = s.project_id
		WHERE pgm.group_id = ? AND DATE(s.start_time) = ?
		GROUP BY p.id, p.name
		HAVING session_count > 0
		ORDER BY (total_input_tokens + total_output_tokens + total_cache_creation_tokens + total_cache_read_tokens) DESC
	`

	rows, err := db.conn.Query(query, groupID, date)
	if err != nil {
		return nil, fmt.Errorf("failed to query group daily project stats: %w", err)
	}
	defer rows.Close()

	var stats []DailyProjectStats
	for rows.Next() {
		var s DailyProjectStats
		err := rows.Scan(
			&s.ProjectID,
			&s.ProjectName,
			&s.SessionCount,
			&s.TotalInputTokens,
			&s.TotalOutputTokens,
			&s.TotalCacheCreationTokens,
			&s.TotalCacheReadTokens,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan daily project stats: %w", err)
		}

		// TotalTokensを計算
		s.TotalTokens = s.TotalInputTokens + s.TotalOutputTokens +
			s.TotalCacheCreationTokens + s.TotalCacheReadTokens

		stats = append(stats, s)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating daily project stats: %w", err)
	}

	return stats, nil
}
