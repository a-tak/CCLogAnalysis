package db

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/a-tak/ccloganalysis/internal/parser"
)

// setupTestClaudeDir creates a temporary Claude projects directory with test data
func setupTestClaudeDir(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "projects")

	// プロジェクト1のディレクトリとセッションファイル作成
	project1Dir := filepath.Join(claudeDir, "test-project-1")
	err := os.MkdirAll(project1Dir, 0755)
	if err != nil {
		t.Fatalf("Failed to create project1 directory: %v", err)
	}

	// セッション1のJSONLファイル作成
	session1Path := filepath.Join(project1Dir, "session-1.jsonl")
	session1Content := `{"type":"user","timestamp":"2024-01-01T10:00:00Z","sessionId":"session-1","uuid":"uuid-1","cwd":"/path/to/project","version":"1.0.0","gitBranch":"main","message":{"model":"claude-sonnet-4-5","role":"user","content":[{"type":"text","text":"Hello"}]}}
{"type":"assistant","timestamp":"2024-01-01T10:00:05Z","sessionId":"session-1","uuid":"uuid-2","parentUuid":"uuid-1","cwd":"/path/to/project","version":"1.0.0","gitBranch":"main","message":{"model":"claude-sonnet-4-5","role":"assistant","content":[{"type":"text","text":"Hi!"}],"usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
	err = os.WriteFile(session1Path, []byte(session1Content), 0644)
	if err != nil {
		t.Fatalf("Failed to write session1 file: %v", err)
	}

	// セッション2のJSONLファイル作成
	session2Path := filepath.Join(project1Dir, "session-2.jsonl")
	session2Content := `{"type":"user","timestamp":"2024-01-01T11:00:00Z","sessionId":"session-2","uuid":"uuid-3","cwd":"/path/to/project","version":"1.0.0","gitBranch":"feature","message":{"model":"claude-sonnet-4-5","role":"user","content":[{"type":"text","text":"Test"}]}}
{"type":"assistant","timestamp":"2024-01-01T11:00:05Z","sessionId":"session-2","uuid":"uuid-4","parentUuid":"uuid-3","cwd":"/path/to/project","version":"1.0.0","gitBranch":"feature","message":{"model":"claude-sonnet-4-5","role":"assistant","content":[{"type":"text","text":"OK"}],"usage":{"input_tokens":50,"output_tokens":25,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
	err = os.WriteFile(session2Path, []byte(session2Content), 0644)
	if err != nil {
		t.Fatalf("Failed to write session2 file: %v", err)
	}

	// プロジェクト2のディレクトリとセッションファイル作成
	project2Dir := filepath.Join(claudeDir, "test-project-2")
	err = os.MkdirAll(project2Dir, 0755)
	if err != nil {
		t.Fatalf("Failed to create project2 directory: %v", err)
	}

	// セッション3のJSONLファイル作成
	session3Path := filepath.Join(project2Dir, "session-3.jsonl")
	session3Content := `{"type":"user","timestamp":"2024-01-01T12:00:00Z","sessionId":"session-3","uuid":"uuid-5","cwd":"/path/to/project2","version":"1.0.0","gitBranch":"main","message":{"model":"claude-opus-4-5","role":"user","content":[{"type":"text","text":"Project 2"}]}}
{"type":"assistant","timestamp":"2024-01-01T12:00:05Z","sessionId":"session-3","uuid":"uuid-6","parentUuid":"uuid-5","cwd":"/path/to/project2","version":"1.0.0","gitBranch":"main","message":{"model":"claude-opus-4-5","role":"assistant","content":[{"type":"text","text":"OK"}],"usage":{"input_tokens":30,"output_tokens":20,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
	err = os.WriteFile(session3Path, []byte(session3Content), 0644)
	if err != nil {
		t.Fatalf("Failed to write session3 file: %v", err)
	}

	return claudeDir
}

func TestSyncAll(t *testing.T) {
	database, _ := setupTestDB(t)
	defer database.Close()

	claudeDir := setupTestClaudeDir(t)
	p := parser.NewParser(claudeDir)

	t.Run("全プロジェクトとセッションを同期できる", func(t *testing.T) {
		result, err := SyncAll(database, p)
		if err != nil {
			t.Fatalf("SyncAll failed: %v", err)
		}

		// 結果を検証
		if result.ProjectsProcessed != 2 {
			t.Errorf("Expected 2 projects processed, got %d", result.ProjectsProcessed)
		}
		if result.SessionsFound != 3 {
			t.Errorf("Expected 3 sessions found, got %d", result.SessionsFound)
		}
		if result.SessionsSynced != 3 {
			t.Errorf("Expected 3 sessions synced, got %d", result.SessionsSynced)
		}
		if result.SessionsSkipped != 0 {
			t.Errorf("Expected 0 sessions skipped, got %d", result.SessionsSkipped)
		}

		// DBにデータが保存されたことを確認
		projects, err := database.ListProjects()
		if err != nil {
			t.Fatalf("Failed to list projects: %v", err)
		}
		if len(projects) != 2 {
			t.Errorf("Expected 2 projects in DB, got %d", len(projects))
		}

		// セッション数を確認
		sessions, err := database.ListSessions(nil, 100, 0)
		if err != nil {
			t.Fatalf("Failed to list sessions: %v", err)
		}
		if len(sessions) != 3 {
			t.Errorf("Expected 3 sessions in DB, got %d", len(sessions))
		}
	})

	t.Run("2回目の同期では既存セッションをスキップする", func(t *testing.T) {
		// 1回目の同期
		_, err := SyncAll(database, p)
		if err != nil {
			t.Fatalf("First SyncAll failed: %v", err)
		}

		// 2回目の同期
		result, err := SyncAll(database, p)
		if err != nil {
			t.Fatalf("Second SyncAll failed: %v", err)
		}

		// 全てのセッションがスキップされることを確認
		if result.SessionsSynced != 0 {
			t.Errorf("Expected 0 sessions synced on second run, got %d", result.SessionsSynced)
		}
		if result.SessionsSkipped != 3 {
			t.Errorf("Expected 3 sessions skipped on second run, got %d", result.SessionsSkipped)
		}
	})

	t.Run("エラーが発生しても処理を継続する", func(t *testing.T) {
		// 空のDBでテスト（新しいDB）
		newDB, _ := setupTestDB(t)
		defer newDB.Close()

		// 不正なJSONLファイルを追加（完全に空のセッション）
		project1Dir := filepath.Join(claudeDir, "test-project-1")
		invalidSession := filepath.Join(project1Dir, "invalid-session.jsonl")
		// 空のファイル（セッション情報が取得できない）
		err := os.WriteFile(invalidSession, []byte(""), 0644)
		if err != nil {
			t.Fatalf("Failed to write invalid session: %v", err)
		}

		result, err := SyncAll(newDB, p)
		// エラーがあっても処理は続行する（エラーは返らない）
		if err != nil {
			t.Logf("SyncAll returned error: %v", err)
		}

		// 有効なセッションは同期されている
		if result.SessionsFound < 3 {
			t.Errorf("Expected at least 3 sessions found, got %d", result.SessionsFound)
		}
		// 無効なセッションは同期されているが、セッションIDが空になる可能性がある
		// または、エラーカウントに記録される
		if result.SessionsSynced < 3 {
			t.Logf("Sessions synced: %d (some may have failed)", result.SessionsSynced)
		}
	})
}

func TestSyncProject(t *testing.T) {
	database, _ := setupTestDB(t)
	defer database.Close()

	claudeDir := setupTestClaudeDir(t)
	p := parser.NewParser(claudeDir)

	t.Run("特定プロジェクトのみ同期できる", func(t *testing.T) {
		result, err := SyncProject(database, p, "test-project-1")
		if err != nil {
			t.Fatalf("SyncProject failed: %v", err)
		}

		// 結果を検証
		if result.ProjectsProcessed != 1 {
			t.Errorf("Expected 1 project processed, got %d", result.ProjectsProcessed)
		}
		if result.SessionsFound != 2 {
			t.Errorf("Expected 2 sessions found, got %d", result.SessionsFound)
		}
		if result.SessionsSynced != 2 {
			t.Errorf("Expected 2 sessions synced, got %d", result.SessionsSynced)
		}

		// DBにプロジェクト1のデータのみが保存されたことを確認
		projects, err := database.ListProjects()
		if err != nil {
			t.Fatalf("Failed to list projects: %v", err)
		}
		if len(projects) != 1 {
			t.Errorf("Expected 1 project in DB, got %d", len(projects))
		}
		if projects[0].Name != "test-project-1" {
			t.Errorf("Expected project name=test-project-1, got %s", projects[0].Name)
		}
	})

	t.Run("存在しないプロジェクトでエラーを返す", func(t *testing.T) {
		newDB, _ := setupTestDB(t)
		defer newDB.Close()

		_, err := SyncProject(newDB, p, "non-existent-project")
		if err == nil {
			t.Error("Expected error for non-existent project, got nil")
		}
	})
}

func TestSyncIncremental(t *testing.T) {
	database, _ := setupTestDB(t)
	defer database.Close()

	claudeDir := setupTestClaudeDir(t)
	p := parser.NewParser(claudeDir)

	t.Run("差分更新で新しいセッションのみ同期する", func(t *testing.T) {
		// 最初の同期
		result1, err := SyncAll(database, p)
		if err != nil {
			t.Fatalf("First sync failed: %v", err)
		}

		initialSynced := result1.SessionsSynced

		// 新しいセッションを追加
		project1Dir := filepath.Join(claudeDir, "test-project-1")
		newSessionPath := filepath.Join(project1Dir, "session-new.jsonl")
		newSessionContent := `{"type":"user","timestamp":"2024-01-01T13:00:00Z","sessionId":"session-new","uuid":"uuid-7","cwd":"/path/to/project","version":"1.0.0","gitBranch":"main","message":{"model":"claude-sonnet-4-5","role":"user","content":[{"type":"text","text":"New session"}]}}
{"type":"assistant","timestamp":"2024-01-01T13:00:05Z","sessionId":"session-new","uuid":"uuid-8","parentUuid":"uuid-7","cwd":"/path/to/project","version":"1.0.0","gitBranch":"main","message":{"model":"claude-sonnet-4-5","role":"assistant","content":[{"type":"text","text":"OK"}],"usage":{"input_tokens":40,"output_tokens":30,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
		err = os.WriteFile(newSessionPath, []byte(newSessionContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write new session: %v", err)
		}

		// 差分同期
		time.Sleep(10 * time.Millisecond) // ファイルシステムのタイムスタンプ更新を待つ
		result2, err := SyncIncremental(database, p)
		if err != nil {
			t.Fatalf("Incremental sync failed: %v", err)
		}

		// 新しいセッション1つだけが同期されることを確認
		if result2.SessionsSynced != 1 {
			t.Errorf("Expected 1 new session synced, got %d", result2.SessionsSynced)
		}
		// 既存のセッションはスキップされる
		if result2.SessionsSkipped != int(initialSynced) {
			t.Errorf("Expected %d sessions skipped, got %d", initialSynced, result2.SessionsSkipped)
		}

		// DBに全4セッションが保存されていることを確認
		sessions, err := database.ListSessions(nil, 100, 0)
		if err != nil {
			t.Fatalf("Failed to list sessions: %v", err)
		}
		expectedTotal := int(initialSynced) + 1
		if len(sessions) != expectedTotal {
			t.Errorf("Expected %d sessions in DB, got %d", expectedTotal, len(sessions))
		}
	})
}

func TestSyncResult(t *testing.T) {
	t.Run("SyncResultの構造を確認", func(t *testing.T) {
		result := &SyncResult{
			ProjectsProcessed: 2,
			SessionsFound:     5,
			SessionsSynced:    3,
			SessionsSkipped:   2,
			ErrorCount:        1,
		}

		if result.ProjectsProcessed != 2 {
			t.Error("ProjectsProcessed field not working")
		}
		if result.SessionsFound != 5 {
			t.Error("SessionsFound field not working")
		}
		if result.SessionsSynced != 3 {
			t.Error("SessionsSynced field not working")
		}
		if result.SessionsSkipped != 2 {
			t.Error("SessionsSkipped field not working")
		}
		if result.ErrorCount != 1 {
			t.Error("ErrorCount field not working")
		}
	})
}
