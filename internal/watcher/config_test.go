package watcher

import (
	"os"
	"testing"
	"time"
)

func TestLoadWatcherConfig_Defaults(t *testing.T) {
	// 環境変数をクリア
	os.Unsetenv("ENABLE_FILE_WATCH")
	os.Unsetenv("FILE_WATCH_INTERVAL")
	os.Unsetenv("FILE_WATCH_DEBOUNCE")

	config := LoadWatcherConfig()

	if config.Enabled {
		t.Errorf("Expected Enabled to be false by default, got true")
	}
	if config.Interval != 15*time.Second {
		t.Errorf("Expected Interval to be 15s, got %v", config.Interval)
	}
	if config.Debounce != 5*time.Second {
		t.Errorf("Expected Debounce to be 5s, got %v", config.Debounce)
	}
}

func TestLoadWatcherConfig_Enabled(t *testing.T) {
	os.Setenv("ENABLE_FILE_WATCH", "true")
	defer os.Unsetenv("ENABLE_FILE_WATCH")

	config := LoadWatcherConfig()

	if !config.Enabled {
		t.Errorf("Expected Enabled to be true, got false")
	}
}

func TestLoadWatcherConfig_CustomInterval(t *testing.T) {
	os.Setenv("FILE_WATCH_INTERVAL", "30")
	defer os.Unsetenv("FILE_WATCH_INTERVAL")

	config := LoadWatcherConfig()

	if config.Interval != 30*time.Second {
		t.Errorf("Expected Interval to be 30s, got %v", config.Interval)
	}
}

func TestLoadWatcherConfig_CustomDebounce(t *testing.T) {
	os.Setenv("FILE_WATCH_DEBOUNCE", "10")
	defer os.Unsetenv("FILE_WATCH_DEBOUNCE")

	config := LoadWatcherConfig()

	if config.Debounce != 10*time.Second {
		t.Errorf("Expected Debounce to be 10s, got %v", config.Debounce)
	}
}

func TestLoadWatcherConfig_BoundaryValues(t *testing.T) {
	tests := []struct {
		name     string
		envVar   string
		envValue string
		expected time.Duration
	}{
		{
			name:     "Interval minimum (5s)",
			envVar:   "FILE_WATCH_INTERVAL",
			envValue: "5",
			expected: 5 * time.Second,
		},
		{
			name:     "Interval below minimum (clamped to 5s)",
			envVar:   "FILE_WATCH_INTERVAL",
			envValue: "1",
			expected: 5 * time.Second,
		},
		{
			name:     "Interval maximum (3600s)",
			envVar:   "FILE_WATCH_INTERVAL",
			envValue: "3600",
			expected: 3600 * time.Second,
		},
		{
			name:     "Interval above maximum (clamped to 3600s)",
			envVar:   "FILE_WATCH_INTERVAL",
			envValue: "5000",
			expected: 3600 * time.Second,
		},
		{
			name:     "Debounce minimum (1s)",
			envVar:   "FILE_WATCH_DEBOUNCE",
			envValue: "1",
			expected: 1 * time.Second,
		},
		{
			name:     "Debounce below minimum (clamped to 1s)",
			envVar:   "FILE_WATCH_DEBOUNCE",
			envValue: "0",
			expected: 1 * time.Second,
		},
		{
			name:     "Debounce maximum (60s)",
			envVar:   "FILE_WATCH_DEBOUNCE",
			envValue: "60",
			expected: 60 * time.Second,
		},
		{
			name:     "Debounce above maximum (clamped to 60s)",
			envVar:   "FILE_WATCH_DEBOUNCE",
			envValue: "100",
			expected: 60 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(tt.envVar, tt.envValue)
			defer os.Unsetenv(tt.envVar)

			config := LoadWatcherConfig()

			var actual time.Duration
			if tt.envVar == "FILE_WATCH_INTERVAL" {
				actual = config.Interval
			} else {
				actual = config.Debounce
			}

			if actual != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, actual)
			}
		})
	}
}

func TestLoadWatcherConfig_InvalidValues(t *testing.T) {
	tests := []struct {
		name     string
		envVar   string
		envValue string
		expected time.Duration
	}{
		{
			name:     "Invalid interval (non-numeric) - use default",
			envVar:   "FILE_WATCH_INTERVAL",
			envValue: "invalid",
			expected: 15 * time.Second,
		},
		{
			name:     "Invalid debounce (non-numeric) - use default",
			envVar:   "FILE_WATCH_DEBOUNCE",
			envValue: "invalid",
			expected: 5 * time.Second,
		},
		{
			name:     "Negative interval - use default",
			envVar:   "FILE_WATCH_INTERVAL",
			envValue: "-10",
			expected: 15 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(tt.envVar, tt.envValue)
			defer os.Unsetenv(tt.envVar)

			config := LoadWatcherConfig()

			var actual time.Duration
			if tt.envVar == "FILE_WATCH_INTERVAL" {
				actual = config.Interval
			} else {
				actual = config.Debounce
			}

			if actual != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, actual)
			}
		})
	}
}
