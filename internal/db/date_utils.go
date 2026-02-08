package db

import (
	"time"
)

// generateDateRange generates a list of dates that a session covers.
// Returns dates in "YYYY-MM-DD" format.
func generateDateRange(startTime, endTime time.Time) []string {
	// Convert to local date boundaries
	startDate := time.Date(startTime.Year(), startTime.Month(), startTime.Day(), 0, 0, 0, 0, startTime.Location())
	endDate := time.Date(endTime.Year(), endTime.Month(), endTime.Day(), 0, 0, 0, 0, endTime.Location())

	var dates []string
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dates = append(dates, d.Format("2006-01-02"))
	}

	return dates
}

// sessionCoversDate checks if a session was running on the specified date.
// targetDate should be in "YYYY-MM-DD" format.
func sessionCoversDate(startTime, endTime time.Time, targetDate string) bool {
	// Parse target date
	target, err := time.Parse("2006-01-02", targetDate)
	if err != nil {
		return false
	}

	// Get date boundaries
	startDate := time.Date(startTime.Year(), startTime.Month(), startTime.Day(), 0, 0, 0, 0, startTime.Location())
	endDate := time.Date(endTime.Year(), endTime.Month(), endTime.Day(), 0, 0, 0, 0, endTime.Location())
	targetDateStart := time.Date(target.Year(), target.Month(), target.Day(), 0, 0, 0, 0, target.Location())

	// Check if session covers the target date
	return !startDate.After(targetDateStart) && !endDate.Before(targetDateStart)
}
