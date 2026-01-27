package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"time"
)

// getTotalStatsHandler returns total statistics across all projects
func (h *Handler) getTotalStatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	stats, err := h.service.GetTotalStats()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to retrieve statistics")
		return
	}

	json.NewEncoder(w).Encode(stats)
}

// getTotalTimelineHandler returns time-series statistics across all projects
func (h *Handler) getTotalTimelineHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

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

	timeline, err := h.service.GetTotalTimeline(period, limit)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to retrieve timeline data")
		return
	}

	json.NewEncoder(w).Encode(timeline)
}

// isValidDateFormat validates if the date string is in YYYY-MM-DD format
func isValidDateFormat(dateStr string) bool {
	// Check format with regex
	pattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	if !pattern.MatchString(dateStr) {
		return false
	}
	// Verify it's a valid date
	_, err := time.Parse("2006-01-02", dateStr)
	return err == nil
}

// getDailyStatsHandler returns group-wise statistics for a specific date
func (h *Handler) getDailyStatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	date := r.PathValue("date")
	if date == "" {
		writeJSONError(w, http.StatusBadRequest, "bad_request", "date is required")
		return
	}

	if !isValidDateFormat(date) {
		writeJSONError(w, http.StatusBadRequest, "bad_request", "date must be in YYYY-MM-DD format")
		return
	}

	stats, err := h.service.GetDailyStats(date)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to retrieve daily statistics")
		return
	}

	json.NewEncoder(w).Encode(stats)
}
