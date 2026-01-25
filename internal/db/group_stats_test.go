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
