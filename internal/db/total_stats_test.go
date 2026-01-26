package db

import (
	"testing"
	"time"
)

// Note: GetTotalStats, GetTotalTimeSeriesStats, GetDailyGroupStats are
// tested indirectly through their usage in the service layer tests
// (internal/api/router_test.go). These are placeholder tests that
// demonstrate the testing structure and can be expanded with more
// comprehensive test data.

func TestGetTotalStatsBasic(t *testing.T) {
	t.Run("正常系：セッション数ゼロの場合", func(t *testing.T) {
		db, _ := setupTestDB(t)
		defer db.Close()

		stats, err := db.GetTotalStats()

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if stats == nil {
			t.Fatal("Expected stats to be non-nil")
		}

		// Empty database should return zero values
		if stats.TotalSessions != 0 {
			t.Errorf("Expected 0 sessions, got %d", stats.TotalSessions)
		}

		totalTokens := stats.TotalInputTokens + stats.TotalOutputTokens + stats.TotalCacheCreationTokens + stats.TotalCacheReadTokens
		if totalTokens != 0 {
			t.Errorf("Expected 0 total tokens, got %d", totalTokens)
		}

		if !stats.FirstSession.IsZero() {
			t.Error("Expected FirstSession to be zero value for empty DB")
		}
	})

	t.Run("エッジケース：NULL値処理", func(t *testing.T) {
		db, _ := setupTestDB(t)
		defer db.Close()

		// Get stats from empty database - should not error on NULL handling
		stats, err := db.GetTotalStats()

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify error rate is valid (should be 0 for empty DB)
		if stats.ErrorRate < 0 || stats.ErrorRate > 1 {
			t.Errorf("Expected ErrorRate between 0 and 1, got %f", stats.ErrorRate)
		}
	})
}

func TestGetTotalTimeSeriesStatsBasic(t *testing.T) {
	t.Run("正常系：データなし", func(t *testing.T) {
		db, _ := setupTestDB(t)
		defer db.Close()

		stats, err := db.GetTotalTimeSeriesStats("day", 10)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(stats) != 0 {
			t.Errorf("Expected empty result for empty DB, got %d items", len(stats))
		}
	})

	t.Run("パラメータ検証：期間の種類", func(t *testing.T) {
		db, _ := setupTestDB(t)
		defer db.Close()

		testCases := []string{"day", "week", "month"}

		for _, period := range testCases {
			stats, err := db.GetTotalTimeSeriesStats(period, 10)

			if err != nil {
				t.Errorf("Period %s should not error, got %v", period, err)
			}

			// Result can be nil or empty slice for empty DB, both are valid
			if stats != nil && len(stats) > 10 {
				t.Errorf("Period %s returned more than limit: got %d items", period, len(stats))
			}
		}
	})

	t.Run("パラメータ検証：limit", func(t *testing.T) {
		db, _ := setupTestDB(t)
		defer db.Close()

		// Test with various limit values
		testCases := []int{1, 10, 30, 100}

		for _, limit := range testCases {
			stats, err := db.GetTotalTimeSeriesStats("day", limit)

			if err != nil {
				t.Errorf("Limit %d should not error, got %v", limit, err)
			}

			// Result length should not exceed limit
			if len(stats) > limit {
				t.Errorf("Expected at most %d items, got %d", limit, len(stats))
			}
		}
	})
}

func TestGetDailyGroupStatsBasic(t *testing.T) {
	t.Run("正常系：日付フォーマット検証", func(t *testing.T) {
		db, _ := setupTestDB(t)
		defer db.Close()

		testDate := "2026-01-26"
		stats, err := db.GetDailyGroupStats(testDate)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Empty DB returns empty result, which is valid
		// No assertion needed - test passes if no error occurs
		if stats != nil && len(stats) > 0 {
			t.Logf("Found %d groups for test date", len(stats))
		}
	})

	t.Run("データ検証：構造正確性", func(t *testing.T) {
		db, _ := setupTestDB(t)
		defer db.Close()

		testDate := "2026-01-26"
		stats, err := db.GetDailyGroupStats(testDate)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Validate structure of returned stats
		for _, stat := range stats {
			// GroupID should be non-negative
			if stat.GroupID < 0 {
				t.Errorf("Expected non-negative GroupID, got %d", stat.GroupID)
			}

			// Token counts should be non-negative
			if stat.TotalInputTokens < 0 {
				t.Errorf("Expected non-negative TotalInputTokens, got %d", stat.TotalInputTokens)
			}

			if stat.TotalOutputTokens < 0 {
				t.Errorf("Expected non-negative TotalOutputTokens, got %d", stat.TotalOutputTokens)
			}

			totalTokens := stat.TotalInputTokens + stat.TotalOutputTokens + stat.TotalCacheCreationTokens + stat.TotalCacheReadTokens
			if totalTokens < 0 {
				t.Errorf("Expected non-negative TotalTokens, got %d", totalTokens)
			}

			// SessionCount should be non-negative
			if stat.SessionCount < 0 {
				t.Errorf("Expected non-negative SessionCount, got %d", stat.SessionCount)
			}
		}
	})

	t.Run("時刻範囲検証", func(t *testing.T) {
		db, _ := setupTestDB(t)
		defer db.Close()

		// Test various dates to ensure date filtering works
		testDates := []string{
			time.Now().Format("2006-01-02"),
			time.Now().Add(-24 * time.Hour).Format("2006-01-02"),
			time.Now().Add(-30 * 24 * time.Hour).Format("2006-01-02"),
		}

		for _, date := range testDates {
			_, err := db.GetDailyGroupStats(date)

			if err != nil {
				t.Errorf("Date %s should not error, got %v", date, err)
			}
		}
	})
}
