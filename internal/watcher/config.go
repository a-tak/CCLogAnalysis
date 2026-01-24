package watcher

import (
	"os"
	"strconv"
	"time"
)

// WatcherConfig holds configuration for file watcher
type WatcherConfig struct {
	Enabled  bool
	Interval time.Duration
	Debounce time.Duration
}

const (
	defaultInterval = 15 // seconds
	minInterval     = 5  // seconds
	maxInterval     = 3600 // seconds

	defaultDebounce = 5  // seconds
	minDebounce     = 1  // seconds
	maxDebounce     = 60 // seconds
)

// LoadWatcherConfig loads watcher configuration from environment variables
func LoadWatcherConfig() WatcherConfig {
	enabled := os.Getenv("ENABLE_FILE_WATCH") == "true"

	interval := parseEnvDuration("FILE_WATCH_INTERVAL", defaultInterval, minInterval, maxInterval)
	debounce := parseEnvDuration("FILE_WATCH_DEBOUNCE", defaultDebounce, minDebounce, maxDebounce)

	return WatcherConfig{
		Enabled:  enabled,
		Interval: time.Duration(interval) * time.Second,
		Debounce: time.Duration(debounce) * time.Second,
	}
}

// parseEnvDuration parses an environment variable as integer duration in seconds
// Returns defaultValue if env var is not set, invalid, or out of bounds
func parseEnvDuration(envVar string, defaultValue, minValue, maxValue int) int {
	valueStr := os.Getenv(envVar)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		// Invalid value - use default
		return defaultValue
	}

	// Negative values are invalid
	if value < 0 {
		return defaultValue
	}

	// Clamp to min/max
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}

	return value
}
