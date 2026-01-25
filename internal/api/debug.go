package api

import (
	"encoding/json"
	"net/http"
)

// DebugStatusResponse represents the response for debug status endpoint
type DebugStatusResponse struct {
	DBProjects int    `json:"db_projects"`
	DBSessions int    `json:"db_sessions"`
	FSProjects int    `json:"fs_projects"`
	SyncStatus string `json:"sync_status"`
	SyncError  string `json:"sync_error,omitempty"`
}

// DebugStatusHandler returns a handler for the debug status endpoint
func DebugStatusHandler(service *DatabaseSessionService) http.HandlerFunc {
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

		// レスポンスを作成
		response := DebugStatusResponse{
			DBProjects: dbProjects,
			DBSessions: dbSessions,
			FSProjects: fsProjects,
			SyncStatus: syncStatus,
			SyncError:  syncError,
		}

		// JSONレスポンスを返す
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}
