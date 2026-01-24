package db

import (
	"fmt"
	"log"

	"github.com/a-tak/ccloganalysis/internal/gitutil"
	"github.com/a-tak/ccloganalysis/internal/parser"
)

// SyncResult represents the result of a sync operation
type SyncResult struct {
	ProjectsProcessed int
	SessionsFound     int
	SessionsSynced    int
	SessionsSkipped   int
	ErrorCount        int
}

// SyncAll synchronizes all projects from the file system to the database
func SyncAll(db *DB, p *parser.Parser) (*SyncResult, error) {
	result := &SyncResult{}

	// プロジェクト一覧を取得
	projectNames, err := p.ListProjects()
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	// 各プロジェクトを同期
	for _, projectName := range projectNames {
		syncResult, err := syncProjectInternal(db, p, projectName)
		if err != nil {
			log.Printf("Warning: failed to sync project %s: %v", projectName, err)
			result.ErrorCount++
			continue
		}

		result.ProjectsProcessed++
		result.SessionsFound += syncResult.SessionsFound
		result.SessionsSynced += syncResult.SessionsSynced
		result.SessionsSkipped += syncResult.SessionsSkipped
		result.ErrorCount += syncResult.ErrorCount
	}

	return result, nil
}

// SyncProject synchronizes a specific project from the file system to the database
func SyncProject(db *DB, p *parser.Parser, projectName string) (*SyncResult, error) {
	result, err := syncProjectInternal(db, p, projectName)
	if err != nil {
		return nil, err
	}

	result.ProjectsProcessed = 1
	return result, nil
}

// SyncIncremental synchronizes only new sessions (not yet in database)
func SyncIncremental(db *DB, p *parser.Parser) (*SyncResult, error) {
	// SyncAllと同じだが、既存セッションはスキップする
	// 実装上、SyncAllが既に重複チェックを行っているため、同じ動作になる
	return SyncAll(db, p)
}

// syncProjectInternal is an internal helper that synchronizes a single project
func syncProjectInternal(db *DB, p *parser.Parser, projectName string) (*SyncResult, error) {
	result := &SyncResult{}

	// プロジェクトをDBに登録（存在しない場合）
	project, err := db.GetProjectByName(projectName)
	if err != nil {
		// プロジェクトが存在しない場合は作成
		decodedPath := parser.DecodeProjectPath(projectName)

		// プロジェクトの実際のディレクトリパスを取得
		projectDir, err := p.GetProjectDir(projectName)
		if err != nil {
			return nil, fmt.Errorf("failed to get project directory: %w", err)
		}

		// Git Rootを検出
		gitRoot, gitErr := gitutil.DetectGitRoot(projectDir)
		if gitErr != nil {
			log.Printf("Warning: failed to detect git root for %s: %v", projectName, gitErr)
			// Git Root検出失敗はエラーとせず、git_root=nullで保存
			_, err = db.CreateProject(projectName, decodedPath)
		} else if gitRoot != "" {
			// Git Root検出成功
			_, err = db.CreateProjectWithGitRoot(projectName, decodedPath, gitRoot)
		} else {
			// Git管理外（空文字列）
			_, err = db.CreateProject(projectName, decodedPath)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to create project: %w", err)
		}

		// 作成したプロジェクトを取得
		project, err = db.GetProjectByName(projectName)
		if err != nil {
			return nil, fmt.Errorf("failed to get created project: %w", err)
		}
	} else if project.GitRoot == nil {
		// 既存プロジェクトでGit Rootが未設定の場合、検出して更新
		projectDir, err := p.GetProjectDir(projectName)
		if err != nil {
			log.Printf("Warning: failed to get project directory for %s: %v", projectName, err)
		} else {
			gitRoot, gitErr := gitutil.DetectGitRoot(projectDir)
			if gitErr != nil {
				log.Printf("Warning: failed to detect git root for existing project %s: %v", projectName, gitErr)
			} else if gitRoot != "" {
				// Git Root検出成功、更新
				err = db.UpdateProjectGitRoot(project.ID, gitRoot)
				if err != nil {
					log.Printf("Warning: failed to update git root for project %s: %v", projectName, err)
				} else {
					log.Printf("Updated git root for project %s: %s", projectName, gitRoot)
				}
			}
		}
	}

	// セッション一覧を取得
	sessionIDs, err := p.ListSessions(projectName)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	result.SessionsFound = len(sessionIDs)

	// 各セッションを同期
	for _, sessionID := range sessionIDs {
		// セッションが既にDBに存在するかチェック
		_, err := db.GetSession(sessionID)
		if err == nil {
			// 既に存在する場合はスキップ
			result.SessionsSkipped++
			continue
		}

		// セッションをパース
		session, err := p.ParseSession(projectName, sessionID)
		if err != nil {
			log.Printf("Warning: failed to parse session %s/%s: %v", projectName, sessionID, err)
			result.ErrorCount++
			continue
		}

		// セッションをDBに保存
		err = db.CreateSession(session, projectName)
		if err != nil {
			log.Printf("Warning: failed to save session %s/%s: %v", projectName, sessionID, err)
			result.ErrorCount++
			continue
		}

		result.SessionsSynced++
	}

	return result, nil
}
