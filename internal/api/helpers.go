package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

// writeJSONError writes an error response with the specified status code
func writeJSONError(w http.ResponseWriter, status int, errorCode, message string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   errorCode,
		Message: message,
	})
}

// parseGroupID parses and validates group ID from path value
func parseGroupID(r *http.Request) (int64, error) {
	groupIDStr := r.PathValue("id")
	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid group ID")
	}
	return groupID, nil
}

// parsePeriodParam parses and validates period query parameter
func parsePeriodParam(r *http.Request) (string, error) {
	period := r.URL.Query().Get("period")
	if period == "" {
		return "day", nil
	}
	if period != "day" && period != "week" && period != "month" {
		return "", fmt.Errorf("period must be 'day', 'week', or 'month'")
	}
	return period, nil
}

// parseLimitParam parses and validates limit query parameter
func parseLimitParam(r *http.Request, defaultLimit int) (int, error) {
	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		return defaultLimit, nil
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		return 0, fmt.Errorf("limit must be a positive integer")
	}
	return limit, nil
}

// extractDisplayName extracts the last folder name from a decoded path
// Example: "C:/Users/username/projects/my-project" -> "my-project"
func extractDisplayName(decodedPath string) string {
	if decodedPath == "" {
		return ""
	}

	// Normalize path separators
	normalized := filepath.ToSlash(decodedPath)

	// Remove trailing slash if present
	normalized = strings.TrimSuffix(normalized, "/")

	// Extract last component
	parts := strings.Split(normalized, "/")
	if len(parts) == 0 {
		return decodedPath
	}

	return parts[len(parts)-1]
}
