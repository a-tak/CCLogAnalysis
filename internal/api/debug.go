package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/a-tak/ccloganalysis/internal/db"
	"github.com/a-tak/ccloganalysis/internal/scanner"
)

// DebugStatusResponse represents the response for debug status endpoint
type DebugStatusResponse struct {
	DBProjects        int        `json:"db_projects"`
	DBSessions        int        `json:"db_sessions"`
	FSProjects        int        `json:"fs_projects"`
	SyncStatus        string     `json:"sync_status"`
	SyncError         string     `json:"sync_error,omitempty"`
	ScanStatus        string     `json:"scan_status"`
	ProjectsProcessed int        `json:"projects_processed"`
	SessionsSynced    int        `json:"sessions_synced"`
	SessionsSkipped   int        `json:"sessions_skipped"`
	ScanStartedAt     *time.Time `json:"scan_started_at,omitempty"`
	ScanCompletedAt   *time.Time `json:"scan_completed_at,omitempty"`
}

// DebugStatusHandler returns a handler for the debug status endpoint
func DebugStatusHandler(service *DatabaseSessionService, scanManager *scanner.ScanManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// DBからプロジェクト数を取得
		projects, err := service.db.ListProjects()
		dbProjects := 0
		if err == nil {
			dbProjects = len(projects)
		}

		// DBからセッション数を取得
		sessions, err := service.db.ListSessions(nil, 10000, 0)
		dbSessions := 0
		if err == nil {
			dbSessions = len(sessions)
		}

		// ファイルシステムからプロジェクト数を取得
		fsProjects := 0
		if service.parser != nil {
			projectNames, err := service.parser.ListProjects()
			if err == nil {
				fsProjects = len(projectNames)
			}
		}

		// 同期ステータスを取得
		syncStatus := "not_synced"
		syncError := ""

		if service.syncError != nil {
			syncStatus = "failed"
			syncError = service.syncError.Error()
		} else if dbProjects > 0 {
			syncStatus = "success"
		}

		// スキャンマネージャーの進捗を取得
		progress := scanManager.GetProgress()

		// レスポンスを作成
		response := DebugStatusResponse{
			DBProjects:        dbProjects,
			DBSessions:        dbSessions,
			FSProjects:        fsProjects,
			SyncStatus:        syncStatus,
			SyncError:         syncError,
			ScanStatus:        string(progress.Status),
			ProjectsProcessed: progress.ProjectsProcessed,
			SessionsSynced:    progress.SessionsSynced,
			SessionsSkipped:   progress.SessionsSkipped,
			ScanStartedAt:     &progress.StartedAt,
			ScanCompletedAt:   progress.CompletedAt,
		}

		// JSONレスポンスを返す
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

// DebugSyncResponse represents the response for debug sync endpoint
type DebugSyncResponse struct {
	ProjectsProcessed int      `json:"projects_processed"`
	SessionsFound     int      `json:"sessions_found"`
	SessionsSynced    int      `json:"sessions_synced"`
	SessionsSkipped   int      `json:"sessions_skipped"`
	ErrorCount        int      `json:"error_count"`
	Errors            []string `json:"errors,omitempty"`
}

// DebugSyncHandler returns a handler for the debug sync endpoint (triggers manual sync)
func DebugSyncHandler(service *DatabaseSessionService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// POST method only
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Execute sync
		result, err := db.SyncAll(service.db, service.parser)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": err.Error(),
			})
			return
		}

		// Create response
		response := DebugSyncResponse{
			ProjectsProcessed: result.ProjectsProcessed,
			SessionsFound:     result.SessionsFound,
			SessionsSynced:    result.SessionsSynced,
			SessionsSkipped:   result.SessionsSkipped,
			ErrorCount:        result.ErrorCount,
			Errors:            result.Errors,
		}

		// Return JSON response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}
