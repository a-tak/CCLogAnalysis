package db

import (
	"fmt"
	"strings"

	"github.com/a-tak/ccloganalysis/internal/logger"
	"github.com/a-tak/ccloganalysis/internal/parser"
)

// SyncResult represents the result of a sync operation
type SyncResult struct {
	ProjectsProcessed int
	SessionsFound     int
	SessionsSynced    int
	SessionsSkipped   int
	ErrorCount        int
	Errors            []string // エラー詳細のリスト（"プロジェクト名: エラー内容" または "プロジェクト名/セッションID: エラー内容"）
}

// SyncAll synchronizes all projects from the file system to the database
func SyncAll(db *DB, p *parser.Parser) (*SyncResult, error) {
	// デフォルトロガーを使用
	return SyncAllWithLogger(db, p, logger.New())
}

// SyncAllWithLogger synchronizes all projects from the file system to the database with custom logger
func SyncAllWithLogger(db *DB, p *parser.Parser, log *logger.Logger) (*SyncResult, error) {
	result := &SyncResult{}

	log.Info("Starting SyncAll")

	// プロジェクト一覧を取得
	projectNames, err := p.ListProjects()
	if err != nil {
		log.ErrorWithContext("Failed to list projects", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	log.InfoWithContext("Projects found", map[string]interface{}{
		"count": len(projectNames),
	})

	// 各プロジェクトを同期
	for _, projectName := range projectNames {
		log.DebugWithContext("Syncing project", map[string]interface{}{
			"project": projectName,
		})

		syncResult, err := syncProjectInternalWithLogger(db, p, projectName, log)
		if err != nil {
			errMsg := fmt.Sprintf("%s: %v", projectName, err)
			log.ErrorWithContext("Failed to sync project", map[string]interface{}{
				"project": projectName,
				"error":   err.Error(),
			})
			result.ErrorCount++
			result.Errors = append(result.Errors, errMsg)
			continue
		}

		log.InfoWithContext("Project synced", map[string]interface{}{
			"project":         projectName,
			"sessions_synced": syncResult.SessionsSynced,
			"sessions_found":  syncResult.SessionsFound,
		})

		result.ProjectsProcessed++
		result.SessionsFound += syncResult.SessionsFound
		result.SessionsSynced += syncResult.SessionsSynced
		result.SessionsSkipped += syncResult.SessionsSkipped
		result.ErrorCount += syncResult.ErrorCount
		result.Errors = append(result.Errors, syncResult.Errors...)
	}

	log.InfoWithContext("SyncAll completed", map[string]interface{}{
		"projects_processed": result.ProjectsProcessed,
		"sessions_synced":    result.SessionsSynced,
		"sessions_found":     result.SessionsFound,
		"error_count":        result.ErrorCount,
	})

	return result, nil
}

// SyncProject synchronizes a specific project from the file system to the database
func SyncProject(db *DB, p *parser.Parser, projectName string) (*SyncResult, error) {
	// デフォルトロガーを使用
	return SyncProjectWithLogger(db, p, projectName, logger.New())
}

// SyncProjectWithLogger synchronizes a specific project from the file system to the database with custom logger
func SyncProjectWithLogger(db *DB, p *parser.Parser, projectName string, log *logger.Logger) (*SyncResult, error) {
	log.InfoWithContext("Starting SyncProject", map[string]interface{}{
		"project": projectName,
	})

	result, err := syncProjectInternalWithLogger(db, p, projectName, log)
	if err != nil {
		log.ErrorWithContext("SyncProject failed", map[string]interface{}{
			"project": projectName,
			"error":   err.Error(),
		})
		return nil, err
	}

	result.ProjectsProcessed = 1

	log.InfoWithContext("SyncProject completed", map[string]interface{}{
		"project":         projectName,
		"sessions_synced": result.SessionsSynced,
		"sessions_found":  result.SessionsFound,
	})

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
	// デフォルトロガーを使用
	return syncProjectInternalWithLogger(db, p, projectName, logger.New())
}

// syncProjectInternalWithLogger is an internal helper that synchronizes a single project with custom logger
func syncProjectInternalWithLogger(db *DB, p *parser.Parser, projectName string, log *logger.Logger) (*SyncResult, error) {
	result := &SyncResult{}

	log.DebugWithContext("Checking project in database", map[string]interface{}{
		"project": projectName,
	})

	// プロジェクトをDBに登録（存在しない場合）
	project, err := db.GetProjectByName(projectName)
	if err != nil {
		// プロジェクトが存在しない場合は作成
		log.InfoWithContext("Creating new project", map[string]interface{}{
			"project": projectName,
		})

		decodedPath := parser.DecodeProjectPath(projectName)
		_, err = db.CreateProject(projectName, decodedPath)
		if err != nil {
			log.ErrorWithContext("Failed to create project", map[string]interface{}{
				"project": projectName,
				"error":   err.Error(),
			})
			return nil, fmt.Errorf("failed to create project: %w", err)
		}

		// 作成したプロジェクトを取得
		project, err = db.GetProjectByName(projectName)
		if err != nil {
			log.ErrorWithContext("Failed to get created project", map[string]interface{}{
				"project": projectName,
				"error":   err.Error(),
			})
			return nil, fmt.Errorf("failed to get created project: %w", err)
		}
	}

	// セッション一覧を取得
	sessionIDs, err := p.ListSessions(projectName)
	if err != nil {
		log.ErrorWithContext("Failed to list sessions", map[string]interface{}{
			"project": projectName,
			"error":   err.Error(),
		})
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	result.SessionsFound = len(sessionIDs)

	log.InfoWithContext("Sessions found in filesystem", map[string]interface{}{
		"project": projectName,
		"count":   result.SessionsFound,
	})

	// 各セッションを同期
	for _, sessionID := range sessionIDs {
		log.DebugWithContext("Processing session", map[string]interface{}{
			"project":    projectName,
			"session_id": sessionID,
		})

		// セッションが既にDBに存在するかチェック
		_, err := db.GetSession(sessionID)
		if err == nil {
			// 既に存在する場合はスキップ
			log.DebugWithContext("Session already exists, skipping", map[string]interface{}{
				"project":    projectName,
				"session_id": sessionID,
			})
			result.SessionsSkipped++
			continue
		}

		// セッションをパース
		session, err := p.ParseSession(projectName, sessionID)
		if err != nil {
			errMsg := fmt.Sprintf("%s/%s: %v", projectName, sessionID, err)
			log.ErrorWithContext("Failed to parse session", map[string]interface{}{
				"project":    projectName,
				"session_id": sessionID,
				"error":      err.Error(),
			})
			result.ErrorCount++
			result.Errors = append(result.Errors, errMsg)
			continue
		}

		// セッションをDBに保存
		err = db.CreateSession(session, projectName)
		if err != nil {
			// UNIQUE制約エラーの場合は、既に別のプロジェクトで登録済みとしてスキップ
			if isUniqueConstraintError(err) {
				log.WarnWithContext("Session already exists (duplicate), skipping", map[string]interface{}{
					"project":    projectName,
					"session_id": sessionID,
				})
				result.SessionsSkipped++
				continue
			}

			// その他のエラーはエラーとしてカウント
			errMsg := fmt.Sprintf("%s/%s: %v", projectName, sessionID, err)
			log.ErrorWithContext("Failed to save session", map[string]interface{}{
				"project":    projectName,
				"session_id": sessionID,
				"error":      err.Error(),
			})
			result.ErrorCount++
			result.Errors = append(result.Errors, errMsg)
			continue
		}

		log.DebugWithContext("Session synced successfully", map[string]interface{}{
			"project":    projectName,
			"session_id": sessionID,
		})

		result.SessionsSynced++
	}

	// GitRootの更新（セッションから取得できる場合）
	if len(sessionIDs) > 0 && result.SessionsSynced > 0 {
		// 最初のセッションからGitルートを推測
		// 実装は簡略化: ProjectPathからGitルートを設定することも可能
		// 現時点ではスキップ（将来の拡張として）
		_ = project
	}

	return result, nil
}

// isUniqueConstraintError checks if the error is a UNIQUE constraint violation
func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "UNIQUE constraint failed")
}
