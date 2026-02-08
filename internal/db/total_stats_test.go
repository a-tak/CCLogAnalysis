package db

import (
	"testing"
	"time"
)

// Note: GetTotalStats, GetTotalTimeSeriesStats, GetDailyGroupStats are
// tested indirectly through their usage in the service layer tests
// (internal/api/router_test.go). These are placeholder tests that
// demonstrate the testing structure and can be expanded with more
// comprehensive test data.

func TestGetTotalStatsBasic(t *testing.T) {
	t.Run("正常系：セッション数ゼロの場合", func(t *testing.T) {
		db, _ := setupTestDB(t)
		defer db.Close()

		stats, err := db.GetTotalStats()

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if stats == nil {
			t.Fatal("Expected stats to be non-nil")
		}

		// Empty database should return zero values
		if stats.TotalSessions != 0 {
			t.Errorf("Expected 0 sessions, got %d", stats.TotalSessions)
		}

		totalTokens := stats.TotalInputTokens + stats.TotalOutputTokens + stats.TotalCacheCreationTokens + stats.TotalCacheReadTokens
		if totalTokens != 0 {
			t.Errorf("Expected 0 total tokens, got %d", totalTokens)
		}

		if !stats.FirstSession.IsZero() {
			t.Error("Expected FirstSession to be zero value for empty DB")
		}
	})

	t.Run("エッジケース：NULL値処理", func(t *testing.T) {
		db, _ := setupTestDB(t)
		defer db.Close()

		// Get stats from empty database - should not error on NULL handling
		stats, err := db.GetTotalStats()

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify error rate is valid (should be 0 for empty DB)
		if stats.ErrorRate < 0 || stats.ErrorRate > 1 {
			t.Errorf("Expected ErrorRate between 0 and 1, got %f", stats.ErrorRate)
		}
	})
}

func TestGetTotalTimeSeriesStatsBasic(t *testing.T) {
	t.Run("正常系：データなし", func(t *testing.T) {
		db, _ := setupTestDB(t)
		defer db.Close()

		stats, err := db.GetTotalTimeSeriesStats("day", 10)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(stats) != 0 {
			t.Errorf("Expected empty result for empty DB, got %d items", len(stats))
		}
	})

	t.Run("パラメータ検証：期間の種類", func(t *testing.T) {
		db, _ := setupTestDB(t)
		defer db.Close()

		testCases := []string{"day", "week", "month"}

		for _, period := range testCases {
			stats, err := db.GetTotalTimeSeriesStats(period, 10)

			if err != nil {
				t.Errorf("Period %s should not error, got %v", period, err)
			}

			// Result can be nil or empty slice for empty DB, both are valid
			if stats != nil && len(stats) > 10 {
				t.Errorf("Period %s returned more than limit: got %d items", period, len(stats))
			}
		}
	})

	t.Run("パラメータ検証：limit", func(t *testing.T) {
		db, _ := setupTestDB(t)
		defer db.Close()

		// Test with various limit values
		testCases := []int{1, 10, 30, 100}

		for _, limit := range testCases {
			stats, err := db.GetTotalTimeSeriesStats("day", limit)

			if err != nil {
				t.Errorf("Limit %d should not error, got %v", limit, err)
			}

			// Result length should not exceed limit
			if len(stats) > limit {
				t.Errorf("Expected at most %d items, got %d", limit, len(stats))
			}
		}
	})
}

func TestGetDailyGroupStatsBasic(t *testing.T) {
	t.Run("正常系：日付フォーマット検証", func(t *testing.T) {
		db, _ := setupTestDB(t)
		defer db.Close()

		testDate := "2026-01-26"
		stats, err := db.GetDailyGroupStats(testDate)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Empty DB returns empty result, which is valid
		// No assertion needed - test passes if no error occurs
		if stats != nil && len(stats) > 0 {
			t.Logf("Found %d groups for test date", len(stats))
		}
	})

	t.Run("データ検証：構造正確性", func(t *testing.T) {
		db, _ := setupTestDB(t)
		defer db.Close()

		testDate := "2026-01-26"
		stats, err := db.GetDailyGroupStats(testDate)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Validate structure of returned stats
		for _, stat := range stats {
			// GroupID should be non-negative
			if stat.GroupID < 0 {
				t.Errorf("Expected non-negative GroupID, got %d", stat.GroupID)
			}

			// Token counts should be non-negative
			if stat.TotalInputTokens < 0 {
				t.Errorf("Expected non-negative TotalInputTokens, got %d", stat.TotalInputTokens)
			}

			if stat.TotalOutputTokens < 0 {
				t.Errorf("Expected non-negative TotalOutputTokens, got %d", stat.TotalOutputTokens)
			}

			totalTokens := stat.TotalInputTokens + stat.TotalOutputTokens + stat.TotalCacheCreationTokens + stat.TotalCacheReadTokens
			if totalTokens < 0 {
				t.Errorf("Expected non-negative TotalTokens, got %d", totalTokens)
			}

			// SessionCount should be non-negative
			if stat.SessionCount < 0 {
				t.Errorf("Expected non-negative SessionCount, got %d", stat.SessionCount)
			}
		}
	})

	t.Run("時刻範囲検証", func(t *testing.T) {
		db, _ := setupTestDB(t)
		defer db.Close()

		// Test various dates to ensure date filtering works
		testDates := []string{
			time.Now().Format("2006-01-02"),
			time.Now().Add(-24 * time.Hour).Format("2006-01-02"),
			time.Now().Add(-30 * 24 * time.Hour).Format("2006-01-02"),
		}

		for _, date := range testDates {
			_, err := db.GetDailyGroupStats(date)

			if err != nil {
				t.Errorf("Date %s should not error, got %v", date, err)
			}
		}
	})
}

func TestGetDailyGroupStats_CrossMidnight(t *testing.T) {
	// テストデータベースの準備
	db, _ := setupTestDB(t)
	defer db.Close()

	// テストグループ作成
	gitRoot := "Test Group"
	groupID, err := db.CreateProjectGroup("test-group", &gitRoot)
	if err != nil {
		t.Fatalf("Failed to create project group: %v", err)
	}

	// テストプロジェクト作成
	projectID, err := db.CreateProject("test-project", "/path/to/project")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// プロジェクトをグループに追加
	err = db.AddProjectToGroup(projectID, groupID)
	if err != nil {
		t.Fatalf("Failed to add project to group: %v", err)
	}

	// 跨日セッションを作成
	sessions := []struct {
		id           string
		startTime    time.Time
		endTime      time.Time
		inputTokens  int
		outputTokens int
	}{
		{
			// 1/20 23:00 → 1/21 02:00 (跨日セッション)
			id:           "session-cross",
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
		`, s.id, projectID, "main",
			s.startTime.Format(time.RFC3339Nano),
			s.endTime.Format(time.RFC3339Nano),
			duration,
			s.inputTokens, s.outputTokens, 0, 0, 0, "test message")
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
	}

	t.Run("跨日セッションが両日の統計に含まれる", func(t *testing.T) {
		// 1/20の統計
		stats20, err := db.GetDailyGroupStats("2026-01-20")
		if err != nil {
			t.Fatalf("GetDailyGroupStats failed: %v", err)
		}

		if len(stats20) != 1 {
			t.Fatalf("Expected 1 group on 2026-01-20, got %d", len(stats20))
		}

		// 1/20には2つのセッションが含まれる
		if stats20[0].SessionCount != 2 {
			t.Errorf("Expected 2 sessions on 2026-01-20, got %d", stats20[0].SessionCount)
		}

		// トークン数を確認
		expectedInput := 3000 // session-cross(1000) + session-same-day(2000)
		expectedOutput := 1500 // session-cross(500) + session-same-day(1000)
		if stats20[0].TotalInputTokens != expectedInput {
			t.Errorf("Expected %d input tokens on 2026-01-20, got %d", expectedInput, stats20[0].TotalInputTokens)
		}
		if stats20[0].TotalOutputTokens != expectedOutput {
			t.Errorf("Expected %d output tokens on 2026-01-20, got %d", expectedOutput, stats20[0].TotalOutputTokens)
		}

		// 1/21の統計
		stats21, err := db.GetDailyGroupStats("2026-01-21")
		if err != nil {
			t.Fatalf("GetDailyGroupStats failed: %v", err)
		}

		if len(stats21) != 1 {
			t.Fatalf("Expected 1 group on 2026-01-21, got %d", len(stats21))
		}

		// 1/21には1つのセッション(session-cross)が含まれる
		if stats21[0].SessionCount != 1 {
			t.Errorf("Expected 1 session on 2026-01-21, got %d", stats21[0].SessionCount)
		}

		// トークン数を確認(session-crossのみ)
		if stats21[0].TotalInputTokens != 1000 {
			t.Errorf("Expected 1000 input tokens on 2026-01-21, got %d", stats21[0].TotalInputTokens)
		}
		if stats21[0].TotalOutputTokens != 500 {
			t.Errorf("Expected 500 output tokens on 2026-01-21, got %d", stats21[0].TotalOutputTokens)
		}
	})

	t.Run("セッションが存在しない日は空配列を返す", func(t *testing.T) {
		stats22, err := db.GetDailyGroupStats("2026-01-22")
		if err != nil {
			t.Fatalf("GetDailyGroupStats failed: %v", err)
		}

		if len(stats22) != 0 {
			t.Errorf("Expected 0 groups on 2026-01-22, got %d", len(stats22))
		}
	})
}

func TestGetTotalTimeSeriesStats_CrossMidnight(t *testing.T) {
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
		timeSeriesStats, err := db.GetTotalTimeSeriesStats("day", 30)
		if err != nil {
			t.Fatalf("GetTotalTimeSeriesStats failed: %v", err)
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
		if day22.SessionCount != 1 {
			t.Errorf("Expected 1 session on 2026-01-22, got %d", day22.SessionCount)
		}
		if day22.TotalInputTokens != 3000 {
			t.Errorf("Expected 3000 input tokens on 2026-01-22, got %d", day22.TotalInputTokens)
		}
	})

	t.Run("週別集計で跨日セッションが正しく処理される", func(t *testing.T) {
		// 週別の時系列統計を取得
		weekStats, err := db.GetTotalTimeSeriesStats("week", 10)
		if err != nil {
			t.Fatalf("GetTotalTimeSeriesStats failed: %v", err)
		}

		// 全てのセッションが同じ週に含まれる
		if len(weekStats) != 1 {
			t.Fatalf("Expected 1 week, got %d", len(weekStats))
		}

		// 4つのセッションが含まれる
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
		monthStats, err := db.GetTotalTimeSeriesStats("month", 10)
		if err != nil {
			t.Fatalf("GetTotalTimeSeriesStats failed: %v", err)
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
