package db

import (
	"reflect"
	"testing"
	"time"
)

func TestGenerateDateRange(t *testing.T) {
	tests := []struct {
		name      string
		startTime string
		endTime   string
		expected  []string
	}{
		{
			name:      "同日内のセッション",
			startTime: "2026-01-20T10:00:00Z",
			endTime:   "2026-01-20T15:00:00Z",
			expected:  []string{"2026-01-20"},
		},
		{
			name:      "跨日セッション（2日間）",
			startTime: "2026-01-20T23:00:00Z",
			endTime:   "2026-01-21T02:00:00Z",
			expected:  []string{"2026-01-20", "2026-01-21"},
		},
		{
			name:      "複数日にまたがるセッション（3日間）",
			startTime: "2026-01-20T10:00:00Z",
			endTime:   "2026-01-22T15:00:00Z",
			expected:  []string{"2026-01-20", "2026-01-21", "2026-01-22"},
		},
		{
			name:      "日付境界ぴったり",
			startTime: "2026-01-20T23:59:59Z",
			endTime:   "2026-01-21T00:00:01Z",
			expected:  []string{"2026-01-20", "2026-01-21"},
		},
		{
			name:      "長期セッション（7日間）",
			startTime: "2026-01-15T08:00:00Z",
			endTime:   "2026-01-21T18:00:00Z",
			expected: []string{
				"2026-01-15", "2026-01-16", "2026-01-17", "2026-01-18",
				"2026-01-19", "2026-01-20", "2026-01-21",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, _ := time.Parse(time.RFC3339, tt.startTime)
			end, _ := time.Parse(time.RFC3339, tt.endTime)

			result := generateDateRange(start, end)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("generateDateRange() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSessionCoversDate(t *testing.T) {
	tests := []struct {
		name       string
		startTime  string
		endTime    string
		targetDate string
		expected   bool
	}{
		{
			name:       "同日内のセッション - 対象日が一致",
			startTime:  "2026-01-20T10:00:00Z",
			endTime:    "2026-01-20T15:00:00Z",
			targetDate: "2026-01-20",
			expected:   true,
		},
		{
			name:       "同日内のセッション - 対象日が不一致",
			startTime:  "2026-01-20T10:00:00Z",
			endTime:    "2026-01-20T15:00:00Z",
			targetDate: "2026-01-21",
			expected:   false,
		},
		{
			name:       "跨日セッション - 開始日",
			startTime:  "2026-01-20T23:00:00Z",
			endTime:    "2026-01-21T02:00:00Z",
			targetDate: "2026-01-20",
			expected:   true,
		},
		{
			name:       "跨日セッション - 終了日",
			startTime:  "2026-01-20T23:00:00Z",
			endTime:    "2026-01-21T02:00:00Z",
			targetDate: "2026-01-21",
			expected:   true,
		},
		{
			name:       "跨日セッション - 範囲外",
			startTime:  "2026-01-20T23:00:00Z",
			endTime:    "2026-01-21T02:00:00Z",
			targetDate: "2026-01-22",
			expected:   false,
		},
		{
			name:       "複数日セッション - 中間日",
			startTime:  "2026-01-20T10:00:00Z",
			endTime:    "2026-01-22T15:00:00Z",
			targetDate: "2026-01-21",
			expected:   true,
		},
		{
			name:       "日付境界ぴったり - 開始日",
			startTime:  "2026-01-20T23:59:59Z",
			endTime:    "2026-01-21T00:00:01Z",
			targetDate: "2026-01-20",
			expected:   true,
		},
		{
			name:       "日付境界ぴったり - 終了日",
			startTime:  "2026-01-20T23:59:59Z",
			endTime:    "2026-01-21T00:00:01Z",
			targetDate: "2026-01-21",
			expected:   true,
		},
		{
			name:       "不正な日付フォーマット",
			startTime:  "2026-01-20T10:00:00Z",
			endTime:    "2026-01-20T15:00:00Z",
			targetDate: "invalid-date",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, _ := time.Parse(time.RFC3339, tt.startTime)
			end, _ := time.Parse(time.RFC3339, tt.endTime)

			result := sessionCoversDate(start, end, tt.targetDate)
			if result != tt.expected {
				t.Errorf("sessionCoversDate() = %v, want %v", result, tt.expected)
			}
		})
	}
}
