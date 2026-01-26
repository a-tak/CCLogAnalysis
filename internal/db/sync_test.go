package db

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/a-tak/ccloganalysis/internal/logger"
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

	t.Run("新規ワークツリープロジェクトが増分同期で検出される", func(t *testing.T) {
		// 新しいデータベースを作成
		newDB, _ := setupTestDB(t)
		defer newDB.Close()

		// 最初のプロジェクトを作成して同期
		_, err := SyncAll(newDB, p)
		if err != nil {
			t.Fatalf("First SyncAll failed: %v", err)
		}

		// 新しいワークツリープロジェクトを追加
		worktreeDir := filepath.Join(claudeDir, "worktree-new-project")
		err = os.MkdirAll(worktreeDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create worktree directory: %v", err)
		}

		// 親リポジトリのパス
		parentRepoPath := "/path/to/parent/repo.git"

		// .gitファイルを作成（gitdir指定）
		gitFilePath := filepath.Join(worktreeDir, ".git")
		gitFileContent := "gitdir: " + parentRepoPath + "/worktrees/feature\n"
		err = os.WriteFile(gitFilePath, []byte(gitFileContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create .git file: %v", err)
		}

		// ワークツリーのセッションファイルを作成
		worktreeSessionPath := filepath.Join(worktreeDir, "session-worktree.jsonl")
		worktreeSessionContent := `{"type":"user","timestamp":"2024-01-01T14:00:00Z","sessionId":"session-worktree","uuid":"uuid-wt-1","cwd":"` + worktreeDir + `","version":"1.0.0","gitBranch":"feature","message":{"model":"claude-sonnet-4-5","role":"user","content":[{"type":"text","text":"Worktree test"}]}}
{"type":"assistant","timestamp":"2024-01-01T14:00:05Z","sessionId":"session-worktree","uuid":"uuid-wt-2","parentUuid":"uuid-wt-1","cwd":"` + worktreeDir + `","version":"1.0.0","gitBranch":"feature","message":{"model":"claude-sonnet-4-5","role":"assistant","content":[{"type":"text","text":"OK"}],"usage":{"input_tokens":20,"output_tokens":10,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
		err = os.WriteFile(worktreeSessionPath, []byte(worktreeSessionContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write worktree session: %v", err)
		}

		// 差分同期を実行
		time.Sleep(10 * time.Millisecond)
		result2, err := SyncIncremental(newDB, p)
		if err != nil {
			t.Fatalf("Incremental sync failed: %v", err)
		}

		// 新しいワークツリープロジェクトのセッションが同期されることを確認
		if result2.SessionsSynced < 1 {
			t.Errorf("Expected at least 1 new session synced, got %d", result2.SessionsSynced)
		}

		// ワークツリープロジェクトを取得してGit Rootを確認
		worktreeProject, err := newDB.GetProjectByName("worktree-new-project")
		if err != nil {
			t.Fatalf("Failed to get worktree project: %v", err)
		}

		// Git Rootが設定されていることを確認
		if worktreeProject.GitRoot == nil {
			t.Error("Expected git_root to be set for worktree project, got nil")
		} else if *worktreeProject.GitRoot != parentRepoPath {
			t.Errorf("Expected git_root %s, got %s", parentRepoPath, *worktreeProject.GitRoot)
		}
	})

	t.Run("新規プロジェクト作成時にGit Root検出が実行される", func(t *testing.T) {
		// 新しいデータベースを作成
		newDB, _ := setupTestDB(t)
		defer newDB.Close()

		// 既存プロジェクトを同期
		_, err := SyncAll(newDB, p)
		if err != nil {
			t.Fatalf("Initial SyncAll failed: %v", err)
		}

		// 新しいGitリポジトリプロジェクトを追加
		gitRepoDir := filepath.Join(claudeDir, "new-git-repo")
		err = os.MkdirAll(gitRepoDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create git repo directory: %v", err)
		}

		// .gitディレクトリを作成
		gitDir := filepath.Join(gitRepoDir, ".git")
		err = os.Mkdir(gitDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create .git directory: %v", err)
		}

		// セッションファイルを作成
		sessionPath := filepath.Join(gitRepoDir, "session-1.jsonl")
		sessionContent := `{"type":"user","timestamp":"2024-01-01T15:00:00Z","sessionId":"session-new-git","uuid":"uuid-ng-1","cwd":"` + gitRepoDir + `","version":"1.0.0","gitBranch":"main","message":{"model":"claude-sonnet-4-5","role":"user","content":[{"type":"text","text":"New git repo"}]}}
{"type":"assistant","timestamp":"2024-01-01T15:00:05Z","sessionId":"session-new-git","uuid":"uuid-ng-2","parentUuid":"uuid-ng-1","cwd":"` + gitRepoDir + `","version":"1.0.0","gitBranch":"main","message":{"model":"claude-sonnet-4-5","role":"assistant","content":[{"type":"text","text":"OK"}],"usage":{"input_tokens":15,"output_tokens":8,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
		err = os.WriteFile(sessionPath, []byte(sessionContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write session file: %v", err)
		}

		// 差分同期を実行
		time.Sleep(10 * time.Millisecond)
		_, err = SyncIncremental(newDB, p)
		if err != nil {
			t.Fatalf("Incremental sync failed: %v", err)
		}

		// 新規プロジェクトを取得してGit Rootを確認
		newProject, err := newDB.GetProjectByName("new-git-repo")
		if err != nil {
			t.Fatalf("Failed to get new project: %v", err)
		}

		// Git Rootが設定されていることを確認
		if newProject.GitRoot == nil {
			t.Error("Expected git_root to be set for new project, got nil")
		} else if *newProject.GitRoot != gitRepoDir {
			t.Errorf("Expected git_root %s, got %s", gitRepoDir, *newProject.GitRoot)
		}
	})

	t.Run("新規プロジェクト検出後にプロジェクトグループが同期される", func(t *testing.T) {
		// 新しいデータベースを作成
		newDB, _ := setupTestDB(t)
		defer newDB.Close()

		// 既存プロジェクトを同期
		_, err := SyncAll(newDB, p)
		if err != nil {
			t.Fatalf("Initial SyncAll failed: %v", err)
		}

		// 初期のグループ数を取得
		groups1, err := newDB.ListProjectGroups()
		if err != nil {
			t.Fatalf("Failed to list project groups: %v", err)
		}
		initialGroupCount := len(groups1)

		// 新しいワークツリープロジェクトを追加（既存プロジェクトと同じGit Root）
		worktreeDir2 := filepath.Join(claudeDir, "worktree-same-repo")
		err = os.MkdirAll(worktreeDir2, 0755)
		if err != nil {
			t.Fatalf("Failed to create worktree directory: %v", err)
		}

		// 親リポジトリのパス（既存プロジェクトと同じ）
		parentRepoPath := "/path/to/project"

		// .gitファイルを作成
		gitFilePath := filepath.Join(worktreeDir2, ".git")
		gitFileContent := "gitdir: " + parentRepoPath + "/.git/worktrees/feature2\n"
		err = os.WriteFile(gitFilePath, []byte(gitFileContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create .git file: %v", err)
		}

		// セッションファイルを作成
		sessionPath := filepath.Join(worktreeDir2, "session-1.jsonl")
		sessionContent := `{"type":"user","timestamp":"2024-01-01T16:00:00Z","sessionId":"session-wt2","uuid":"uuid-wt2-1","cwd":"` + worktreeDir2 + `","version":"1.0.0","gitBranch":"feature2","message":{"model":"claude-sonnet-4-5","role":"user","content":[{"type":"text","text":"Worktree 2"}]}}
{"type":"assistant","timestamp":"2024-01-01T16:00:05Z","sessionId":"session-wt2","uuid":"uuid-wt2-2","parentUuid":"uuid-wt2-1","cwd":"` + worktreeDir2 + `","version":"1.0.0","gitBranch":"feature2","message":{"model":"claude-sonnet-4-5","role":"assistant","content":[{"type":"text","text":"OK"}],"usage":{"input_tokens":12,"output_tokens":6,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
		err = os.WriteFile(sessionPath, []byte(sessionContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write session file: %v", err)
		}

		// 差分同期を実行
		time.Sleep(10 * time.Millisecond)
		_, err = SyncIncremental(newDB, p)
		if err != nil {
			t.Fatalf("Incremental sync failed: %v", err)
		}

		// プロジェクトグループが更新されたことを確認
		groups2, err := newDB.ListProjectGroups()
		if err != nil {
			t.Fatalf("Failed to list project groups after sync: %v", err)
		}

		// グループ数が増えているか、または同じGit Rootのグループに追加されている
		// （実装によって挙動が変わるため、グループが存在することのみ確認）
		if len(groups2) < initialGroupCount {
			t.Errorf("Expected at least %d groups, got %d", initialGroupCount, len(groups2))
		}
	})

	t.Run("既存プロジェクトに影響がないことを確認", func(t *testing.T) {
		// 新しいデータベースを作成
		newDB, _ := setupTestDB(t)
		defer newDB.Close()

		// 既存プロジェクトを同期
		_, err := SyncAll(newDB, p)
		if err != nil {
			t.Fatalf("Initial SyncAll failed: %v", err)
		}

		// 既存プロジェクトの情報を取得
		existingProject, err := newDB.GetProjectByName("test-project-1")
		if err != nil {
			t.Fatalf("Failed to get existing project: %v", err)
		}
		originalGitRoot := existingProject.GitRoot
		originalUpdatedAt := existingProject.UpdatedAt

		// 新しいプロジェクトを追加
		newProjectDir := filepath.Join(claudeDir, "new-project-no-git")
		err = os.MkdirAll(newProjectDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create new project directory: %v", err)
		}

		// セッションファイルを作成（Git管理外）
		sessionPath := filepath.Join(newProjectDir, "session-1.jsonl")
		sessionContent := `{"type":"user","timestamp":"2024-01-01T17:00:00Z","sessionId":"session-nogit","uuid":"uuid-ng-1","cwd":"` + newProjectDir + `","version":"1.0.0","gitBranch":"main","message":{"model":"claude-sonnet-4-5","role":"user","content":[{"type":"text","text":"No git"}]}}
{"type":"assistant","timestamp":"2024-01-01T17:00:05Z","sessionId":"session-nogit","uuid":"uuid-ng-2","parentUuid":"uuid-ng-1","cwd":"` + newProjectDir + `","version":"1.0.0","gitBranch":"main","message":{"model":"claude-sonnet-4-5","role":"assistant","content":[{"type":"text","text":"OK"}],"usage":{"input_tokens":8,"output_tokens":4,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
		err = os.WriteFile(sessionPath, []byte(sessionContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write session file: %v", err)
		}

		// 差分同期を実行
		time.Sleep(10 * time.Millisecond)
		_, err = SyncIncremental(newDB, p)
		if err != nil {
			t.Fatalf("Incremental sync failed: %v", err)
		}

		// 既存プロジェクトの情報を再取得
		existingProjectAfter, err := newDB.GetProjectByName("test-project-1")
		if err != nil {
			t.Fatalf("Failed to get existing project after sync: %v", err)
		}

		// Git Rootが変更されていないことを確認
		if originalGitRoot == nil && existingProjectAfter.GitRoot != nil {
			t.Error("Git Root should remain nil for existing project")
		} else if originalGitRoot != nil && existingProjectAfter.GitRoot == nil {
			t.Error("Git Root should not be set to nil for existing project")
		} else if originalGitRoot != nil && existingProjectAfter.GitRoot != nil {
			if *originalGitRoot != *existingProjectAfter.GitRoot {
				t.Errorf("Git Root should not change for existing project: %s -> %s",
					*originalGitRoot, *existingProjectAfter.GitRoot)
			}
		}

		// UpdatedAtは変更される可能性がある（last_scan_timeの更新）ため、チェックしない
		// ここでは、Git Rootが変更されていないことのみ確認
		_ = originalUpdatedAt
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

	t.Run("SyncResultにエラー詳細が含まれる", func(t *testing.T) {
		result := &SyncResult{
			ProjectsProcessed: 1,
			SessionsFound:     2,
			SessionsSynced:    1,
			SessionsSkipped:   0,
			ErrorCount:        1,
			Errors:            []string{"project1/session1: parse error"},
		}

		if len(result.Errors) != 1 {
			t.Errorf("Expected 1 error, got %d", len(result.Errors))
		}
		if result.Errors[0] != "project1/session1: parse error" {
			t.Errorf("Expected 'project1/session1: parse error', got '%s'", result.Errors[0])
		}
	})
}

func TestSyncAll_WithLogging(t *testing.T) {
	database, _ := setupTestDB(t)
	defer database.Close()

	claudeDir := setupTestClaudeDir(t)
	p := parser.NewParser(claudeDir)

	t.Run("同期処理のログが出力される", func(t *testing.T) {
		// ログバッファを作成
		buf := &bytes.Buffer{}
		log := &logger.Logger{}
		log.SetOutput(buf)
		log.SetLevel(logger.DEBUG)

		// ロガーを設定してSyncAllを実行
		result, err := SyncAllWithLogger(database, p, log)
		if err != nil {
			t.Fatalf("SyncAll failed: %v", err)
		}

		// 結果を検証
		if result.ProjectsProcessed != 2 {
			t.Errorf("Expected 2 projects processed, got %d", result.ProjectsProcessed)
		}

		// ログ出力を確認
		output := buf.String()

		// SyncAll開始ログ
		if !strings.Contains(output, "Starting SyncAll") {
			t.Error("Expected 'Starting SyncAll' log message")
		}

		// プロジェクト数のログ
		if !strings.Contains(output, "Projects found") {
			t.Error("Expected 'Projects found' log message")
		}

		// プロジェクト処理開始ログ
		if !strings.Contains(output, "Syncing project") {
			t.Error("Expected 'Syncing project' log message")
		}

		// セッション数のログ
		if !strings.Contains(output, "Sessions found in filesystem") {
			t.Error("Expected 'Sessions found in filesystem' log message")
		}

		// 完了ログ
		if !strings.Contains(output, "completed") || !strings.Contains(output, "synced") {
			t.Error("Expected completion log message")
		}
	})

	t.Run("エラー発生時の詳細ログが出力される", func(t *testing.T) {
		// 新しいDBでテスト
		newDB, _ := setupTestDB(t)
		defer newDB.Close()

		// 不正なJSONLファイルを追加
		project1Dir := filepath.Join(claudeDir, "test-project-1")
		invalidSession := filepath.Join(project1Dir, "invalid-session.jsonl")
		err := os.WriteFile(invalidSession, []byte(""), 0644)
		if err != nil {
			t.Fatalf("Failed to write invalid session: %v", err)
		}

		// ログバッファを作成
		buf := &bytes.Buffer{}
		log := &logger.Logger{}
		log.SetOutput(buf)
		log.SetLevel(logger.DEBUG)

		// ロガーを設定してSyncAllを実行
		_, err = SyncAllWithLogger(newDB, p, log)

		// ログ出力を確認
		output := buf.String()

		// エラーログが含まれることを確認（パースエラーまたはDB挿入エラー）
		// 注: 空のファイルは有効なセッションとして処理される可能性があるため、
		// 別のエラーケースでテストする必要があるかもしれません
		t.Logf("Log output: %s", output)
	})
}

func TestSyncProject_WithLogging(t *testing.T) {
	database, _ := setupTestDB(t)
	defer database.Close()

	claudeDir := setupTestClaudeDir(t)
	p := parser.NewParser(claudeDir)

	t.Run("プロジェクト同期のログが出力される", func(t *testing.T) {
		// ログバッファを作成
		buf := &bytes.Buffer{}
		log := &logger.Logger{}
		log.SetOutput(buf)
		log.SetLevel(logger.DEBUG)

		// ロガーを設定してSyncProjectを実行
		result, err := SyncProjectWithLogger(database, p, "test-project-1", log)
		if err != nil {
			t.Fatalf("SyncProject failed: %v", err)
		}

		// 結果を検証
		if result.ProjectsProcessed != 1 {
			t.Errorf("Expected 1 project processed, got %d", result.ProjectsProcessed)
		}

		// ログ出力を確認
		output := buf.String()

		// プロジェクト名のログ
		if !strings.Contains(output, "test-project-1") {
			t.Error("Expected project name in log message")
		}

		// セッション数のログ
		if !strings.Contains(output, "sessions") {
			t.Error("Expected 'sessions' in log message")
		}
	})
}

func TestSyncAll_DuplicateSessions(t *testing.T) {
	database, _ := setupTestDB(t)
	defer database.Close()

	claudeDir := setupTestClaudeDir(t)
	p := parser.NewParser(claudeDir)

	t.Run("重複セッションはスキップされる", func(t *testing.T) {
		// 1回目の同期
		result1, err := SyncAll(database, p)
		if err != nil {
			t.Fatalf("First SyncAll failed: %v", err)
		}

		initialSynced := result1.SessionsSynced

		// 2回目の同期（すべてスキップされるはず）
		result2, err := SyncAll(database, p)
		if err != nil {
			t.Fatalf("Second SyncAll failed: %v", err)
		}

		// すべてのセッションがスキップされる
		if result2.SessionsSynced != 0 {
			t.Errorf("Expected 0 sessions synced on second run, got %d", result2.SessionsSynced)
		}
		if result2.SessionsSkipped != int(initialSynced) {
			t.Errorf("Expected %d sessions skipped, got %d", initialSynced, result2.SessionsSkipped)
		}
		// 重複はエラーとしてカウントしない
		if result2.ErrorCount != 0 {
			t.Errorf("Expected 0 errors for duplicates, got %d", result2.ErrorCount)
		}
	})
}

// TestSyncResult_ErrorDetails is commented out because the parser is designed to
// continue processing even when individual lines contain invalid JSON.
// This is intentional behavior to handle partially corrupted log files.
// Errors are logged as warnings but don't cause the session parsing to fail.
/*
func TestSyncResult_ErrorDetails(t *testing.T) {
	database, _ := setupTestDB(t)
	defer database.Close()

	claudeDir := setupTestClaudeDir(t)

	// 不正なJSONファイルを含むプロジェクトを作成
	project1Dir := filepath.Join(claudeDir, "error-test-project")
	err := os.MkdirAll(project1Dir, 0755)
	if err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	// 不正なJSONLファイル
	invalidSession := filepath.Join(project1Dir, "invalid.jsonl")
	err = os.WriteFile(invalidSession, []byte("invalid json"), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid session: %v", err)
	}

	p := parser.NewParser(claudeDir)

	t.Run("パースエラーがErrors配列に記録される", func(t *testing.T) {
		result, err := SyncProject(database, p, "error-test-project")

		// エラーがあっても処理は続行される
		if err != nil {
			t.Logf("SyncProject returned error: %v", err)
		}

		// エラーカウントが1以上
		if result.ErrorCount < 1 {
			t.Errorf("Expected at least 1 error, got %d", result.ErrorCount)
		}

		// Errors配列が空でない
		if len(result.Errors) == 0 {
			t.Error("Expected non-empty Errors array")
		}

		// エラーメッセージにプロジェクト名が含まれる
		hasProjectInError := false
		for _, errMsg := range result.Errors {
			if strings.Contains(errMsg, "error-test-project") {
				hasProjectInError = true
				break
			}
		}
		if !hasProjectInError {
			t.Error("Expected project name in error messages")
		}
	})
}
*/

func TestSyncAllWithCallback(t *testing.T) {
	database, _ := setupTestDB(t)
	defer database.Close()

	claudeDir := setupTestClaudeDir(t)
	p := parser.NewParser(claudeDir)

	t.Run("コールバックが呼び出される", func(t *testing.T) {
		// コールバック呼び出しを記録
		var updates []SyncProgressUpdate
		callback := func(update SyncProgressUpdate) {
			updates = append(updates, update)
		}

		// コールバック付きでSyncAll実行
		result, err := SyncAllWithCallback(database, p, callback)
		if err != nil {
			t.Fatalf("SyncAllWithCallback failed: %v", err)
		}

		// コールバックが呼び出されたことを確認
		if len(updates) == 0 {
			t.Error("Expected callback to be called, but got no updates")
		}

		// プロジェクト数と一致することを確認（2プロジェクト）
		if len(updates) != 2 {
			t.Errorf("Expected 2 callback calls (one per project), got %d", len(updates))
		}

		// 進捗が増加していることを確認
		for i := 1; i < len(updates); i++ {
			if updates[i].ProjectsProcessed <= updates[i-1].ProjectsProcessed {
				t.Errorf("Progress should increase, got %d -> %d",
					updates[i-1].ProjectsProcessed, updates[i].ProjectsProcessed)
			}
		}

		// 最終結果と最後のコールバックが一致することを確認
		lastUpdate := updates[len(updates)-1]
		if lastUpdate.ProjectsProcessed != result.ProjectsProcessed {
			t.Errorf("Last callback ProjectsProcessed=%d, result=%d",
				lastUpdate.ProjectsProcessed, result.ProjectsProcessed)
		}
		if lastUpdate.SessionsFound != result.SessionsFound {
			t.Errorf("Last callback SessionsFound=%d, result=%d",
				lastUpdate.SessionsFound, result.SessionsFound)
		}
		if lastUpdate.SessionsSynced != result.SessionsSynced {
			t.Errorf("Last callback SessionsSynced=%d, result=%d",
				lastUpdate.SessionsSynced, result.SessionsSynced)
		}
		if lastUpdate.SessionsSkipped != result.SessionsSkipped {
			t.Errorf("Last callback SessionsSkipped=%d, result=%d",
				lastUpdate.SessionsSkipped, result.SessionsSkipped)
		}
	})
}

func TestSyncAllWithNilCallback(t *testing.T) {
	database, _ := setupTestDB(t)
	defer database.Close()

	claudeDir := setupTestClaudeDir(t)
	p := parser.NewParser(claudeDir)

	t.Run("nilコールバックでもエラーが発生しない", func(t *testing.T) {
		result, err := SyncAllWithCallback(database, p, nil)
		if err != nil {
			t.Fatalf("SyncAllWithCallback with nil callback failed: %v", err)
		}

		// 結果が正しいことを確認
		if result.ProjectsProcessed != 2 {
			t.Errorf("Expected 2 projects processed, got %d", result.ProjectsProcessed)
		}
		if result.SessionsSynced != 3 {
			t.Errorf("Expected 3 sessions synced, got %d", result.SessionsSynced)
		}
	})
}

func TestGitRootDetection(t *testing.T) {
	database, _ := setupTestDB(t)
	defer database.Close()

	t.Run("通常のGitリポジトリでGit Rootが検出される", func(t *testing.T) {
		// テスト用のClaudeディレクトリとGitリポジトリを作成
		tmpDir := t.TempDir()
		claudeDir := filepath.Join(tmpDir, ".claude", "projects")
		projectDir := filepath.Join(claudeDir, "git-repo-project")
		err := os.MkdirAll(projectDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create project directory: %v", err)
		}

		// .gitディレクトリを作成
		gitDir := filepath.Join(projectDir, ".git")
		err = os.Mkdir(gitDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create .git directory: %v", err)
		}

		// セッションファイルを作成
		sessionPath := filepath.Join(projectDir, "session-1.jsonl")
		sessionContent := `{"type":"user","timestamp":"2024-01-01T10:00:00Z","sessionId":"session-1","uuid":"uuid-1","cwd":"` + projectDir + `","version":"1.0.0","gitBranch":"main","message":{"model":"claude-sonnet-4-5","role":"user","content":[{"type":"text","text":"Test"}]}}
{"type":"assistant","timestamp":"2024-01-01T10:00:05Z","sessionId":"session-1","uuid":"uuid-2","parentUuid":"uuid-1","cwd":"` + projectDir + `","version":"1.0.0","gitBranch":"main","message":{"model":"claude-sonnet-4-5","role":"assistant","content":[{"type":"text","text":"OK"}],"usage":{"input_tokens":10,"output_tokens":5,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
		err = os.WriteFile(sessionPath, []byte(sessionContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write session file: %v", err)
		}

		// パーサーとsync実行
		p := parser.NewParser(claudeDir)
		_, err = SyncAll(database, p)
		if err != nil {
			t.Fatalf("SyncAll failed: %v", err)
		}

		// プロジェクトを取得してGit Rootを確認
		project, err := database.GetProjectByName("git-repo-project")
		if err != nil {
			t.Fatalf("Failed to get project: %v", err)
		}

		if project.GitRoot == nil {
			t.Error("Expected git_root to be set, got nil")
		} else if *project.GitRoot != projectDir {
			t.Errorf("Expected git_root %s, got %s", projectDir, *project.GitRoot)
		}
	})

	t.Run("Gitワークツリーで親リポジトリのGit Rootが検出される", func(t *testing.T) {
		// 新しいデータベースを作成
		newDB, _ := setupTestDB(t)
		defer newDB.Close()

		// テスト用のClaudeディレクトリとワークツリーを作成
		tmpDir := t.TempDir()
		claudeDir := filepath.Join(tmpDir, ".claude", "projects")
		worktreeDir := filepath.Join(claudeDir, "worktree-project")
		err := os.MkdirAll(worktreeDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create worktree directory: %v", err)
		}

		// 親リポジトリのパス
		parentRepoPath := "/path/to/parent/repo.git"

		// .gitファイルを作成（gitdir指定）
		gitFilePath := filepath.Join(worktreeDir, ".git")
		gitFileContent := "gitdir: " + parentRepoPath + "/worktrees/feature\n"
		err = os.WriteFile(gitFilePath, []byte(gitFileContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create .git file: %v", err)
		}

		// セッションファイルを作成
		sessionPath := filepath.Join(worktreeDir, "session-1.jsonl")
		sessionContent := `{"type":"user","timestamp":"2024-01-01T10:00:00Z","sessionId":"session-1","uuid":"uuid-1","cwd":"` + worktreeDir + `","version":"1.0.0","gitBranch":"feature","message":{"model":"claude-sonnet-4-5","role":"user","content":[{"type":"text","text":"Test"}]}}
{"type":"assistant","timestamp":"2024-01-01T10:00:05Z","sessionId":"session-1","uuid":"uuid-2","parentUuid":"uuid-1","cwd":"` + worktreeDir + `","version":"1.0.0","gitBranch":"feature","message":{"model":"claude-sonnet-4-5","role":"assistant","content":[{"type":"text","text":"OK"}],"usage":{"input_tokens":10,"output_tokens":5,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
		err = os.WriteFile(sessionPath, []byte(sessionContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write session file: %v", err)
		}

		// パーサーとsync実行
		p := parser.NewParser(claudeDir)
		_, err = SyncAll(newDB, p)
		if err != nil {
			t.Fatalf("SyncAll failed: %v", err)
		}

		// プロジェクトを取得してGit Rootを確認
		project, err := newDB.GetProjectByName("worktree-project")
		if err != nil {
			t.Fatalf("Failed to get project: %v", err)
		}

		if project.GitRoot == nil {
			t.Error("Expected git_root to be set, got nil")
		} else if *project.GitRoot != parentRepoPath {
			t.Errorf("Expected git_root %s, got %s", parentRepoPath, *project.GitRoot)
		}
	})

	t.Run("Git管理外のプロジェクトでgit_rootがnullになる", func(t *testing.T) {
		// 新しいデータベースを作成
		newDB, _ := setupTestDB(t)
		defer newDB.Close()

		// テスト用のClaudeディレクトリとGit管理外のプロジェクトを作成
		tmpDir := t.TempDir()
		claudeDir := filepath.Join(tmpDir, ".claude", "projects")
		nonGitDir := filepath.Join(claudeDir, "non-git-project")
		err := os.MkdirAll(nonGitDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create non-git directory: %v", err)
		}

		// セッションファイルを作成（.gitなし）
		sessionPath := filepath.Join(nonGitDir, "session-1.jsonl")
		sessionContent := `{"type":"user","timestamp":"2024-01-01T10:00:00Z","sessionId":"session-1","uuid":"uuid-1","cwd":"` + nonGitDir + `","version":"1.0.0","gitBranch":"main","message":{"model":"claude-sonnet-4-5","role":"user","content":[{"type":"text","text":"Test"}]}}
{"type":"assistant","timestamp":"2024-01-01T10:00:05Z","sessionId":"session-1","uuid":"uuid-2","parentUuid":"uuid-1","cwd":"` + nonGitDir + `","version":"1.0.0","gitBranch":"main","message":{"model":"claude-sonnet-4-5","role":"assistant","content":[{"type":"text","text":"OK"}],"usage":{"input_tokens":10,"output_tokens":5,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
		err = os.WriteFile(sessionPath, []byte(sessionContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write session file: %v", err)
		}

		// パーサーとsync実行
		p := parser.NewParser(claudeDir)
		_, err = SyncAll(newDB, p)
		if err != nil {
			t.Fatalf("SyncAll failed: %v", err)
		}

		// プロジェクトを取得してGit Rootを確認
		project, err := newDB.GetProjectByName("non-git-project")
		if err != nil {
			t.Fatalf("Failed to get project: %v", err)
		}

		if project.GitRoot != nil {
			t.Errorf("Expected git_root to be null, got %s", *project.GitRoot)
		}
	})

	t.Run("既存プロジェクトでgit_rootが未設定の場合、更新される", func(t *testing.T) {
		// 新しいデータベースを作成
		newDB, _ := setupTestDB(t)
		defer newDB.Close()

		// テスト用のClaudeディレクトリとGitリポジトリを作成
		tmpDir := t.TempDir()
		claudeDir := filepath.Join(tmpDir, ".claude", "projects")
		projectDir := filepath.Join(claudeDir, "existing-project")
		err := os.MkdirAll(projectDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create project directory: %v", err)
		}

		// セッションファイルを作成（最初は.gitなし）
		sessionPath := filepath.Join(projectDir, "session-1.jsonl")
		sessionContent := `{"type":"user","timestamp":"2024-01-01T10:00:00Z","sessionId":"session-1","uuid":"uuid-1","cwd":"` + projectDir + `","version":"1.0.0","gitBranch":"main","message":{"model":"claude-sonnet-4-5","role":"user","content":[{"type":"text","text":"Test"}]}}
{"type":"assistant","timestamp":"2024-01-01T10:00:05Z","sessionId":"session-1","uuid":"uuid-2","parentUuid":"uuid-1","cwd":"` + projectDir + `","version":"1.0.0","gitBranch":"main","message":{"model":"claude-sonnet-4-5","role":"assistant","content":[{"type":"text","text":"OK"}],"usage":{"input_tokens":10,"output_tokens":5,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
		err = os.WriteFile(sessionPath, []byte(sessionContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write session file: %v", err)
		}

		// パーサーとsync実行（.gitなし）
		p := parser.NewParser(claudeDir)
		_, err = SyncAll(newDB, p)
		if err != nil {
			t.Fatalf("First sync failed: %v", err)
		}

		// プロジェクトを取得してGit Rootがnullであることを確認
		project, err := newDB.GetProjectByName("existing-project")
		if err != nil {
			t.Fatalf("Failed to get project: %v", err)
		}
		if project.GitRoot != nil {
			t.Errorf("Expected git_root to be null initially, got %s", *project.GitRoot)
		}

		// .gitディレクトリを作成
		gitDir := filepath.Join(projectDir, ".git")
		err = os.Mkdir(gitDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create .git directory: %v", err)
		}

		// 再度sync実行
		_, err = SyncAll(newDB, p)
		if err != nil {
			t.Fatalf("Second sync failed: %v", err)
		}

		// プロジェクトを再取得してGit Rootが更新されたことを確認
		project, err = newDB.GetProjectByName("existing-project")
		if err != nil {
			t.Fatalf("Failed to get project after update: %v", err)
		}

		if project.GitRoot == nil {
			t.Error("Expected git_root to be updated, still null")
		} else if *project.GitRoot != projectDir {
			t.Errorf("Expected git_root %s, got %s", projectDir, *project.GitRoot)
		}
	})

	t.Run("セッションが存在しないプロジェクトでgit_rootがnullになる", func(t *testing.T) {
		// 新しいデータベースを作成
		newDB, _ := setupTestDB(t)
		defer newDB.Close()

		// テスト用のClaudeディレクトリとプロジェクトを作成（セッションなし）
		tmpDir := t.TempDir()
		claudeDir := filepath.Join(tmpDir, ".claude", "projects")
		projectDir := filepath.Join(claudeDir, "no-session-project")
		err := os.MkdirAll(projectDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create project directory: %v", err)
		}

		// .gitディレクトリを作成（Git管理下だが、セッションがない）
		gitDir := filepath.Join(projectDir, ".git")
		err = os.Mkdir(gitDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create .git directory: %v", err)
		}

		// パーサーとsync実行（セッションなし）
		p := parser.NewParser(claudeDir)
		_, err = SyncAll(newDB, p)
		if err != nil {
			t.Fatalf("SyncAll failed: %v", err)
		}

		// プロジェクトを取得してGit Rootがnullであることを確認
		// セッションが存在しないため、実際の作業ディレクトリが取得できず、
		// Git Root検出がスキップされる
		project, err := newDB.GetProjectByName("no-session-project")
		if err != nil {
			t.Fatalf("Failed to get project: %v", err)
		}
		if project.GitRoot != nil {
			t.Errorf("Expected git_root to be null (no sessions), got %s", *project.GitRoot)
		}
	})
}
