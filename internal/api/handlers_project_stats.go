package api

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// getProjectStatsHandler returns project-level statistics
func (h *Handler) getProjectStatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	projectName := r.PathValue("name")
	if projectName == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "bad_request",
			Message: "project name is required",
		})
		return
	}

	stats, err := h.service.GetProjectStats(projectName)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "not_found",
			Message: err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(stats)
}

// getProjectTimelineHandler returns time-series statistics for a project
func (h *Handler) getProjectTimelineHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	projectName := r.PathValue("name")
	if projectName == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "bad_request",
			Message: "project name is required",
		})
		return
	}

	// クエリパラメータ取得
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "day"
	}

	// periodのバリデーション
	if period != "day" && period != "week" && period != "month" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "bad_request",
			Message: "period must be 'day', 'week', or 'month'",
		})
		return
	}

	limit := 30
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	timeline, err := h.service.GetProjectTimeline(projectName, period, limit)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "not_found",
			Message: err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(timeline)
}
