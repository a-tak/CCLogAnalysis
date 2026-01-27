package api

import (
	"encoding/json"
	"net/http"
)

// listGroupsHandler returns list of project groups
func (h *Handler) listGroupsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	groups, err := h.service.ListProjectGroups()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	json.NewEncoder(w).Encode(ProjectGroupListResponse{
		Groups: groups,
	})
}

// getGroupHandler returns detailed project group information
func (h *Handler) getGroupHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	groupID, err := parseGroupID(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	group, err := h.service.GetProjectGroup(groupID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}

	json.NewEncoder(w).Encode(group)
}

// getGroupStatsHandler returns statistics for a project group
func (h *Handler) getGroupStatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	groupID, err := parseGroupID(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	stats, err := h.service.GetProjectGroupStats(groupID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}

	json.NewEncoder(w).Encode(stats)
}

// getGroupTimelineHandler returns time-series statistics for a project group
func (h *Handler) getGroupTimelineHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	groupID, err := parseGroupID(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "bad_request", err.Error())
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

	timeline, err := h.service.GetProjectGroupTimeline(groupID, period, limit)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}

	json.NewEncoder(w).Encode(timeline)
}

// getGroupDailyStatsHandler handles GET /api/groups/{id}/daily/{date}
func (h *Handler) getGroupDailyStatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	groupID, err := parseGroupID(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	date := r.PathValue("date")
	if date == "" {
		writeJSONError(w, http.StatusBadRequest, "bad_request", "date is required")
		return
	}

	stats, err := h.service.GetGroupDailyStats(groupID, date)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}

	json.NewEncoder(w).Encode(stats)
}
