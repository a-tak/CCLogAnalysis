package api

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// listGroupsHandler returns list of project groups
func (h *Handler) listGroupsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	groups, err := h.service.ListProjectGroups()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "internal_error",
			Message: err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(ProjectGroupListResponse{
		Groups: groups,
	})
}

// getGroupHandler returns detailed project group information
func (h *Handler) getGroupHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	groupIDStr := r.PathValue("id")
	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "bad_request",
			Message: "invalid group ID",
		})
		return
	}

	group, err := h.service.GetProjectGroup(groupID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "not_found",
			Message: err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(group)
}

// getGroupStatsHandler returns statistics for a project group
func (h *Handler) getGroupStatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	groupIDStr := r.PathValue("id")
	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "bad_request",
			Message: "invalid group ID",
		})
		return
	}

	stats, err := h.service.GetProjectGroupStats(groupID)
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
