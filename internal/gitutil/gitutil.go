package gitutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DetectGitRoot detects the Git root directory for a given project path
// Returns:
// - Git root path for normal repositories and worktrees
// - Empty string if the directory is not under Git control (not an error)
// - Error if the path doesn't exist or .git file is malformed
func DetectGitRoot(projectPath string) (string, error) {
	// プロジェクトパスが存在するか確認
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		return "", fmt.Errorf("project path does not exist: %s", projectPath)
	}

	// .gitのパスを構築
	gitPath := filepath.Join(projectPath, ".git")

	// .gitが存在するか確認
	info, err := os.Stat(gitPath)
	if os.IsNotExist(err) {
		// Git管理外（エラーではない）
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to stat .git: %w", err)
	}

	// .gitがディレクトリの場合（通常のGitリポジトリ）
	if info.IsDir() {
		return projectPath, nil
	}

	// .gitがファイルの場合（Gitワークツリー）
	content, err := os.ReadFile(gitPath)
	if err != nil {
		return "", fmt.Errorf("failed to read .git file: %w", err)
	}

	// .gitファイルからgitdirパスを抽出
	gitdirPath, err := parseGitdirFile(string(content))
	if err != nil {
		return "", fmt.Errorf("failed to parse .git file: %w", err)
	}

	// 相対パスの場合は絶対パスに変換
	if !filepath.IsAbs(gitdirPath) {
		gitdirPath = filepath.Join(projectPath, gitdirPath)
	}

	// worktreeパスから親リポジトリのパスを抽出
	gitRoot, err := extractGitRootFromWorktree(gitdirPath)
	if err != nil {
		return "", fmt.Errorf("failed to extract git root from worktree path: %w", err)
	}

	return gitRoot, nil
}

// parseGitdirFile parses the content of a .git file and extracts the gitdir path
// Expected format: "gitdir: /path/to/repo.git/worktrees/name"
func parseGitdirFile(content string) (string, error) {
	// "gitdir: " プレフィックスを探す
	prefix := "gitdir: "
	if !strings.HasPrefix(content, prefix) {
		return "", fmt.Errorf("invalid .git file format: missing 'gitdir:' prefix")
	}

	// プレフィックスを削除
	path := strings.TrimPrefix(content, prefix)

	// 改行・空白を削除
	path = strings.TrimSpace(path)

	// パスが空でないことを確認
	if path == "" {
		return "", fmt.Errorf("invalid .git file format: empty gitdir path")
	}

	return path, nil
}

// extractGitRootFromWorktree extracts the Git root path from a worktree gitdir path
// Example: "/path/to/repo.git/worktrees/feature" -> "/path/to/repo.git"
// Example: "/path/to/repo/.git/worktrees/feature" -> "/path/to/repo"
func extractGitRootFromWorktree(worktreePath string) (string, error) {
	if worktreePath == "" {
		return "", fmt.Errorf("worktree path is empty")
	}

	// パスをクリーンアップして正規化
	cleanPath := filepath.Clean(worktreePath)

	// パスを要素に分割（OSに応じた区切り文字を使用）
	parts := strings.Split(cleanPath, string(filepath.Separator))

	// "worktrees"の位置を逆順で探す（最後の"worktrees"を見つける）
	worktreesIndex := -1
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == "worktrees" {
			worktreesIndex = i
			break
		}
	}

	if worktreesIndex == -1 {
		return "", fmt.Errorf("'worktrees' directory not found in path: %s", worktreePath)
	}

	// worktreesより前の部分を取得
	gitRootParts := parts[:worktreesIndex]

	// 空のパーツがある場合の処理（Unixルートパスなど）
	if len(gitRootParts) == 0 {
		return "", fmt.Errorf("invalid worktree path: %s", worktreePath)
	}

	// 最後の要素が ".git" の場合は除去（通常のリポジトリ構造）
	if len(gitRootParts) > 0 && gitRootParts[len(gitRootParts)-1] == ".git" {
		gitRootParts = gitRootParts[:len(gitRootParts)-1]
	}

	// パスを再構築
	gitRoot := filepath.Join(gitRootParts...)

	// Unixの絶対パスの場合、先頭の/を復元
	if filepath.IsAbs(worktreePath) && !filepath.IsAbs(gitRoot) {
		gitRoot = string(filepath.Separator) + gitRoot
	}

	return gitRoot, nil
}
