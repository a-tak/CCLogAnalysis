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
