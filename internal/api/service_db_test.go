package api

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/a-tak/ccloganalysis/internal/db"
	"github.com/a-tak/ccloganalysis/internal/parser"
)

// setupTestDBService creates a test database service
func setupTestDBService(t *testing.T) (*DatabaseSessionService, *db.DB) {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	database, err := db.NewDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// parserはnil（Analyze機能を使わない場合）
	service := NewDatabaseSessionService(database, nil)
	return service, database
}

// createTestData creates test projects and sessions in the database
func createTestData(t *testing.T, database *db.DB) {
	t.Helper()

	// プロジェクト1作成
	project1Name := "test-project-1"
	project1Path := "/path/to/test-project-1"
	_, err := database.CreateProject(project1Name, project1Path)
	if err != nil {
		t.Fatalf("Failed to create project1: %v", err)
	}

	// プロジェクト2作成
	project2Name := "test-project-2"
	project2Path := "/path/to/test-project-2"
	_, err = database.CreateProject(project2Name, project2Path)
	if err != nil {
		t.Fatalf("Failed to create project2: %v", err)
	}

	// セッション作成（プロジェクト1に2つ）
	now := time.Now().Truncate(time.Second)
	session1 := &parser.Session{
		ID:          "session-1",
		ProjectPath: project1Path,
		GitBranch:   "main",
		StartTime:   now.Add(-2 * time.Hour),
		EndTime:     now.Add(-1 * time.Hour),
		Entries: []parser.LogEntry{
			{
				Type:      "user",
				Timestamp: now.Add(-2 * time.Hour),
				SessionID: "session-1",
				UUID:      "entry-1-1",
				Cwd:       project1Path,
				GitBranch: "main",
				Message: &parser.Message{
					Model: "claude-sonnet-4-5",
					Role:  "user",
					Content: []parser.Content{
						{Type: "text", Text: "Hello"},
					},
				},
			},
		},
		TotalTokens: parser.TokenSummary{
			InputTokens:              100,
			OutputTokens:             50,
			CacheCreationInputTokens: 0,
			CacheReadInputTokens:     0,
		},
		ModelUsage: map[string]parser.TokenSummary{
			"claude-sonnet-4-5": {
				InputTokens:  100,
				OutputTokens: 50,
			},
		},
		ToolCalls: []parser.ToolCall{
			{
				Timestamp: now.Add(-90 * time.Minute),
				Name:      "read_file",
				Input:     map[string]interface{}{"path": "/test.go"},
				IsError:   false,
				Result:    "file content",
			},
		},
		ErrorCount: 0,
	}

	err = database.CreateSession(session1, project1Name)
	if err != nil {
		t.Fatalf("Failed to create session1: %v", err)
	}

	session2 := &parser.Session{
		ID:          "session-2",
		ProjectPath: project1Path,
		GitBranch:   "feature-branch",
		StartTime:   now.Add(-1 * time.Hour),
		EndTime:     now,
		Entries: []parser.LogEntry{
			{
				Type:      "user",
				Timestamp: now.Add(-1 * time.Hour),
				SessionID: "session-2",
				UUID:      "entry-2-1",
				Cwd:       project1Path,
				GitBranch: "feature-branch",
				Message: &parser.Message{
					Model: "claude-sonnet-4-5",
					Role:  "user",
					Content: []parser.Content{
						{Type: "text", Text: "Test"},
					},
				},
			},
		},
		TotalTokens: parser.TokenSummary{
			InputTokens:  200,
			OutputTokens: 100,
		},
		ModelUsage: map[string]parser.TokenSummary{
			"claude-sonnet-4-5": {
				InputTokens:  200,
				OutputTokens: 100,
			},
		},
		ToolCalls:  []parser.ToolCall{},
		ErrorCount: 1,
	}

	err = database.CreateSession(session2, project1Name)
	if err != nil {
		t.Fatalf("Failed to create session2: %v", err)
	}

	// セッション作成（プロジェクト2に1つ）
	session3 := &parser.Session{
		ID:          "session-3",
		ProjectPath: project2Path,
		GitBranch:   "main",
		StartTime:   now.Add(-30 * time.Minute),
		EndTime:     now,
		Entries: []parser.LogEntry{
			{
				Type:      "user",
				Timestamp: now.Add(-30 * time.Minute),
				SessionID: "session-3",
				UUID:      "entry-3-1",
				Cwd:       project2Path,
				GitBranch: "main",
				Message: &parser.Message{
					Model: "claude-opus-4-5",
					Role:  "user",
					Content: []parser.Content{
						{Type: "text", Text: "Test project 2"},
					},
				},
			},
		},
		TotalTokens: parser.TokenSummary{
			InputTokens:  50,
			OutputTokens: 25,
		},
		ModelUsage: map[string]parser.TokenSummary{
			"claude-opus-4-5": {
				InputTokens:  50,
				OutputTokens: 25,
			},
		},
		ToolCalls:  []parser.ToolCall{},
		ErrorCount: 0,
	}

	err = database.CreateSession(session3, project2Name)
	if err != nil {
		t.Fatalf("Failed to create session3: %v", err)
	}
}

func TestDatabaseSessionService_ListProjects(t *testing.T) {
	service, database := setupTestDBService(t)
	defer database.Close()

	t.Run("空のリストを返す", func(t *testing.T) {
		projects, err := service.ListProjects()
		if err != nil {
			t.Fatalf("ListProjects failed: %v", err)
		}

		if len(projects) != 0 {
			t.Errorf("Expected 0 projects, got %d", len(projects))
		}
	})

	t.Run("プロジェクト一覧を返す", func(t *testing.T) {
		// テストデータ作成
		createTestData(t, database)

		projects, err := service.ListProjects()
		if err != nil {
			t.Fatalf("ListProjects failed: %v", err)
		}

		if len(projects) != 2 {
			t.Fatalf("Expected 2 projects, got %d", len(projects))
		}

		// プロジェクト1を検証
		project1 := projects[0]
		if project1.Name != "test-project-1" {
			t.Errorf("Expected name=test-project-1, got %s", project1.Name)
		}
		if project1.DecodedPath != "/path/to/test-project-1" {
			t.Errorf("Expected path=/path/to/test-project-1, got %s", project1.DecodedPath)
		}
		if project1.SessionCount != 2 {
			t.Errorf("Expected 2 sessions, got %d", project1.SessionCount)
		}

		// プロジェクト2を検証
		project2 := projects[1]
		if project2.Name != "test-project-2" {
			t.Errorf("Expected name=test-project-2, got %s", project2.Name)
		}
		if project2.SessionCount != 1 {
			t.Errorf("Expected 1 session, got %d", project2.SessionCount)
		}
	})
}

func TestDatabaseSessionService_ListSessions(t *testing.T) {
	service, database := setupTestDBService(t)
	defer database.Close()

	createTestData(t, database)

	t.Run("全プロジェクトのセッション一覧を返す", func(t *testing.T) {
		sessions, err := service.ListSessions("")
		if err != nil {
			t.Fatalf("ListSessions failed: %v", err)
		}

		if len(sessions) != 3 {
			t.Fatalf("Expected 3 sessions, got %d", len(sessions))
		}

		// 開始時刻の降順でソートされていることを確認
		for i := 0; i < len(sessions)-1; i++ {
			if sessions[i].StartTime.Before(sessions[i+1].StartTime) {
				t.Error("Sessions are not sorted by start time descending")
			}
		}
	})

	t.Run("特定プロジェクトのセッション一覧を返す", func(t *testing.T) {
		sessions, err := service.ListSessions("test-project-1")
		if err != nil {
			t.Fatalf("ListSessions failed: %v", err)
		}

		if len(sessions) != 2 {
			t.Fatalf("Expected 2 sessions, got %d", len(sessions))
		}

		// 全てのセッションがtest-project-1に属することを確認
		for _, session := range sessions {
			if session.ProjectName != "test-project-1" {
				t.Errorf("Expected project=test-project-1, got %s", session.ProjectName)
			}
		}
	})

	t.Run("存在しないプロジェクト名で空のリストを返す", func(t *testing.T) {
		sessions, err := service.ListSessions("non-existent-project")
		if err != nil {
			t.Fatalf("ListSessions failed: %v", err)
		}

		if len(sessions) != 0 {
			t.Errorf("Expected 0 sessions, got %d", len(sessions))
		}
	})
}

func TestDatabaseSessionService_GetSession(t *testing.T) {
	service, database := setupTestDBService(t)
	defer database.Close()

	createTestData(t, database)

	t.Run("セッション詳細を返す", func(t *testing.T) {
		session, err := service.GetSession("test-project-1", "session-1")
		if err != nil {
			t.Fatalf("GetSession failed: %v", err)
		}

		if session.ID != "session-1" {
			t.Errorf("Expected ID=session-1, got %s", session.ID)
		}
		if session.ProjectName != "test-project-1" {
			t.Errorf("Expected project=test-project-1, got %s", session.ProjectName)
		}
		if session.GitBranch != "main" {
			t.Errorf("Expected branch=main, got %s", session.GitBranch)
		}

		// トークン集計を検証
		if session.TotalTokens.InputTokens != 100 {
			t.Errorf("Expected input_tokens=100, got %d", session.TotalTokens.InputTokens)
		}
		if session.TotalTokens.OutputTokens != 50 {
			t.Errorf("Expected output_tokens=50, got %d", session.TotalTokens.OutputTokens)
		}
		if session.TotalTokens.TotalTokens != 150 {
			t.Errorf("Expected total_tokens=150, got %d", session.TotalTokens.TotalTokens)
		}

		// モデル使用量を検証
		if len(session.ModelUsage) != 1 {
			t.Fatalf("Expected 1 model, got %d", len(session.ModelUsage))
		}

		// ツール呼び出しを検証
		if len(session.ToolCalls) != 1 {
			t.Fatalf("Expected 1 tool call, got %d", len(session.ToolCalls))
		}

		// メッセージを検証
		if len(session.Messages) != 1 {
			t.Fatalf("Expected 1 message, got %d", len(session.Messages))
		}

		// Durationを検証
		if session.Duration == "" {
			t.Error("Duration should not be empty")
		}
	})

	t.Run("存在しないセッションIDでエラーを返す", func(t *testing.T) {
		_, err := service.GetSession("test-project-1", "non-existent-session")
		if err == nil {
			t.Error("Expected error for non-existent session, got nil")
		}
	})

	t.Run("存在しないプロジェクト名でエラーを返す", func(t *testing.T) {
		_, err := service.GetSession("non-existent-project", "session-1")
		if err == nil {
			t.Error("Expected error for non-existent project, got nil")
		}
	})
}

func TestDatabaseSessionService_Analyze(t *testing.T) {
	service, database := setupTestDBService(t)
	defer database.Close()

	t.Run("Analyzeはまだ実装されていない", func(t *testing.T) {
		// Sprint 4で実装予定
		// 現時点ではエラーを返すか、no-op実装
		result, err := service.Analyze(nil)

		// 実装がない場合、エラーかno-op結果を期待
		if err != nil {
			// エラーを返す実装の場合
			t.Logf("Analyze not yet implemented: %v", err)
		} else if result != nil {
			// no-op実装の場合
			t.Logf("Analyze returned: %+v", result)
		}
	})
}
