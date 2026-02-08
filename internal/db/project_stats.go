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

	// 期間の検証
	if period != "day" && period != "week" && period != "month" {
		return nil, fmt.Errorf("invalid period: %s (must be day, week, or month)", period)
	}

	// セッションを取得（広めの期間で取得）
	// limitの期間分のデータを取得するため、start_timeベースで範囲を広めに取る
	query := `
		SELECT
			id, project_id, git_branch, start_time, end_time, duration_seconds,
			total_input_tokens, total_output_tokens,
			total_cache_creation_tokens, total_cache_read_tokens,
			error_count, first_user_message, created_at, updated_at
		FROM sessions
		WHERE project_id = ? AND start_time > '0001-01-02'
		ORDER BY start_time DESC
	`

	rows, err := db.conn.Query(query, projectID)
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

	// セッションを日付ごとに展開して集約
	return aggregateSessionsByPeriod(sessions, period, limit), nil
}

// aggregateSessionsByPeriod セッションを期間ごとに集約
func aggregateSessionsByPeriod(sessions []SessionRow, period string, limit int) []TimeSeriesStats {
	// 日付ごとにセッションを展開
	type dailySession struct {
		date    string
		session SessionRow
	}
	var dailySessions []dailySession

	for _, s := range sessions {
		dates := generateDateRange(s.StartTime, s.EndTime)
		for _, date := range dates {
			dailySessions = append(dailySessions, dailySession{date: date, session: s})
		}
	}

	// 期間ごとにグループ化
	periodMap := make(map[string]*TimeSeriesStats)
	periodSessions := make(map[string]map[string]bool) // 各期間に含まれるセッションID

	for _, ds := range dailySessions {
		periodKey := getPeriodKey(ds.date, period)

		if _, exists := periodMap[periodKey]; !exists {
			periodMap[periodKey] = &TimeSeriesStats{
				SessionCount:             0,
				TotalInputTokens:         0,
				TotalOutputTokens:        0,
				TotalCacheCreationTokens: 0,
				TotalCacheReadTokens:     0,
			}
			periodSessions[periodKey] = make(map[string]bool)
		}

		// セッションIDの重複チェック（同じ期間内で同じセッションは1回だけカウント）
		if !periodSessions[periodKey][ds.session.ID] {
			periodMap[periodKey].SessionCount++
			periodMap[periodKey].TotalInputTokens += ds.session.TotalInputTokens
			periodMap[periodKey].TotalOutputTokens += ds.session.TotalOutputTokens
			periodMap[periodKey].TotalCacheCreationTokens += ds.session.TotalCacheCreationTokens
			periodMap[periodKey].TotalCacheReadTokens += ds.session.TotalCacheReadTokens
			periodSessions[periodKey][ds.session.ID] = true
		}
	}

	// 期間キーをソートして時系列データに変換
	var periodKeys []string
	for key := range periodMap {
		periodKeys = append(periodKeys, key)
	}

	// キーをソート（降順）
	sortPeriodKeys(periodKeys, period)

	// 結果を構築（limitまで）
	var result []TimeSeriesStats
	for i, key := range periodKeys {
		if i >= limit {
			break
		}

		stats := periodMap[key]
		stats.PeriodStart, stats.PeriodEnd = getPeriodRange(key, period)
		result = append(result, *stats)
	}

	// 結果を古い順に並び替え
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result
}

// getPeriodKey 日付から期間キーを生成
func getPeriodKey(date string, period string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}

	switch period {
	case "day":
		return date
	case "week":
		// ISO週番号を使用（月曜日始まり）
		year, week := t.ISOWeek()
		return fmt.Sprintf("%d-W%02d", year, week)
	case "month":
		return t.Format("2006-01")
	default:
		return date
	}
}

// sortPeriodKeys 期間キーをソート（降順）
func sortPeriodKeys(keys []string, period string) {
	// 降順ソート
	for i := 0; i < len(keys)-1; i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] < keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
}

// getPeriodRange 期間キーから開始日と終了日を取得
func getPeriodRange(periodKey string, period string) (time.Time, time.Time) {
	switch period {
	case "day":
		t, _ := time.Parse("2006-01-02", periodKey)
		return t, t
	case "week":
		// ISO週番号から開始日と終了日を計算
		var year, week int
		fmt.Sscanf(periodKey, "%d-W%d", &year, &week)

		// その年の1月4日を基準にISO週を計算
		jan4 := time.Date(year, time.January, 4, 0, 0, 0, 0, time.UTC)
		// 1月4日が含まれる週の月曜日を取得
		mondayOffset := int(time.Monday - jan4.Weekday())
		if mondayOffset > 0 {
			mondayOffset -= 7
		}
		firstMonday := jan4.AddDate(0, 0, mondayOffset)

		// 目的の週の月曜日を計算
		weekStart := firstMonday.AddDate(0, 0, (week-1)*7)
		weekEnd := weekStart.AddDate(0, 0, 6)

		return weekStart, weekEnd
	case "month":
		t, _ := time.Parse("2006-01", periodKey)
		// 月の最終日を取得
		nextMonth := t.AddDate(0, 1, 0)
		lastDay := nextMonth.AddDate(0, 0, -1)
		return t, lastDay
	default:
		t, _ := time.Parse("2006-01-02", periodKey)
		return t, t
	}
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
		WHERE project_id = ?
		  AND DATE(start_time) <= ?
		  AND DATE(end_time) >= ?
		ORDER BY start_time DESC
	`

	rows, err := db.conn.Query(query, projectID, date, date)
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
