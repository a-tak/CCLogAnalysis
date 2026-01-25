package api

import (
	"encoding/json"
	"net/http"
	"time"
)

// getScanStatusHandler returns the current scan status
func (h *Handler) getScanStatusHandler(w http.ResponseWriter, r *http.Request) {
	if h.scanManager == nil {
		w.WriteHeader(http.StatusNotImplemented)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "not_available",
			Message: "Scan manager not available",
		})
		return
	}

	progress := h.scanManager.GetProgress()

	var completedAt *string
	if progress.CompletedAt != nil {
		t := progress.CompletedAt.Format(time.RFC3339)
		completedAt = &t
	}

	response := ScanStatusResponse{
		Status:            string(progress.Status),
		ProjectsProcessed: progress.ProjectsProcessed,
		SessionsFound:     progress.SessionsFound,
		SessionsSynced:    progress.SessionsSynced,
		SessionsSkipped:   progress.SessionsSkipped,
		ErrorCount:        progress.ErrorCount,
		StartedAt:         progress.StartedAt.Format(time.RFC3339),
		CompletedAt:       completedAt,
		LastError:         progress.LastError,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
