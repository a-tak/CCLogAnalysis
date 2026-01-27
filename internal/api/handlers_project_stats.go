package api

import (
	"encoding/json"
	"net/http"
)

// getProjectStatsHandler returns project-level statistics
func (h *Handler) getProjectStatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	projectName := r.PathValue("name")
	if projectName == "" {
		writeJSONError(w, http.StatusBadRequest, "bad_request", "project name is required")
		return
	}

	stats, err := h.service.GetProjectStats(projectName)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}

	json.NewEncoder(w).Encode(stats)
}

// getProjectTimelineHandler returns time-series statistics for a project
func (h *Handler) getProjectTimelineHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	projectName := r.PathValue("name")
	if projectName == "" {
		writeJSONError(w, http.StatusBadRequest, "bad_request", "project name is required")
		return
	}

	period, err := parsePeriodParam(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	limit, err := parseLimitParam(r, 30)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	timeline, err := h.service.GetProjectTimeline(projectName, period, limit)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}

	json.NewEncoder(w).Encode(timeline)
}

// getProjectDailyStatsHandler handles GET /api/projects/{name}/daily/{date}
func (h *Handler) getProjectDailyStatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	projectName := r.PathValue("name")
	if projectName == "" {
		writeJSONError(w, http.StatusBadRequest, "bad_request", "project name is required")
		return
	}

	date := r.PathValue("date")
	if date == "" {
		writeJSONError(w, http.StatusBadRequest, "bad_request", "date is required")
		return
	}

	stats, err := h.service.GetProjectDailyStats(projectName, date)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}

	json.NewEncoder(w).Encode(stats)
}
