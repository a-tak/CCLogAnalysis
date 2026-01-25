package api

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strings"

	"github.com/a-tak/ccloganalysis/internal/scanner"
	"github.com/a-tak/ccloganalysis/internal/static"
)

// Handler holds the dependencies for HTTP handlers
type Handler struct {
	service     SessionService
	dbService   *DatabaseSessionService
	scanManager *scanner.ScanManager
}

// NewHandler creates a new Handler with the given service and scan manager
func NewHandler(service SessionService, scanManager *scanner.ScanManager) *Handler {
	// DatabaseSessionService の場合は dbService にも設定
	var dbService *DatabaseSessionService
	if dbs, ok := service.(*DatabaseSessionService); ok {
		dbService = dbs
	}

	return &Handler{
		service:     service,
		dbService:   dbService,
		scanManager: scanManager,
	}
}

// spaHandler serves static files and falls back to index.html for SPA routing
type spaHandler struct {
	staticFS   http.FileSystem
	indexBytes []byte
}

func newSPAHandler(fsys embed.FS) (*spaHandler, error) {
	// dist を取り除いた fs.FS を作成
	stripped, err := fs.Sub(fsys, "dist")
	if err != nil {
		return nil, err
	}

	// index.html を読み込んでキャッシュ
	indexBytes, err := fs.ReadFile(stripped, "index.html")
	if err != nil {
		return nil, err
	}

	return &spaHandler{
		staticFS:   http.FS(stripped),
		indexBytes: indexBytes,
	}, nil
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// APIパスは処理しない（念のため）
	if strings.HasPrefix(path, "/api/") {
		http.NotFound(w, r)
		return
	}

	// ルートパスの場合は index.html を返す
	if path == "/" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(h.indexBytes)
		return
	}

	// ファイルが存在するか確認
	cleanPath := strings.TrimPrefix(path, "/")
	if f, err := h.staticFS.Open(cleanPath); err == nil {
		defer f.Close()
		// ファイルが存在する場合は、http.StripPrefixを使ってFileServerで処理
		http.StripPrefix("/", http.FileServer(h.staticFS)).ServeHTTP(w, r)
		return
	}

	// ファイルが存在しない場合は index.html を返す（SPAルーティング）
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(h.indexBytes)
}

// corsMiddleware adds CORS headers to all responses
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 開発時は常にCORSを有効化（DISABLE_CORS=trueで無効化可能）
		if os.Getenv("DISABLE_CORS") != "true" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Routes returns the HTTP handler with all routes configured
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("GET /api/health", h.healthHandler)
	mux.HandleFunc("GET /api/projects", h.listProjectsHandler)
	mux.HandleFunc("GET /api/projects/{name}/stats", h.getProjectStatsHandler)
	mux.HandleFunc("GET /api/projects/{name}/timeline", h.getProjectTimelineHandler)
	mux.HandleFunc("GET /api/sessions", h.listSessionsHandler)
	mux.HandleFunc("GET /api/sessions/{project}/{id}", h.getSessionHandler)
	mux.HandleFunc("POST /api/analyze", h.analyzeHandler)
	mux.HandleFunc("GET /api/groups", h.listGroupsHandler)
	mux.HandleFunc("GET /api/groups/{id}", h.getGroupHandler)
	mux.HandleFunc("GET /api/groups/{id}/stats", h.getGroupStatsHandler)

	// Scan status endpoint
	mux.HandleFunc("GET /api/scan/status", h.getScanStatusHandler)

	// Debug endpoint (only available when using DatabaseSessionService)
	if h.dbService != nil {
		mux.HandleFunc("GET /api/debug/status", DebugStatusHandler(h.dbService))
		mux.HandleFunc("POST /api/debug/sync", DebugSyncHandler(h.dbService))
	}

	// Static files for React frontend
	spaHandler, err := newSPAHandler(static.Files)
	if err != nil {
		panic(fmt.Sprintf("Failed to create SPA handler: %v", err))
	}
	mux.Handle("/", spaHandler)

	// Wrap with CORS middleware
	return corsMiddleware(mux)
}

// healthHandler returns server health status
func (h *Handler) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HealthResponse{
		Status: "ok",
	})
}

// listProjectsHandler returns list of projects
func (h *Handler) listProjectsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	projects, err := h.service.ListProjects()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "internal_error",
			Message: err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(ProjectListResponse{
		Projects: projects,
	})
}

// listSessionsHandler returns list of sessions
func (h *Handler) listSessionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	projectName := r.URL.Query().Get("project")

	sessions, err := h.service.ListSessions(projectName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "internal_error",
			Message: err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(SessionListResponse{
		Sessions: sessions,
	})
}

// getSessionHandler returns a specific session
func (h *Handler) getSessionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	project := r.PathValue("project")
	id := r.PathValue("id")

	session, err := h.service.GetSession(project, id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "not_found",
			Message: err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(session)
}

// analyzeHandler triggers log analysis
func (h *Handler) analyzeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req AnalyzeRequest
	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error:   "invalid_request",
				Message: err.Error(),
			})
			return
		}
	}

	result, err := h.service.Analyze(req.ProjectNames)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "internal_error",
			Message: err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(result)
}
