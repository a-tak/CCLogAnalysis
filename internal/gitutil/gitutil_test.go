package gitutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectGitRoot(t *testing.T) {
	t.Run("通常のGitリポジトリ", func(t *testing.T) {
		// テスト用の通常のGitリポジトリを作成
		tmpDir := t.TempDir()
		gitDir := filepath.Join(tmpDir, ".git")
		err := os.Mkdir(gitDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create .git directory: %v", err)
		}

		// Git Rootを検出
		gitRoot, err := DetectGitRoot(tmpDir)
		if err != nil {
			t.Fatalf("DetectGitRoot failed: %v", err)
		}

		// Git RootはtmpDir自身であるべき
		if gitRoot != tmpDir {
			t.Errorf("Expected git root %s, got %s", tmpDir, gitRoot)
		}
	})

	t.Run("Gitワークツリー", func(t *testing.T) {
		// テスト用のワークツリーを作成
		tmpDir := t.TempDir()

		// 親リポジトリのパス（実際には存在しない仮のパス）
		parentRepoPath := "/path/to/parent/repo.git"

		// .gitファイルを作成（gitdir指定）
		gitFilePath := filepath.Join(tmpDir, ".git")
		gitFileContent := "gitdir: " + parentRepoPath + "/worktrees/feature-branch\n"
		err := os.WriteFile(gitFilePath, []byte(gitFileContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create .git file: %v", err)
		}

		// Git Rootを検出
		gitRoot, err := DetectGitRoot(tmpDir)
		if err != nil {
			t.Fatalf("DetectGitRoot failed: %v", err)
		}

		// Git Rootは親リポジトリのパスであるべき
		if gitRoot != parentRepoPath {
			t.Errorf("Expected git root %s, got %s", parentRepoPath, gitRoot)
		}
	})

	t.Run("Gitワークツリー（相対パス）", func(t *testing.T) {
		// テスト用のワークツリーを作成
		tmpDir := t.TempDir()

		// 相対パスのgitdir
		gitFileContent := "gitdir: ../.git/worktrees/feature-branch\n"
		gitFilePath := filepath.Join(tmpDir, ".git")
		err := os.WriteFile(gitFilePath, []byte(gitFileContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create .git file: %v", err)
		}

		// Git Rootを検出
		gitRoot, err := DetectGitRoot(tmpDir)
		if err != nil {
			t.Fatalf("DetectGitRoot failed: %v", err)
		}

		// 相対パスが解決されて絶対パスになることを確認
		expectedRoot := filepath.Clean(filepath.Join(tmpDir, ".."))
		if gitRoot != expectedRoot {
			t.Errorf("Expected git root %s, got %s", expectedRoot, gitRoot)
		}
	})

	t.Run("Git管理外のディレクトリ", func(t *testing.T) {
		// .gitが存在しないディレクトリ
		tmpDir := t.TempDir()

		// Git Rootを検出
		gitRoot, err := DetectGitRoot(tmpDir)
		if err != nil {
			t.Fatalf("DetectGitRoot should not return error for non-git directory: %v", err)
		}

		// Git管理外の場合は空文字列を返す
		if gitRoot != "" {
			t.Errorf("Expected empty string for non-git directory, got %s", gitRoot)
		}
	})

	t.Run("不正な.gitファイル形式", func(t *testing.T) {
		// テスト用のディレクトリを作成
		tmpDir := t.TempDir()

		// 不正な形式の.gitファイルを作成
		gitFilePath := filepath.Join(tmpDir, ".git")
		gitFileContent := "invalid content without gitdir:\n"
		err := os.WriteFile(gitFilePath, []byte(gitFileContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create .git file: %v", err)
		}

		// Git Rootを検出（エラーになるべき）
		_, err = DetectGitRoot(tmpDir)
		if err == nil {
			t.Error("Expected error for invalid .git file format, got nil")
		}
	})

	t.Run("存在しないディレクトリ", func(t *testing.T) {
		// 存在しないディレクトリ
		nonExistentDir := "/path/that/does/not/exist"

		// Git Rootを検出（エラーになるべき）
		_, err := DetectGitRoot(nonExistentDir)
		if err == nil {
			t.Error("Expected error for non-existent directory, got nil")
		}
	})

	t.Run("空のgitdirパス", func(t *testing.T) {
		// テスト用のディレクトリを作成
		tmpDir := t.TempDir()

		// gitdir:の後が空の.gitファイルを作成
		gitFilePath := filepath.Join(tmpDir, ".git")
		gitFileContent := "gitdir: \n"
		err := os.WriteFile(gitFilePath, []byte(gitFileContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create .git file: %v", err)
		}

		// Git Rootを検出（エラーになるべき）
		_, err = DetectGitRoot(tmpDir)
		if err == nil {
			t.Error("Expected error for empty gitdir path, got nil")
		}
	})
}

func TestParseGitdirFile(t *testing.T) {
	t.Run("正常なgitdirパス（絶対パス）", func(t *testing.T) {
		content := "gitdir: /path/to/repo.git/worktrees/feature\n"
		expectedPath := "/path/to/repo.git/worktrees/feature"

		path, err := parseGitdirFile(content)
		if err != nil {
			t.Fatalf("parseGitdirFile failed: %v", err)
		}

		if path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, path)
		}
	})

	t.Run("正常なgitdirパス（相対パス）", func(t *testing.T) {
		content := "gitdir: ../.git/worktrees/feature\n"
		expectedPath := "../.git/worktrees/feature"

		path, err := parseGitdirFile(content)
		if err != nil {
			t.Fatalf("parseGitdirFile failed: %v", err)
		}

		if path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, path)
		}
	})

	t.Run("gitdir:がない", func(t *testing.T) {
		content := "invalid content\n"

		_, err := parseGitdirFile(content)
		if err == nil {
			t.Error("Expected error for content without gitdir:, got nil")
		}
	})

	t.Run("空のgitdirパス", func(t *testing.T) {
		content := "gitdir: \n"

		_, err := parseGitdirFile(content)
		if err == nil {
			t.Error("Expected error for empty gitdir path, got nil")
		}
	})

	t.Run("gitdir:のみ（改行なし）", func(t *testing.T) {
		content := "gitdir:"

		_, err := parseGitdirFile(content)
		if err == nil {
			t.Error("Expected error for gitdir: without path, got nil")
		}
	})
}

func TestExtractGitRootFromWorktree(t *testing.T) {
	tests := []struct {
		name           string
		worktreePath   string
		expectedRoot   string
		expectError    bool
	}{
		{
			name:         "標準的なworktreeパス",
			worktreePath: "/path/to/repo.git/worktrees/feature",
			expectedRoot: "/path/to/repo.git",
			expectError:  false,
		},
		// Windowsパスのテストは削除（filepathはOSネイティブの区切り文字を使用するため、
		// Unix系OSではWindowsパスを正しく処理できない）
		{
			name:         "深い階層のworktree",
			worktreePath: "/path/to/repo.git/worktrees/nested/feature",
			expectedRoot: "/path/to/repo.git",
			expectError:  false,
		},
		{
			name:         "worktreesが含まれないパス",
			worktreePath: "/path/to/repo.git/feature",
			expectedRoot: "",
			expectError:  true,
		},
		{
			name:         "空のパス",
			worktreePath: "",
			expectedRoot: "",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, err := extractGitRootFromWorktree(tt.worktreePath)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if root != tt.expectedRoot {
				t.Errorf("Expected root %s, got %s", tt.expectedRoot, root)
			}
		})
	}
}
