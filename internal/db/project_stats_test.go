package db

import (
	"testing"
	"time"
)

func TestGetProjectStats(t *testing.T) {
	// テストデータベースの準備
	db, _ := setupTestDB(t)
	defer db.Close()

	// テストプロジェクト作成
	projectID, err := db.CreateProject("test-project", "/path/to/project")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// 複数のセッションを作成（異なる日付、ブランチ、エラー状態）
	sessions := []struct {
		id          string
		branch      string
		startTime   time.Time
		endTime     time.Time
		inputTokens int
		outputTokens int
		errorCount  int
	}{
		{
			id:           "session-1",
			branch:       "main",
			startTime:    time.Date(2026, 1, 20, 10, 0, 0, 0, time.UTC),
			endTime:      time.Date(2026, 1, 20, 11, 0, 0, 0, time.UTC),
			inputTokens:  1000,
			outputTokens: 500,
			errorCount:   0,
		},
		{
			id:           "session-2",
			branch:       "feature-a",
			startTime:    time.Date(2026, 1, 21, 10, 0, 0, 0, time.UTC),
			endTime:      time.Date(2026, 1, 21, 11, 30, 0, 0, time.UTC),
			inputTokens:  2000,
			outputTokens: 1000,
			errorCount:   2,
		},
		{
			id:           "session-3",
			branch:       "main",
			startTime:    time.Date(2026, 1, 22, 10, 0, 0, 0, time.UTC),
			endTime:      time.Date(2026, 1, 22, 10, 30, 0, 0, time.UTC),
			inputTokens:  500,
			outputTokens: 250,
			errorCount:   0,
		},
	}

	for _, s := range sessions {
		duration := int(s.endTime.Sub(s.startTime).Seconds())
		_, err := db.conn.Exec(`
			INSERT INTO sessions (
				id, project_id, git_branch, start_time, end_time, duration_seconds,
				total_input_tokens, total_output_tokens,
				total_cache_creation_tokens, total_cache_read_tokens,
				error_count
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, s.id, projectID, s.branch,
			s.startTime.Format(time.RFC3339Nano),
			s.endTime.Format(time.RFC3339Nano),
			duration,
			s.inputTokens, s.outputTokens, 0, 0, s.errorCount)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
	}

	// プロジェクト統計を取得
	stats, err := db.GetProjectStats(projectID)
	if err != nil {
		t.Fatalf("GetProjectStats failed: %v", err)
	}

	// 検証
	if stats.TotalSessions != 3 {
		t.Errorf("Expected 3 sessions, got %d", stats.TotalSessions)
	}

	expectedInputTokens := 1000 + 2000 + 500
	if stats.TotalInputTokens != expectedInputTokens {
		t.Errorf("Expected %d input tokens, got %d", expectedInputTokens, stats.TotalInputTokens)
	}

	expectedOutputTokens := 500 + 1000 + 250
	if stats.TotalOutputTokens != expectedOutputTokens {
		t.Errorf("Expected %d output tokens, got %d", expectedOutputTokens, stats.TotalOutputTokens)
	}

	expectedAvgTokens := float64(expectedInputTokens+expectedOutputTokens) / float64(3)
	if stats.AvgTokens != expectedAvgTokens {
		t.Errorf("Expected %f avg tokens, got %f", expectedAvgTokens, stats.AvgTokens)
	}

	expectedErrorRate := float64(1) / float64(3) // 1セッションにエラーあり
	if stats.ErrorRate != expectedErrorRate {
		t.Errorf("Expected error rate %.2f, got %.2f", expectedErrorRate, stats.ErrorRate)
	}

	if !stats.FirstSession.Equal(sessions[0].startTime) {
		t.Errorf("Expected first session %v, got %v", sessions[0].startTime, stats.FirstSession)
	}

	if !stats.LastSession.Equal(sessions[2].endTime) {
		t.Errorf("Expected last session %v, got %v", sessions[2].endTime, stats.LastSession)
	}
}

func TestGetProjectStatsNoSessions(t *testing.T) {
	// テストデータベースの準備
	db, _ := setupTestDB(t)
	defer db.Close()

	// セッションがないプロジェクトを作成
	projectID, err := db.CreateProject("empty-project", "/path/to/empty")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// プロジェクト統計を取得
	stats, err := db.GetProjectStats(projectID)
	if err != nil {
		t.Fatalf("GetProjectStats failed: %v", err)
	}

	// ゼロ値であることを確認
	if stats.TotalSessions != 0 {
		t.Errorf("Expected 0 sessions, got %d", stats.TotalSessions)
	}
	if stats.TotalInputTokens != 0 {
		t.Errorf("Expected 0 input tokens, got %d", stats.TotalInputTokens)
	}
	if stats.ErrorRate != 0 {
		t.Errorf("Expected 0 error rate, got %.2f", stats.ErrorRate)
	}
}

func TestGetBranchStats(t *testing.T) {
	// テストデータベースの準備
	db, _ := setupTestDB(t)
	defer db.Close()

	// テストプロジェクト作成
	projectID, err := db.CreateProject("test-project", "/path/to/project")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// 複数のブランチでセッションを作成
	sessions := []struct {
		id          string
		branch      string
		inputTokens int
		outputTokens int
	}{
		{"session-1", "main", 1000, 500},
		{"session-2", "main", 1500, 750},
		{"session-3", "feature-a", 2000, 1000},
		{"session-4", "feature-b", 500, 250},
		{"session-5", "feature-a", 1000, 500},
	}

	now := time.Now()
	for _, s := range sessions {
		_, err := db.conn.Exec(`
			INSERT INTO sessions (
				id, project_id, git_branch, start_time, end_time, duration_seconds,
				total_input_tokens, total_output_tokens,
				total_cache_creation_tokens, total_cache_read_tokens,
				error_count
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, s.id, projectID, s.branch,
			now.Format(time.RFC3339Nano),
			now.Add(time.Hour).Format(time.RFC3339Nano),
			3600,
			s.inputTokens, s.outputTokens, 0, 0, 0)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
	}

	// ブランチ統計を取得
	branchStats, err := db.GetBranchStats(projectID)
	if err != nil {
		t.Fatalf("GetBranchStats failed: %v", err)
	}

	// 3つのブランチがあることを確認
	if len(branchStats) != 3 {
		t.Fatalf("Expected 3 branches, got %d", len(branchStats))
	}

	// mainブランチの検証
	mainStats := findBranchStats(branchStats, "main")
	if mainStats == nil {
		t.Fatal("main branch not found")
	}
	if mainStats.SessionCount != 2 {
		t.Errorf("Expected 2 sessions for main, got %d", mainStats.SessionCount)
	}
	if mainStats.TotalInputTokens != 2500 {
		t.Errorf("Expected 2500 input tokens for main, got %d", mainStats.TotalInputTokens)
	}

	// feature-aブランチの検証
	featureAStats := findBranchStats(branchStats, "feature-a")
	if featureAStats == nil {
		t.Fatal("feature-a branch not found")
	}
	if featureAStats.SessionCount != 2 {
		t.Errorf("Expected 2 sessions for feature-a, got %d", featureAStats.SessionCount)
	}
	if featureAStats.TotalInputTokens != 3000 {
		t.Errorf("Expected 3000 input tokens for feature-a, got %d", featureAStats.TotalInputTokens)
	}
}

func TestGetTimeSeriesStats(t *testing.T) {
	// テストデータベースの準備
	db, _ := setupTestDB(t)
	defer db.Close()

	// テストプロジェクト作成
	projectID, err := db.CreateProject("test-project", "/path/to/project")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// 異なる日付のセッションを作成
	sessions := []struct {
		id        string
		startTime time.Time
		tokens    int
	}{
		{"session-1", time.Date(2026, 1, 20, 10, 0, 0, 0, time.UTC), 1000},
		{"session-2", time.Date(2026, 1, 20, 14, 0, 0, 0, time.UTC), 1500},
		{"session-3", time.Date(2026, 1, 21, 10, 0, 0, 0, time.UTC), 2000},
		{"session-4", time.Date(2026, 1, 22, 10, 0, 0, 0, time.UTC), 500},
	}

	for _, s := range sessions {
		endTime := s.startTime.Add(time.Hour)
		_, err := db.conn.Exec(`
			INSERT INTO sessions (
				id, project_id, git_branch, start_time, end_time, duration_seconds,
				total_input_tokens, total_output_tokens,
				total_cache_creation_tokens, total_cache_read_tokens,
				error_count
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, s.id, projectID, "main",
			s.startTime.Format(time.RFC3339Nano),
			endTime.Format(time.RFC3339Nano),
			3600,
			s.tokens, 0, 0, 0, 0)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
	}

	// 日別の時系列統計を取得（limit=30）
	timeSeriesStats, err := db.GetTimeSeriesStats(projectID, "day", 30)
	if err != nil {
		t.Fatalf("GetTimeSeriesStats failed: %v", err)
	}

	// 3日分のデータがあることを確認
	if len(timeSeriesStats) != 3 {
		t.Fatalf("Expected 3 days, got %d", len(timeSeriesStats))
	}

	// 昇順ソート（古い順）なので、最初は2026-01-20
	day1 := timeSeriesStats[0]
	expectedDate1 := time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC)
	if !day1.PeriodStart.Equal(expectedDate1) {
		t.Errorf("Expected period start %v, got %v", expectedDate1, day1.PeriodStart)
	}
	if day1.SessionCount != 2 {
		t.Errorf("Expected 2 sessions on 2026-01-20, got %d", day1.SessionCount)
	}
	if day1.TotalInputTokens != 2500 {
		t.Errorf("Expected 2500 tokens on 2026-01-20, got %d", day1.TotalInputTokens)
	}

	// 2番目は2026-01-21
	day2 := timeSeriesStats[1]
	expectedDate2 := time.Date(2026, 1, 21, 0, 0, 0, 0, time.UTC)
	if !day2.PeriodStart.Equal(expectedDate2) {
		t.Errorf("Expected period start %v, got %v", expectedDate2, day2.PeriodStart)
	}
	if day2.SessionCount != 1 {
		t.Errorf("Expected 1 session on 2026-01-21, got %d", day2.SessionCount)
	}
	if day2.TotalInputTokens != 2000 {
		t.Errorf("Expected 2000 tokens on 2026-01-21, got %d", day2.TotalInputTokens)
	}

	// 3番目は2026-01-22（1セッション）
	day3 := timeSeriesStats[2]
	expectedDate3 := time.Date(2026, 1, 22, 0, 0, 0, 0, time.UTC)
	if !day3.PeriodStart.Equal(expectedDate3) {
		t.Errorf("Expected period start %v, got %v", expectedDate3, day3.PeriodStart)
	}
	if day3.SessionCount != 1 {
		t.Errorf("Expected 1 session on 2026-01-22, got %d", day3.SessionCount)
	}
	if day3.TotalInputTokens != 500 {
		t.Errorf("Expected 500 tokens on 2026-01-22, got %d", day3.TotalInputTokens)
	}
}

func TestGetTimeSeriesStats_CrossMidnight(t *testing.T) {
	// テストデータベースの準備
	db, _ := setupTestDB(t)
	defer db.Close()

	// テストプロジェクト作成
	projectID, err := db.CreateProject("test-project", "/path/to/project")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// 跨日セッションを含むテストデータ
	sessions := []struct {
		id           string
		startTime    time.Time
		endTime      time.Time
		inputTokens  int
		outputTokens int
	}{
		{
			// 1/20 23:00 → 1/21 02:00 (跨日セッション)
			id:           "session-cross-midnight",
			startTime:    time.Date(2026, 1, 20, 23, 0, 0, 0, time.UTC),
			endTime:      time.Date(2026, 1, 21, 2, 0, 0, 0, time.UTC),
			inputTokens:  1000,
			outputTokens: 500,
		},
		{
			// 1/20 10:00 → 1/20 15:00 (同日セッション)
			id:           "session-same-day",
			startTime:    time.Date(2026, 1, 20, 10, 0, 0, 0, time.UTC),
			endTime:      time.Date(2026, 1, 20, 15, 0, 0, 0, time.UTC),
			inputTokens:  2000,
			outputTokens: 1000,
		},
		{
			// 1/20 10:00 → 1/22 15:00 (複数日にまたがるセッション)
			id:           "session-multi-day",
			startTime:    time.Date(2026, 1, 20, 10, 0, 0, 0, time.UTC),
			endTime:      time.Date(2026, 1, 22, 15, 0, 0, 0, time.UTC),
			inputTokens:  3000,
			outputTokens: 1500,
		},
		{
			// 1/21 10:00 → 1/21 15:00 (通常セッション)
			id:           "session-normal",
			startTime:    time.Date(2026, 1, 21, 10, 0, 0, 0, time.UTC),
			endTime:      time.Date(2026, 1, 21, 15, 0, 0, 0, time.UTC),
			inputTokens:  1500,
			outputTokens: 750,
		},
	}

	for _, s := range sessions {
		duration := int(s.endTime.Sub(s.startTime).Seconds())
		_, err := db.conn.Exec(`
			INSERT INTO sessions (
				id, project_id, git_branch, start_time, end_time, duration_seconds,
				total_input_tokens, total_output_tokens,
				total_cache_creation_tokens, total_cache_read_tokens,
				error_count
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, s.id, projectID, "main",
			s.startTime.Format(time.RFC3339Nano),
			s.endTime.Format(time.RFC3339Nano),
			duration,
			s.inputTokens, s.outputTokens, 0, 0, 0)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
	}

	t.Run("跨日セッションが各日に含まれる", func(t *testing.T) {
		// 日別の時系列統計を取得
		timeSeriesStats, err := db.GetTimeSeriesStats(projectID, "day", 30)
		if err != nil {
			t.Fatalf("GetTimeSeriesStats failed: %v", err)
		}

		// 3日分のデータがあることを確認
		if len(timeSeriesStats) != 3 {
			t.Fatalf("Expected 3 days, got %d", len(timeSeriesStats))
		}

		// 日付ごとのマップを作成
		dayMap := make(map[string]TimeSeriesStats)
		for _, stats := range timeSeriesStats {
			dateStr := stats.PeriodStart.Format("2006-01-02")
			dayMap[dateStr] = stats
		}

		// 1/20の検証
		day20, exists := dayMap["2026-01-20"]
		if !exists {
			t.Fatal("Expected data for 2026-01-20")
		}
		// 1/20には3つのセッションが含まれる
		// - session-same-day (同日)
		// - session-cross-midnight (跨日)
		// - session-multi-day (複数日)
		if day20.SessionCount != 3 {
			t.Errorf("Expected 3 sessions on 2026-01-20, got %d", day20.SessionCount)
		}
		expectedTokens20 := 1000 + 2000 + 3000 // cross + same + multi
		if day20.TotalInputTokens != expectedTokens20 {
			t.Errorf("Expected %d input tokens on 2026-01-20, got %d", expectedTokens20, day20.TotalInputTokens)
		}

		// 1/21の検証
		day21, exists := dayMap["2026-01-21"]
		if !exists {
			t.Fatal("Expected data for 2026-01-21")
		}
		// 1/21には3つのセッションが含まれる
		// - session-cross-midnight (跨日)
		// - session-multi-day (複数日)
		// - session-normal (通常)
		if day21.SessionCount != 3 {
			t.Errorf("Expected 3 sessions on 2026-01-21, got %d", day21.SessionCount)
		}
		expectedTokens21 := 1000 + 3000 + 1500 // cross + multi + normal
		if day21.TotalInputTokens != expectedTokens21 {
			t.Errorf("Expected %d input tokens on 2026-01-21, got %d", expectedTokens21, day21.TotalInputTokens)
		}

		// 1/22の検証
		day22, exists := dayMap["2026-01-22"]
		if !exists {
			t.Fatal("Expected data for 2026-01-22")
		}
		// 1/22には1つのセッションが含まれる
		// - session-multi-day (複数日)
		if day22.SessionCount != 1 {
			t.Errorf("Expected 1 session on 2026-01-22, got %d", day22.SessionCount)
		}
		if day22.TotalInputTokens != 3000 {
			t.Errorf("Expected 3000 input tokens on 2026-01-22, got %d", day22.TotalInputTokens)
		}
	})

	t.Run("週別集計で跨日セッションが正しく処理される", func(t *testing.T) {
		// 週別の時系列統計を取得
		weekStats, err := db.GetTimeSeriesStats(projectID, "week", 10)
		if err != nil {
			t.Fatalf("GetTimeSeriesStats failed: %v", err)
		}

		// 全てのセッションが同じ週に含まれるはず（1/20-1/22は同じ週）
		if len(weekStats) != 1 {
			t.Fatalf("Expected 1 week, got %d", len(weekStats))
		}

		// 4つのセッションが含まれるが、各セッションは1回だけカウント
		if weekStats[0].SessionCount != 4 {
			t.Errorf("Expected 4 sessions in the week, got %d", weekStats[0].SessionCount)
		}

		// トークン数の合計
		expectedTokens := 1000 + 2000 + 3000 + 1500
		if weekStats[0].TotalInputTokens != expectedTokens {
			t.Errorf("Expected %d input tokens, got %d", expectedTokens, weekStats[0].TotalInputTokens)
		}
	})

	t.Run("月別集計で跨日セッションが正しく処理される", func(t *testing.T) {
		// 月別の時系列統計を取得
		monthStats, err := db.GetTimeSeriesStats(projectID, "month", 10)
		if err != nil {
			t.Fatalf("GetTimeSeriesStats failed: %v", err)
		}

		// 全てのセッションが同じ月に含まれる
		if len(monthStats) != 1 {
			t.Fatalf("Expected 1 month, got %d", len(monthStats))
		}

		// 4つのセッションが含まれる
		if monthStats[0].SessionCount != 4 {
			t.Errorf("Expected 4 sessions in the month, got %d", monthStats[0].SessionCount)
		}

		// トークン数の合計
		expectedTokens := 1000 + 2000 + 3000 + 1500
		if monthStats[0].TotalInputTokens != expectedTokens {
			t.Errorf("Expected %d input tokens, got %d", expectedTokens, monthStats[0].TotalInputTokens)
		}
	})
}

// ヘルパー関数：ブランチ統計を検索
func findBranchStats(stats []BranchStats, branch string) *BranchStats {
	for _, s := range stats {
		if s.Branch == branch {
			return &s
		}
	}
	return nil
}

func TestGetProjectDailySessions(t *testing.T) {
	// テストデータベースの準備
	db, _ := setupTestDB(t)
	defer db.Close()

	// テストプロジェクト作成
	projectID, err := db.CreateProject("test-project", "/path/to/project")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// 異なる日付でセッションを作成
	targetDate := time.Date(2026, 1, 25, 0, 0, 0, 0, time.UTC)
	otherDate := time.Date(2026, 1, 26, 0, 0, 0, 0, time.UTC)

	sessions := []struct {
		id              string
		branch          string
		startTime       time.Time
		endTime         time.Time
		inputTokens     int
		outputTokens    int
		cacheCreate     int
		cacheRead       int
		errorCount      int
		firstUserMsg    string
	}{
		// ターゲット日のセッション
		{
			id:           "session-1",
			branch:       "main",
			startTime:    targetDate.Add(10 * time.Hour),
			endTime:      targetDate.Add(11 * time.Hour),
			inputTokens:  1000,
			outputTokens: 500,
			cacheCreate:  100,
			cacheRead:    50,
			errorCount:   0,
			firstUserMsg: "First message 1",
		},
		{
			id:           "session-2",
			branch:       "feature-a",
			startTime:    targetDate.Add(14 * time.Hour),
			endTime:      targetDate.Add(15 * time.Hour),
			inputTokens:  1500,
			outputTokens: 750,
			cacheCreate:  150,
			cacheRead:    75,
			errorCount:   1,
			firstUserMsg: "First message 2",
		},
		{
			id:           "session-3",
			branch:       "feature-b",
			startTime:    targetDate.Add(16 * time.Hour),
			endTime:      targetDate.Add(17 * time.Hour),
			inputTokens:  2000,
			outputTokens: 1000,
			cacheCreate:  200,
			cacheRead:    100,
			errorCount:   0,
			firstUserMsg: "First message 3",
		},
		// 他の日のセッション（フィルタされるべき）
		{
			id:           "session-4",
			branch:       "main",
			startTime:    otherDate.Add(10 * time.Hour),
			endTime:      otherDate.Add(11 * time.Hour),
			inputTokens:  999,
			outputTokens: 999,
			cacheCreate:  999,
			cacheRead:    999,
			errorCount:   0,
			firstUserMsg: "Should not appear",
		},
	}

	for _, s := range sessions {
		duration := int(s.endTime.Sub(s.startTime).Seconds())
		_, err := db.conn.Exec(`
			INSERT INTO sessions (
				id, project_id, git_branch, start_time, end_time, duration_seconds,
				total_input_tokens, total_output_tokens,
				total_cache_creation_tokens, total_cache_read_tokens,
				error_count, first_user_message
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, s.id, projectID, s.branch,
			s.startTime.Format(time.RFC3339Nano),
			s.endTime.Format(time.RFC3339Nano),
			duration,
			s.inputTokens, s.outputTokens, s.cacheCreate, s.cacheRead, s.errorCount, s.firstUserMsg)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
	}

	t.Run("指定日のセッション一覧を取得できる", func(t *testing.T) {
		// ターゲット日のセッション取得
		dateStr := targetDate.Format("2006-01-02")
		sessionRows, err := db.GetProjectDailySessions(projectID, dateStr)
		if err != nil {
			t.Fatalf("GetProjectDailySessions failed: %v", err)
		}

		// 3セッション分のデータが返されることを確認
		if len(sessionRows) != 3 {
			t.Fatalf("Expected 3 sessions, got %d", len(sessionRows))
		}

		// 開始時刻降順（session-3 → session-2 → session-1）
		// session-3
		if sessionRows[0].ID != "session-3" {
			t.Errorf("Expected session-3 first, got %s", sessionRows[0].ID)
		}
		if sessionRows[0].GitBranch != "feature-b" {
			t.Errorf("Expected branch feature-b, got %s", sessionRows[0].GitBranch)
		}
		if sessionRows[0].TotalInputTokens != 2000 {
			t.Errorf("Expected 2000 input tokens, got %d", sessionRows[0].TotalInputTokens)
		}
		if sessionRows[0].TotalOutputTokens != 1000 {
			t.Errorf("Expected 1000 output tokens, got %d", sessionRows[0].TotalOutputTokens)
		}
		if sessionRows[0].TotalCacheCreationTokens != 200 {
			t.Errorf("Expected 200 cache creation tokens, got %d", sessionRows[0].TotalCacheCreationTokens)
		}
		if sessionRows[0].TotalCacheReadTokens != 100 {
			t.Errorf("Expected 100 cache read tokens, got %d", sessionRows[0].TotalCacheReadTokens)
		}
		if sessionRows[0].ErrorCount != 0 {
			t.Errorf("Expected 0 errors, got %d", sessionRows[0].ErrorCount)
		}
		if sessionRows[0].FirstUserMessage != "First message 3" {
			t.Errorf("Expected 'First message 3', got %s", sessionRows[0].FirstUserMessage)
		}

		// session-2
		if sessionRows[1].ID != "session-2" {
			t.Errorf("Expected session-2 second, got %s", sessionRows[1].ID)
		}
		if sessionRows[1].ErrorCount != 1 {
			t.Errorf("Expected 1 error, got %d", sessionRows[1].ErrorCount)
		}

		// session-1
		if sessionRows[2].ID != "session-1" {
			t.Errorf("Expected session-1 third, got %s", sessionRows[2].ID)
		}
	})

	t.Run("セッションが存在しない日付で空配列を返す", func(t *testing.T) {
		// 存在しない日付
		noSessionDate := "2026-01-27"
		sessionRows, err := db.GetProjectDailySessions(projectID, noSessionDate)
		if err != nil {
			t.Fatalf("GetProjectDailySessions failed: %v", err)
		}

		// 空配列が返ることを確認
		if len(sessionRows) != 0 {
			t.Errorf("Expected 0 sessions, got %d", len(sessionRows))
		}
	})

	t.Run("存在しないプロジェクトIDでエラーを返さず空配列を返す", func(t *testing.T) {
		// 存在しないプロジェクトID
		dateStr := targetDate.Format("2006-01-02")
		sessionRows, err := db.GetProjectDailySessions(99999, dateStr)
		if err != nil {
			t.Fatalf("GetProjectDailySessions failed: %v", err)
		}

		// 空配列が返ることを確認
		if len(sessionRows) != 0 {
			t.Errorf("Expected 0 sessions, got %d", len(sessionRows))
		}
	})

	t.Run("不正な日付フォーマットでも処理できる", func(t *testing.T) {
		// SQLiteはDATE()関数でパースを試みるので、エラーにはならない可能性がある
		// 空配列が返ることを確認
		sessionRows, err := db.GetProjectDailySessions(projectID, "invalid-date")
		if err != nil {
			t.Fatalf("GetProjectDailySessions failed: %v", err)
		}

		// 空配列が返ることを確認
		if len(sessionRows) != 0 {
			t.Errorf("Expected 0 sessions, got %d", len(sessionRows))
		}
	})
}

func TestGetProjectDailySessions_CrossMidnight(t *testing.T) {
	// テストデータベースの準備
	db, _ := setupTestDB(t)
	defer db.Close()

	// テストプロジェクト作成
	projectID, err := db.CreateProject("test-project", "/path/to/project")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// 跨日セッションのテストケース
	sessions := []struct {
		id           string
		branch       string
		startTime    time.Time
		endTime      time.Time
		inputTokens  int
		outputTokens int
	}{
		{
			// 1/20 23:00 → 1/21 02:00 (跨日セッション)
			id:           "session-cross-midnight",
			branch:       "main",
			startTime:    time.Date(2026, 1, 20, 23, 0, 0, 0, time.UTC),
			endTime:      time.Date(2026, 1, 21, 2, 0, 0, 0, time.UTC),
			inputTokens:  1000,
			outputTokens: 500,
		},
		{
			// 1/20 10:00 → 1/20 15:00 (同日セッション)
			id:           "session-same-day",
			branch:       "feature-a",
			startTime:    time.Date(2026, 1, 20, 10, 0, 0, 0, time.UTC),
			endTime:      time.Date(2026, 1, 20, 15, 0, 0, 0, time.UTC),
			inputTokens:  2000,
			outputTokens: 1000,
		},
		{
			// 1/20 10:00 → 1/22 15:00 (複数日にまたがるセッション)
			id:           "session-multi-day",
			branch:       "feature-b",
			startTime:    time.Date(2026, 1, 20, 10, 0, 0, 0, time.UTC),
			endTime:      time.Date(2026, 1, 22, 15, 0, 0, 0, time.UTC),
			inputTokens:  3000,
			outputTokens: 1500,
		},
		{
			// 1/20 23:59:59 → 1/21 00:00:01 (日付境界ぴったり)
			id:           "session-boundary",
			branch:       "main",
			startTime:    time.Date(2026, 1, 20, 23, 59, 59, 0, time.UTC),
			endTime:      time.Date(2026, 1, 21, 0, 0, 1, 0, time.UTC),
			inputTokens:  500,
			outputTokens: 250,
		},
	}

	for _, s := range sessions {
		duration := int(s.endTime.Sub(s.startTime).Seconds())
		_, err := db.conn.Exec(`
			INSERT INTO sessions (
				id, project_id, git_branch, start_time, end_time, duration_seconds,
				total_input_tokens, total_output_tokens,
				total_cache_creation_tokens, total_cache_read_tokens,
				error_count, first_user_message
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, s.id, projectID, s.branch,
			s.startTime.Format(time.RFC3339Nano),
			s.endTime.Format(time.RFC3339Nano),
			duration,
			s.inputTokens, s.outputTokens, 0, 0, 0, "test message")
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
	}

	t.Run("跨日セッションが両日に含まれる", func(t *testing.T) {
		// 1/20の統計
		sessions20, err := db.GetProjectDailySessions(projectID, "2026-01-20")
		if err != nil {
			t.Fatalf("GetProjectDailySessions failed: %v", err)
		}

		// 1/20には4つのセッションが含まれるはず
		// - session-same-day (同日)
		// - session-cross-midnight (跨日)
		// - session-multi-day (複数日)
		// - session-boundary (境界)
		if len(sessions20) != 4 {
			t.Errorf("Expected 4 sessions on 2026-01-20, got %d", len(sessions20))
		}

		// 1/21の統計
		sessions21, err := db.GetProjectDailySessions(projectID, "2026-01-21")
		if err != nil {
			t.Fatalf("GetProjectDailySessions failed: %v", err)
		}

		// 1/21には3つのセッションが含まれるはず
		// - session-cross-midnight (跨日)
		// - session-multi-day (複数日)
		// - session-boundary (境界)
		if len(sessions21) != 3 {
			t.Errorf("Expected 3 sessions on 2026-01-21, got %d", len(sessions21))
		}

		// セッションIDの確認
		sessionIDs21 := make(map[string]bool)
		for _, s := range sessions21 {
			sessionIDs21[s.ID] = true
		}

		if !sessionIDs21["session-cross-midnight"] {
			t.Error("session-cross-midnight should be included on 2026-01-21")
		}
		if !sessionIDs21["session-multi-day"] {
			t.Error("session-multi-day should be included on 2026-01-21")
		}
		if !sessionIDs21["session-boundary"] {
			t.Error("session-boundary should be included on 2026-01-21")
		}
		if sessionIDs21["session-same-day"] {
			t.Error("session-same-day should NOT be included on 2026-01-21")
		}
	})

	t.Run("複数日にまたがるセッションがすべての日に含まれる", func(t *testing.T) {
		// 1/22の統計
		sessions22, err := db.GetProjectDailySessions(projectID, "2026-01-22")
		if err != nil {
			t.Fatalf("GetProjectDailySessions failed: %v", err)
		}

		// 1/22には1つのセッションが含まれるはず
		// - session-multi-day (複数日)
		if len(sessions22) != 1 {
			t.Errorf("Expected 1 session on 2026-01-22, got %d", len(sessions22))
		}

		if sessions22[0].ID != "session-multi-day" {
			t.Errorf("Expected session-multi-day on 2026-01-22, got %s", sessions22[0].ID)
		}
	})

	t.Run("同日セッションは該当日のみに含まれる", func(t *testing.T) {
		// 1/21でsession-same-dayが含まれないことを確認
		sessions21, err := db.GetProjectDailySessions(projectID, "2026-01-21")
		if err != nil {
			t.Fatalf("GetProjectDailySessions failed: %v", err)
		}

		for _, s := range sessions21 {
			if s.ID == "session-same-day" {
				t.Error("session-same-day should NOT be included on 2026-01-21")
			}
		}
	})
}
