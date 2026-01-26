package api

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// getTotalStatsHandler returns total statistics across all projects
func (h *Handler) getTotalStatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	stats, err := h.service.GetTotalStats()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "internal_error",
			Message: err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(stats)
}

// getTotalTimelineHandler returns time-series statistics across all projects
func (h *Handler) getTotalTimelineHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

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
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil || parsedLimit <= 0 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error:   "bad_request",
				Message: "limit must be a positive integer",
			})
			return
		}
		limit = parsedLimit
	}

	timeline, err := h.service.GetTotalTimeline(period, limit)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "internal_error",
			Message: err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(timeline)
}

// getDailyStatsHandler returns group-wise statistics for a specific date
func (h *Handler) getDailyStatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	date := r.PathValue("date")
	if date == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "bad_request",
			Message: "date is required",
		})
		return
	}

	stats, err := h.service.GetDailyStats(date)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "internal_error",
			Message: err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(stats)
}
