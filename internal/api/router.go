package api

import (
	"encoding/json"
	"net/http"
)

// Handler holds the dependencies for HTTP handlers
type Handler struct {
	service SessionService
}

// NewHandler creates a new Handler with the given service
func NewHandler(service SessionService) *Handler {
	return &Handler{service: service}
}

// Routes returns the HTTP handler with all routes configured
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("GET /api/health", h.healthHandler)
	mux.HandleFunc("GET /api/projects", h.listProjectsHandler)
	mux.HandleFunc("GET /api/sessions", h.listSessionsHandler)
	mux.HandleFunc("GET /api/sessions/{project}/{id}", h.getSessionHandler)
	mux.HandleFunc("POST /api/analyze", h.analyzeHandler)

	// TODO: Serve static files for React frontend
	// mux.Handle("/", http.FileServer(http.FS(staticFiles)))

	return mux
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
