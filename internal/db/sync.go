package db

import (
	"fmt"
	"strings"
	"time"

	"github.com/a-tak/ccloganalysis/internal/gitutil"
	"github.com/a-tak/ccloganalysis/internal/logger"
	"github.com/a-tak/ccloganalysis/internal/parser"
)

// SyncProgressCallback is called during sync to report progress
type SyncProgressCallback func(progress SyncProgressUpdate)

// SyncProgressUpdate represents incremental progress during sync
type SyncProgressUpdate struct {
	ProjectsProcessed int // 処理済みプロジェクト数
	SessionsFound     int // 発見したセッション数（累積、重複を含む）
	SessionsSynced    int // 同期したセッション数（累積、重複を除く）
	SessionsSkipped   int // スキップしたセッション数（累積、既存または重複）
	ErrorCount        int // エラー数（累積）
}

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
	return SyncAllWithCallback(db, p, nil)
}

// SyncAllWithCallback synchronizes all projects with progress callbacks
func SyncAllWithCallback(db *DB, p *parser.Parser, callback SyncProgressCallback) (*SyncResult, error) {
	return SyncAllWithLoggerAndCallback(db, p, logger.New(), callback)
}

// SyncAllWithLogger synchronizes all projects from the file system to the database with custom logger
func SyncAllWithLogger(db *DB, p *parser.Parser, log *logger.Logger) (*SyncResult, error) {
	return SyncAllWithLoggerAndCallback(db, p, log, nil)
}

// SyncAllWithLoggerAndCallback synchronizes all projects with custom logger and callbacks
func SyncAllWithLoggerAndCallback(db *DB, p *parser.Parser, log *logger.Logger, callback SyncProgressCallback) (*SyncResult, error) {
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
	for i, projectName := range projectNames {
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

			// エラー発生時もコールバックで進捗報告
			if callback != nil {
				callback(SyncProgressUpdate{
					ProjectsProcessed: result.ProjectsProcessed,
					SessionsFound:     result.SessionsFound,
					SessionsSynced:    result.SessionsSynced,
					SessionsSkipped:   result.SessionsSkipped,
					ErrorCount:        result.ErrorCount,
				})
			}
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

		// コールバックで進捗を報告
		if callback != nil {
			callback(SyncProgressUpdate{
				ProjectsProcessed: result.ProjectsProcessed,
				SessionsFound:     result.SessionsFound,
				SessionsSynced:    result.SessionsSynced,
				SessionsSkipped:   result.SessionsSkipped,
				ErrorCount:        result.ErrorCount,
			})
		}

		// 10プロジェクトごと、または最後のプロジェクトの場合にグループを同期
		// これにより、スキャン進行中でもグループが徐々に表示される
		if (i+1)%10 == 0 || i == len(projectNames)-1 {
			if err := db.SyncProjectGroups(); err != nil {
				log.WarnWithContext("Failed to sync project groups during scan", map[string]interface{}{
					"processed": i + 1,
					"error":     err.Error(),
				})
			} else {
				log.DebugWithContext("Project groups synchronized", map[string]interface{}{
					"processed": i + 1,
				})
			}
		}
	}

	log.InfoWithContext("SyncAll completed", map[string]interface{}{
		"projects_processed": result.ProjectsProcessed,
		"sessions_synced":    result.SessionsSynced,
		"sessions_found":     result.SessionsFound,
		"error_count":        result.ErrorCount,
	})

	// プロジェクトグループを自動的に同期
	if err := db.SyncProjectGroups(); err != nil {
		log.WarnWithContext("Failed to sync project groups", map[string]interface{}{
			"error": err.Error(),
		})
		// エラーが発生しても処理を続行（同期処理全体は失敗させない）
	}

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

// shouldSyncSession determines if a session should be synced based on file modification time
func shouldSyncSession(sessionID string, fileModTime time.Time, lastScanTime *time.Time, database *DB, log *logger.Logger) (bool, error) {
	// lastScanTimeとファイルModTimeを直接比較する
	// （ファイル名とDBのセッションID（UUID）が異なるため、DBでの検索は使えない）

	if lastScanTime == nil {
		// 初回スキャン直後など、lastScanTimeがない場合は全て同期
		log.DebugWithContext("No last scan time, will sync", map[string]interface{}{
			"session_id": sessionID,
		})
		return true, nil
	}

	// 時刻の精度を秒単位に揃える（ファイルシステムとDBの精度の違いを吸収）
	fileModTimeTrunc := fileModTime.Truncate(time.Second)
	lastScanTimeTrunc := lastScanTime.Truncate(time.Second)

	if fileModTimeTrunc.After(lastScanTimeTrunc) {
		// ファイルが前回スキャン後に更新された → 同期する
		log.DebugWithContext("File modified after last scan, will sync", map[string]interface{}{
			"session_id":     sessionID,
			"file_mod_time":  fileModTimeTrunc,
			"last_scan_time": lastScanTimeTrunc,
		})
		return true, nil
	}

	// ファイルが前回スキャン前 → スキップ
	log.DebugWithContext("File not modified since last scan, will skip", map[string]interface{}{
		"session_id":     sessionID,
		"file_mod_time":  fileModTimeTrunc,
		"last_scan_time": lastScanTimeTrunc,
	})
	return false, nil
}

// SyncIncremental synchronizes only new or updated sessions since last scan
func SyncIncremental(db *DB, p *parser.Parser) (*SyncResult, error) {
	log := logger.New()
	return SyncIncrementalWithLogger(db, p, log)
}

// SyncIncrementalWithLogger synchronizes only new or updated sessions with custom logger
func SyncIncrementalWithLogger(database *DB, p *parser.Parser, log *logger.Logger) (*SyncResult, error) {
	result := &SyncResult{}
	scanStartTime := time.Now()

	log.Debug("Starting SyncIncremental")

	// プロジェクト一覧取得
	projectNames, err := p.ListProjects()
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	log.DebugWithContext("Projects found for incremental scan", map[string]interface{}{
		"count": len(projectNames),
	})

	// 新規プロジェクト検出フラグ
	hasNewProjects := false

	for _, projectName := range projectNames {
		// プロジェクトをDBから取得または作成
		project, err := database.GetProjectByName(projectName)
		if err != nil {
			// プロジェクトが存在しない場合は作成
			hasNewProjects = true

			decodedPath := parser.DecodeProjectPath(projectName)

			// 実際の作業ディレクトリを取得してGit Root検出
			workingDir, err := p.GetProjectWorkingDirectory(projectName)
			if err != nil {
				// セッションが存在しない、またはcwdが取得できない
				log.WarnWithContext("Could not get working directory", map[string]interface{}{
					"project": projectName,
					"error":   err.Error(),
				})
				projectID, err := database.CreateProject(projectName, decodedPath)
				if err != nil {
					log.ErrorWithContext("Failed to create project", map[string]interface{}{
						"project": projectName,
						"error":   err.Error(),
					})
					result.ErrorCount++
					continue
				}

				// 作成したプロジェクトを取得
				project, err = database.GetProjectByID(projectID)
				if err != nil {
					log.ErrorWithContext("Failed to get created project", map[string]interface{}{
						"project": projectName,
						"error":   err.Error(),
					})
					result.ErrorCount++
					continue
				}
			} else {
				// 実際の作業ディレクトリでGit Root検出
				gitRoot, gitErr := gitutil.DetectGitRoot(workingDir)
				if gitErr != nil {
					log.WarnWithContext("Failed to detect git root", map[string]interface{}{
						"project": projectName,
						"error":   gitErr.Error(),
					})
					projectID, err := database.CreateProject(projectName, decodedPath)
					if err != nil {
						log.ErrorWithContext("Failed to create project", map[string]interface{}{
							"project": projectName,
							"error":   err.Error(),
						})
						result.ErrorCount++
						continue
					}

					project, err = database.GetProjectByID(projectID)
					if err != nil {
						log.ErrorWithContext("Failed to get created project", map[string]interface{}{
							"project": projectName,
							"error":   err.Error(),
						})
						result.ErrorCount++
						continue
					}
				} else if gitRoot != "" {
					projectID, err := database.CreateProjectWithGitRoot(projectName, decodedPath, gitRoot)
					if err != nil {
						log.ErrorWithContext("Failed to create project with git root", map[string]interface{}{
							"project":  projectName,
							"git_root": gitRoot,
							"error":    err.Error(),
						})
						result.ErrorCount++
						continue
					}

					log.InfoWithContext("Created project with git root", map[string]interface{}{
						"project":  projectName,
						"git_root": gitRoot,
					})

					project, err = database.GetProjectByID(projectID)
					if err != nil {
						log.ErrorWithContext("Failed to get created project", map[string]interface{}{
							"project": projectName,
							"error":   err.Error(),
						})
						result.ErrorCount++
						continue
					}
				} else {
					projectID, err := database.CreateProject(projectName, decodedPath)
					if err != nil {
						log.ErrorWithContext("Failed to create project", map[string]interface{}{
							"project": projectName,
							"error":   err.Error(),
						})
						result.ErrorCount++
						continue
					}

					project, err = database.GetProjectByID(projectID)
					if err != nil {
						log.ErrorWithContext("Failed to get created project", map[string]interface{}{
							"project": projectName,
							"error":   err.Error(),
						})
						result.ErrorCount++
						continue
					}
				}
			}
		}

		// このプロジェクトのスキャン開始時点でのカウントを記録
		projectSessionsSyncedBefore := result.SessionsSynced
		projectSessionsSkippedBefore := result.SessionsSkipped

		// 前回のスキャン時刻を取得
		lastScanTime, err := database.GetProjectLastScanTime(project.ID)
		if err != nil {
			log.WarnWithContext("Failed to get last scan time", map[string]interface{}{
				"project": projectName,
				"error":   err.Error(),
			})
			// エラーでもスキャンは続行（lastScanTimeをnilとして扱う）
		}

		// セッションファイル一覧とmodTimeを取得
		sessionInfos, err := p.ListSessionsWithModTime(projectName)
		if err != nil {
			log.ErrorWithContext("Failed to list sessions with modTime", map[string]interface{}{
				"project": projectName,
				"error":   err.Error(),
			})
			result.ErrorCount++
			continue
		}

		result.SessionsFound += len(sessionInfos)

		log.DebugWithContext("Processing project sessions", map[string]interface{}{
			"project":        projectName,
			"sessions_found": len(sessionInfos),
			"last_scan_time": lastScanTime,
		})

		for _, info := range sessionInfos {
			// スキャン対象か判定
			shouldSync, err := shouldSyncSession(info.SessionID, info.ModTime, lastScanTime, database, log)
			if err != nil {
				log.ErrorWithContext("Failed to check sync status", map[string]interface{}{
					"project":    projectName,
					"session_id": info.SessionID,
					"error":      err.Error(),
				})
				result.ErrorCount++
				continue
			}

			if !shouldSync {
				result.SessionsSkipped++
				continue
			}

			// セッションをパース
			log.DebugWithContext("Parsing session", map[string]interface{}{
				"project":    projectName,
				"session_id": info.SessionID,
			})

			session, err := p.ParseSession(projectName, info.SessionID)
			if err != nil {
				log.ErrorWithContext("Failed to parse session", map[string]interface{}{
					"project":    projectName,
					"session_id": info.SessionID,
					"error":      err.Error(),
				})
				result.ErrorCount++
				continue
			}

			// セッションをDBに保存（file_mod_timeも保存）
			log.DebugWithContext("Saving session to DB", map[string]interface{}{
				"project":       projectName,
				"session_id":    info.SessionID,
				"file_mod_time": info.ModTime,
			})

			err = database.CreateSession(session, projectName, info.ModTime)
			if err != nil {
				if isUniqueConstraintError(err) {
					// セッションが既に存在する場合は更新
					log.DebugWithContext("Session already exists, updating", map[string]interface{}{
						"project":    projectName,
						"session_id": info.SessionID,
					})

					err = database.UpdateSession(session, projectName, info.ModTime)
					if err != nil {
						log.ErrorWithContext("Failed to update session", map[string]interface{}{
							"project":    projectName,
							"session_id": info.SessionID,
							"error":      err.Error(),
						})
						result.ErrorCount++
						continue
					}

					log.DebugWithContext("Session updated", map[string]interface{}{
						"project":    projectName,
						"session_id": info.SessionID,
					})
					result.SessionsSynced++
					continue
				}
				log.ErrorWithContext("Failed to save session", map[string]interface{}{
					"project":    projectName,
					"session_id": info.SessionID,
					"error":      err.Error(),
				})
				result.ErrorCount++
				continue
			}

			result.SessionsSynced++
		}

		// プロジェクトの最終スキャン時刻を更新
		err = database.UpdateProjectLastScanTime(project.ID, scanStartTime)
		if err != nil {
			log.WarnWithContext("Failed to update last scan time", map[string]interface{}{
				"project": projectName,
				"error":   err.Error(),
			})
			// 警告だけでエラーカウントはしない（次回のスキャンで再度チェックされるため）
		}

		result.ProjectsProcessed++

		// このプロジェクトで同期/スキップされたセッション数を計算
		projectSessionsSynced := result.SessionsSynced - projectSessionsSyncedBefore
		projectSessionsSkipped := result.SessionsSkipped - projectSessionsSkippedBefore

		// 変更があったプロジェクトのみINFOレベルでログ出力
		if projectSessionsSynced > 0 {
			log.InfoWithContext("Project scan completed", map[string]interface{}{
				"project":          projectName,
				"sessions_synced":  projectSessionsSynced,
				"sessions_skipped": projectSessionsSkipped,
			})
		} else {
			log.DebugWithContext("Project scan completed (no changes)", map[string]interface{}{
				"project":          projectName,
				"sessions_skipped": projectSessionsSkipped,
			})
		}
	}

	// 新規プロジェクトが検出された場合のみグループを同期
	if hasNewProjects {
		if err := database.SyncProjectGroups(); err != nil {
			log.WarnWithContext("Failed to sync project groups", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			log.Info("Project groups synchronized after new projects detected")
		}
	}

	// 変更があった場合のみINFOレベル、なければDEBUGレベルでログ出力
	if result.SessionsSynced > 0 {
		log.InfoWithContext("SyncIncremental completed", map[string]interface{}{
			"projects_processed": result.ProjectsProcessed,
			"sessions_synced":    result.SessionsSynced,
			"sessions_skipped":   result.SessionsSkipped,
			"errors":             result.ErrorCount,
		})
	} else {
		log.DebugWithContext("SyncIncremental completed (no changes)", map[string]interface{}{
			"projects_processed": result.ProjectsProcessed,
			"sessions_skipped":   result.SessionsSkipped,
		})
	}

	return result, nil
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

		// 実際の作業ディレクトリを取得
		workingDir, err := p.GetProjectWorkingDirectory(projectName)
		if err != nil {
			// セッションが存在しない、またはcwdが取得できない
			log.WarnWithContext("Could not get working directory", map[string]interface{}{
				"project": projectName,
				"error":   err.Error(),
			})
			_, err = db.CreateProject(projectName, decodedPath)
		} else {
			// 実際の作業ディレクトリでGit Root検出
			gitRoot, gitErr := gitutil.DetectGitRoot(workingDir)
			if gitErr != nil {
				log.WarnWithContext("Failed to detect git root", map[string]interface{}{
					"project": projectName,
					"error":   gitErr.Error(),
				})
				_, err = db.CreateProject(projectName, decodedPath)
			} else if gitRoot != "" {
				_, err = db.CreateProjectWithGitRoot(projectName, decodedPath, gitRoot)
			} else {
				_, err = db.CreateProject(projectName, decodedPath)
			}
		}

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
	} else if project.GitRoot == nil {
		// 既存プロジェクトでGit Rootが未設定の場合、検出して更新
		workingDir, err := p.GetProjectWorkingDirectory(projectName)
		if err != nil {
			log.WarnWithContext("Could not get working directory for existing project", map[string]interface{}{
				"project": projectName,
				"error":   err.Error(),
			})
		} else {
			gitRoot, gitErr := gitutil.DetectGitRoot(workingDir)
			if gitErr != nil {
				log.WarnWithContext("Failed to detect git root for existing project", map[string]interface{}{
					"project": projectName,
					"error":   gitErr.Error(),
				})
			} else if gitRoot != "" {
				// Git Root検出成功、更新
				err = db.UpdateProjectGitRoot(project.ID, gitRoot)
				if err != nil {
					log.WarnWithContext("Failed to update git root", map[string]interface{}{
						"project": projectName,
						"error":   err.Error(),
					})
				} else {
					log.InfoWithContext("Updated git root for project", map[string]interface{}{
						"project":  projectName,
						"git_root": gitRoot,
					})
				}
			}
		}
	}

	// セッション一覧をmodTimeと共に取得
	sessionInfos, err := p.ListSessionsWithModTime(projectName)
	if err != nil {
		log.ErrorWithContext("Failed to list sessions", map[string]interface{}{
			"project": projectName,
			"error":   err.Error(),
		})
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	result.SessionsFound = len(sessionInfos)

	log.InfoWithContext("Sessions found in filesystem", map[string]interface{}{
		"project": projectName,
		"count":   result.SessionsFound,
	})

	// 各セッションを同期
	for _, info := range sessionInfos {
		log.DebugWithContext("Processing session", map[string]interface{}{
			"project":    projectName,
			"session_id": info.SessionID,
		})

		// セッションが既にDBに存在するかチェック
		existingSession, err := db.GetSession(info.SessionID)
		if err == nil {
			// 既に存在する場合はスキップ
			log.DebugWithContext("Session already exists, skipping", map[string]interface{}{
				"project":           projectName,
				"session_id":        info.SessionID,
				"existing_project":  existingSession.ProjectPath,
			})
			result.SessionsSkipped++
			continue
		} else {
			log.DebugWithContext("Session not found in DB, will sync", map[string]interface{}{
				"project":    projectName,
				"session_id": info.SessionID,
				"error":      err.Error(),
			})
		}

		// セッションをパース
		log.DebugWithContext("Parsing session", map[string]interface{}{
			"project":    projectName,
			"session_id": info.SessionID,
		})
		session, err := p.ParseSession(projectName, info.SessionID)
		if err != nil {
			errMsg := fmt.Sprintf("%s/%s: %v", projectName, info.SessionID, err)
			log.ErrorWithContext("Failed to parse session", map[string]interface{}{
				"project":    projectName,
				"session_id": info.SessionID,
				"error":      err.Error(),
			})
			result.ErrorCount++
			result.Errors = append(result.Errors, errMsg)
			continue
		}

		// セッションをDBに保存（ファイルmodTimeを含む）
		log.DebugWithContext("Saving session to DB", map[string]interface{}{
			"project":       projectName,
			"session_id":    info.SessionID,
			"file_mod_time": info.ModTime,
		})
		err = db.CreateSession(session, projectName, info.ModTime)
		if err != nil {
			// UNIQUE制約エラーの場合は、既に別のプロジェクトで登録済みとしてスキップ
			if isUniqueConstraintError(err) {
				log.DebugWithContext("Session already exists (duplicate), skipping", map[string]interface{}{
					"project":       projectName,
					"session_id":    info.SessionID,
					"error_message": err.Error(),
				})
				result.SessionsSkipped++
				continue
			}

			// その他のエラーはエラーとしてカウント
			errMsg := fmt.Sprintf("%s/%s: %v", projectName, info.SessionID, err)
			log.ErrorWithContext("Failed to save session", map[string]interface{}{
				"project":    projectName,
				"session_id": info.SessionID,
				"error":      err.Error(),
			})
			result.ErrorCount++
			result.Errors = append(result.Errors, errMsg)
			continue
		}

		log.DebugWithContext("Session synced successfully", map[string]interface{}{
			"project":    projectName,
			"session_id": info.SessionID,
		})

		result.SessionsSynced++
	}

	// プロジェクトの最終スキャン時刻を更新
	scanStartTime := time.Now()
	err = db.UpdateProjectLastScanTime(project.ID, scanStartTime)
	if err != nil {
		log.WarnWithContext("Failed to update last scan time", map[string]interface{}{
			"project": projectName,
			"error":   err.Error(),
		})
		// 警告だけでエラーカウントはしない
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
