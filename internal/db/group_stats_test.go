package db

import (
	"testing"
	"time"
)

// stringPtr はテスト用のヘルパー関数で、文字列のポインタを返す
func stringPtr2(s string) *string {
	return &s
}

func TestGetGroupStats(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	t.Run("グループ全体の統計を取得できる", func(t *testing.T) {
		// グループを作成
		groupID, err := db.CreateProjectGroup("test-group", stringPtr2("/path/to/repo.git"))
		if err != nil {
			t.Fatalf("CreateProjectGroup failed: %v", err)
		}

		// 複数のプロジェクトを作成してグループに追加
		project1ID, err := db.CreateProject("project-1", "/path/to/project1")
		if err != nil {
			t.Fatalf("CreateProject failed: %v", err)
		}
		project2ID, err := db.CreateProject("project-2", "/path/to/project2")
		if err != nil {
			t.Fatalf("CreateProject failed: %v", err)
		}

		err = db.AddProjectToGroup(project1ID, groupID)
		if err != nil {
			t.Fatalf("AddProjectToGroup failed: %v", err)
		}
		err = db.AddProjectToGroup(project2ID, groupID)
		if err != nil {
			t.Fatalf("AddProjectToGroup failed: %v", err)
		}

		// 各プロジェクトにセッションを作成
		now := time.Now()
		sessions := []struct {
			id          string
			projectID   int64
			inputTokens int
			outputTokens int
			errorCount  int
		}{
			{"session-1", project1ID, 1000, 500, 0},
			{"session-2", project1ID, 1500, 750, 1},
			{"session-3", project2ID, 2000, 1000, 0},
		}

		for _, s := range sessions {
			_, err := db.conn.Exec(`
				INSERT INTO sessions (
					id, project_id, git_branch, start_time, end_time, duration_seconds,
					total_input_tokens, total_output_tokens,
					total_cache_creation_tokens, total_cache_read_tokens,
					error_count
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, s.id, s.projectID, "main",
				now.Format(time.RFC3339Nano),
				now.Add(time.Hour).Format(time.RFC3339Nano),
				3600,
				s.inputTokens, s.outputTokens, 0, 0, s.errorCount)
			if err != nil {
				t.Fatalf("Failed to create session: %v", err)
			}
		}

		// グループ統計を取得
		stats, err := db.GetGroupStats(groupID)
		if err != nil {
			t.Fatalf("GetGroupStats failed: %v", err)
		}

		// 検証
		if stats.TotalProjects != 2 {
			t.Errorf("Expected 2 projects, got %d", stats.TotalProjects)
		}
		if stats.TotalSessions != 3 {
			t.Errorf("Expected 3 sessions, got %d", stats.TotalSessions)
		}

		expectedInputTokens := 1000 + 1500 + 2000
		if stats.TotalInputTokens != expectedInputTokens {
			t.Errorf("Expected %d input tokens, got %d", expectedInputTokens, stats.TotalInputTokens)
		}

		expectedOutputTokens := 500 + 750 + 1000
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
	})

	t.Run("プロジェクトがないグループで統計がゼロになる", func(t *testing.T) {
		// 空のグループを作成
		groupID, err := db.CreateProjectGroup("empty-group", stringPtr2("/path/to/empty.git"))
		if err != nil {
			t.Fatalf("CreateProjectGroup failed: %v", err)
		}

		// グループ統計を取得
		stats, err := db.GetGroupStats(groupID)
		if err != nil {
			t.Fatalf("GetGroupStats failed: %v", err)
		}

		// ゼロ値であることを確認
		if stats.TotalProjects != 0 {
			t.Errorf("Expected 0 projects, got %d", stats.TotalProjects)
		}
		if stats.TotalSessions != 0 {
			t.Errorf("Expected 0 sessions, got %d", stats.TotalSessions)
		}
		if stats.TotalInputTokens != 0 {
			t.Errorf("Expected 0 input tokens, got %d", stats.TotalInputTokens)
		}
		if stats.ErrorRate != 0 {
			t.Errorf("Expected 0 error rate, got %.2f", stats.ErrorRate)
		}
	})

	t.Run("セッションがないプロジェクトのみのグループで統計がゼロになる", func(t *testing.T) {
		// グループを作成
		groupID, err := db.CreateProjectGroup("no-session-group", stringPtr2("/path/to/no-session.git"))
		if err != nil {
			t.Fatalf("CreateProjectGroup failed: %v", err)
		}

		// セッションがないプロジェクトを作成
		projectID, err := db.CreateProject("no-session-project", "/path/to/no-session")
		if err != nil {
			t.Fatalf("CreateProject failed: %v", err)
		}

		err = db.AddProjectToGroup(projectID, groupID)
		if err != nil {
			t.Fatalf("AddProjectToGroup failed: %v", err)
		}

		// グループ統計を取得
		stats, err := db.GetGroupStats(groupID)
		if err != nil {
			t.Fatalf("GetGroupStats failed: %v", err)
		}

		// プロジェクト数は1、セッション数は0
		if stats.TotalProjects != 1 {
			t.Errorf("Expected 1 project, got %d", stats.TotalProjects)
		}
		if stats.TotalSessions != 0 {
			t.Errorf("Expected 0 sessions, got %d", stats.TotalSessions)
		}
	})
}

func TestGetGroupTimeSeriesStats(t *testing.T) {
	// テストデータベースの準備
	db, _ := setupTestDB(t)
	defer db.Close()

	// グループを作成
	groupID, err := db.CreateProjectGroup("test-group", stringPtr2("/path/to/repo.git"))
	if err != nil {
		t.Fatalf("CreateProjectGroup failed: %v", err)
	}

	// 複数のプロジェクトを作成してグループに追加
	project1ID, err := db.CreateProject("project-1", "/path/to/project1")
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}
	project2ID, err := db.CreateProject("project-2", "/path/to/project2")
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	err = db.AddProjectToGroup(project1ID, groupID)
	if err != nil {
		t.Fatalf("AddProjectToGroup failed: %v", err)
	}
	err = db.AddProjectToGroup(project2ID, groupID)
	if err != nil {
		t.Fatalf("AddProjectToGroup failed: %v", err)
	}

	// 異なる日付とプロジェクトでセッションを作成
	sessions := []struct {
		id        string
		projectID int64
		startTime time.Time
		tokens    int
	}{
		{"session-1", project1ID, time.Date(2026, 1, 20, 10, 0, 0, 0, time.UTC), 1000},
		{"session-2", project1ID, time.Date(2026, 1, 20, 14, 0, 0, 0, time.UTC), 1500},
		{"session-3", project2ID, time.Date(2026, 1, 20, 16, 0, 0, 0, time.UTC), 800},
		{"session-4", project1ID, time.Date(2026, 1, 21, 10, 0, 0, 0, time.UTC), 2000},
		{"session-5", project2ID, time.Date(2026, 1, 22, 10, 0, 0, 0, time.UTC), 500},
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
		`, s.id, s.projectID, "main",
			s.startTime.Format(time.RFC3339Nano),
			endTime.Format(time.RFC3339Nano),
			3600,
			s.tokens, 0, 0, 0, 0)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
	}

	t.Run("グループ全体の日別時系列統計を取得できる", func(t *testing.T) {
		// 日別の時系列統計を取得
		timeSeriesStats, err := db.GetGroupTimeSeriesStats(groupID, "day", 30)
		if err != nil {
			t.Fatalf("GetGroupTimeSeriesStats failed: %v", err)
		}

		// 3日分のデータがあることを確認
		if len(timeSeriesStats) != 3 {
			t.Fatalf("Expected 3 days, got %d", len(timeSeriesStats))
		}

		// 降順ソート（新しい順）なので、最初は2026-01-22
		day1 := timeSeriesStats[0]
		expectedDate1 := time.Date(2026, 1, 22, 0, 0, 0, 0, time.UTC)
		if !day1.PeriodStart.Equal(expectedDate1) {
			t.Errorf("Expected period start %v, got %v", expectedDate1, day1.PeriodStart)
		}
		if day1.SessionCount != 1 {
			t.Errorf("Expected 1 session on 2026-01-22, got %d", day1.SessionCount)
		}
		if day1.TotalInputTokens != 500 {
			t.Errorf("Expected 500 tokens on 2026-01-22, got %d", day1.TotalInputTokens)
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

		// 3番目は2026-01-20（複数プロジェクトのセッションが集計される）
		day3 := timeSeriesStats[2]
		expectedDate3 := time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC)
		if !day3.PeriodStart.Equal(expectedDate3) {
			t.Errorf("Expected period start %v, got %v", expectedDate3, day3.PeriodStart)
		}
		if day3.SessionCount != 3 {
			t.Errorf("Expected 3 sessions on 2026-01-20, got %d", day3.SessionCount)
		}
		expectedTokens := 1000 + 1500 + 800
		if day3.TotalInputTokens != expectedTokens {
			t.Errorf("Expected %d tokens on 2026-01-20, got %d", expectedTokens, day3.TotalInputTokens)
		}
	})

	t.Run("期間パラメータで集計単位を変更できる", func(t *testing.T) {
		// 週別の時系列統計を取得（全て同じ週なので1件になる）
		weekStats, err := db.GetGroupTimeSeriesStats(groupID, "week", 30)
		if err != nil {
			t.Fatalf("GetGroupTimeSeriesStats with week failed: %v", err)
		}

		// 全てのセッションが同じ週なので1週分のデータ
		if len(weekStats) != 1 {
			t.Fatalf("Expected 1 week, got %d", len(weekStats))
		}

		// 全セッションが集計される
		if weekStats[0].SessionCount != 5 {
			t.Errorf("Expected 5 sessions in week, got %d", weekStats[0].SessionCount)
		}

		// 月別の時系列統計を取得（全て同じ月なので1件になる）
		monthStats, err := db.GetGroupTimeSeriesStats(groupID, "month", 30)
		if err != nil {
			t.Fatalf("GetGroupTimeSeriesStats with month failed: %v", err)
		}

		// 全てのセッションが同じ月なので1ヶ月分のデータ
		if len(monthStats) != 1 {
			t.Fatalf("Expected 1 month, got %d", len(monthStats))
		}

		// 全セッションが集計される
		if monthStats[0].SessionCount != 5 {
			t.Errorf("Expected 5 sessions in month, got %d", monthStats[0].SessionCount)
		}
	})

	t.Run("セッションがない場合は空配列を返す", func(t *testing.T) {
		// 空のグループを作成
		emptyGroupID, err := db.CreateProjectGroup("empty-group", stringPtr2("/path/to/empty.git"))
		if err != nil {
			t.Fatalf("CreateProjectGroup failed: %v", err)
		}

		// 時系列統計を取得
		timeSeriesStats, err := db.GetGroupTimeSeriesStats(emptyGroupID, "day", 30)
		if err != nil {
			t.Fatalf("GetGroupTimeSeriesStats failed: %v", err)
		}

		// 空配列が返ることを確認
		if len(timeSeriesStats) != 0 {
			t.Errorf("Expected 0 days, got %d", len(timeSeriesStats))
		}
	})

	t.Run("無効な期間パラメータでエラーを返す", func(t *testing.T) {
		_, err := db.GetGroupTimeSeriesStats(groupID, "invalid", 30)
		if err == nil {
			t.Error("Expected error for invalid period, got nil")
		}
	})
}

func TestGetGroupDailyProjectStats(t *testing.T) {
	// テストデータベースの準備
	db, _ := setupTestDB(t)
	defer db.Close()

	// グループを作成
	groupID, err := db.CreateProjectGroup("test-group", stringPtr2("/path/to/repo.git"))
	if err != nil {
		t.Fatalf("CreateProjectGroup failed: %v", err)
	}

	// 複数のプロジェクトを作成してグループに追加
	project1ID, err := db.CreateProject("project-1", "/path/to/project1")
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}
	project2ID, err := db.CreateProject("project-2", "/path/to/project2")
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}
	project3ID, err := db.CreateProject("project-3", "/path/to/project3")
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	err = db.AddProjectToGroup(project1ID, groupID)
	if err != nil {
		t.Fatalf("AddProjectToGroup failed: %v", err)
	}
	err = db.AddProjectToGroup(project2ID, groupID)
	if err != nil {
		t.Fatalf("AddProjectToGroup failed: %v", err)
	}
	err = db.AddProjectToGroup(project3ID, groupID)
	if err != nil {
		t.Fatalf("AddProjectToGroup failed: %v", err)
	}

	// 異なる日付でセッションを作成
	targetDate := time.Date(2026, 1, 25, 0, 0, 0, 0, time.UTC)
	otherDate := time.Date(2026, 1, 26, 0, 0, 0, 0, time.UTC)

	sessions := []struct {
		id           string
		projectID    int64
		startTime    time.Time
		inputTokens  int
		outputTokens int
		cacheCreate  int
		cacheRead    int
	}{
		// ターゲット日のセッション
		{"session-1", project1ID, targetDate.Add(10 * time.Hour), 1000, 500, 100, 50},
		{"session-2", project1ID, targetDate.Add(14 * time.Hour), 1500, 750, 150, 75},
		{"session-3", project2ID, targetDate.Add(16 * time.Hour), 2000, 1000, 200, 100},
		// 他の日のセッション（フィルタされるべき）
		{"session-4", project1ID, otherDate.Add(10 * time.Hour), 999, 999, 999, 999},
		// project-3はターゲット日にセッションなし
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
		`, s.id, s.projectID, "main",
			s.startTime.Format(time.RFC3339Nano),
			endTime.Format(time.RFC3339Nano),
			3600,
			s.inputTokens, s.outputTokens, s.cacheCreate, s.cacheRead, 0)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
	}

	t.Run("指定日のプロジェクト別統計を取得できる", func(t *testing.T) {
		// ターゲット日の統計を取得
		dateStr := targetDate.Format("2006-01-02")
		stats, err := db.GetGroupDailyProjectStats(groupID, dateStr)
		if err != nil {
			t.Fatalf("GetGroupDailyProjectStats failed: %v", err)
		}

		// 2プロジェクト分のデータが返されることを確認
		if len(stats) != 2 {
			t.Fatalf("Expected 2 projects, got %d", len(stats))
		}

		// トークン数が多い順（project-2 → project-1）
		// project-2
		if stats[0].ProjectName != "project-2" {
			t.Errorf("Expected project-2 first, got %s", stats[0].ProjectName)
		}
		if stats[0].SessionCount != 1 {
			t.Errorf("Expected 1 session for project-2, got %d", stats[0].SessionCount)
		}
		if stats[0].TotalInputTokens != 2000 {
			t.Errorf("Expected 2000 input tokens for project-2, got %d", stats[0].TotalInputTokens)
		}
		if stats[0].TotalOutputTokens != 1000 {
			t.Errorf("Expected 1000 output tokens for project-2, got %d", stats[0].TotalOutputTokens)
		}
		if stats[0].TotalCacheCreationTokens != 200 {
			t.Errorf("Expected 200 cache creation tokens for project-2, got %d", stats[0].TotalCacheCreationTokens)
		}
		if stats[0].TotalCacheReadTokens != 100 {
			t.Errorf("Expected 100 cache read tokens for project-2, got %d", stats[0].TotalCacheReadTokens)
		}
		expectedTotal := 2000 + 1000 + 200 + 100
		if stats[0].TotalTokens != expectedTotal {
			t.Errorf("Expected %d total tokens for project-2, got %d", expectedTotal, stats[0].TotalTokens)
		}

		// project-1
		if stats[1].ProjectName != "project-1" {
			t.Errorf("Expected project-1 second, got %s", stats[1].ProjectName)
		}
		if stats[1].SessionCount != 2 {
			t.Errorf("Expected 2 sessions for project-1, got %d", stats[1].SessionCount)
		}
		expectedInput := 1000 + 1500
		if stats[1].TotalInputTokens != expectedInput {
			t.Errorf("Expected %d input tokens for project-1, got %d", expectedInput, stats[1].TotalInputTokens)
		}
		expectedOutput := 500 + 750
		if stats[1].TotalOutputTokens != expectedOutput {
			t.Errorf("Expected %d output tokens for project-1, got %d", expectedOutput, stats[1].TotalOutputTokens)
		}
	})

	t.Run("セッションが存在しない日付で空配列を返す", func(t *testing.T) {
		// 存在しない日付
		noSessionDate := "2026-01-27"
		stats, err := db.GetGroupDailyProjectStats(groupID, noSessionDate)
		if err != nil {
			t.Fatalf("GetGroupDailyProjectStats failed: %v", err)
		}

		// 空配列が返ることを確認
		if len(stats) != 0 {
			t.Errorf("Expected 0 projects, got %d", len(stats))
		}
	})

	t.Run("存在しないグループIDでエラーを返さず空配列を返す", func(t *testing.T) {
		// 存在しないグループID
		dateStr := targetDate.Format("2006-01-02")
		stats, err := db.GetGroupDailyProjectStats(99999, dateStr)
		if err != nil {
			t.Fatalf("GetGroupDailyProjectStats failed: %v", err)
		}

		// 空配列が返ることを確認
		if len(stats) != 0 {
			t.Errorf("Expected 0 projects, got %d", len(stats))
		}
	})

	t.Run("不正な日付フォーマットでも処理できる", func(t *testing.T) {
		// SQLiteはDATE()関数でパースを試みるので、エラーにはならない可能性がある
		// 空配列が返ることを確認
		stats, err := db.GetGroupDailyProjectStats(groupID, "invalid-date")
		if err != nil {
			t.Fatalf("GetGroupDailyProjectStats failed: %v", err)
		}

		// 空配列が返ることを確認
		if len(stats) != 0 {
			t.Errorf("Expected 0 projects, got %d", len(stats))
		}
	})
}
