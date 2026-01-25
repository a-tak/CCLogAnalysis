package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetProjectStatsHandler(t *testing.T) {
	// モックサービスを使用
	now := time.Now()
	mockService := &MockSessionService{
		stats: &ProjectStatsResponse{
			TotalSessions:            1,
			TotalInputTokens:         1000,
			TotalOutputTokens:        500,
			TotalCacheCreationTokens: 100,
			TotalCacheReadTokens:     200,
			TotalTokens:              1500,
			AvgTokens:                1500,
			FirstSession:             now,
			LastSession:              now.Add(time.Hour),
			ErrorRate:                0,
		},
	}
	handler := NewHandler(mockService)

	// リクエスト作成
	req := httptest.NewRequest(http.MethodGet, "/api/projects/test-project/stats", nil)
	req.SetPathValue("name", "test-project")
	w := httptest.NewRecorder()

	// ハンドラー実行
	handler.getProjectStatsHandler(w, req)

	// レスポンス検証
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response ProjectStatsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.TotalSessions != 1 {
		t.Errorf("Expected 1 session, got %d", response.TotalSessions)
	}
	if response.TotalInputTokens != 1000 {
		t.Errorf("Expected 1000 input tokens, got %d", response.TotalInputTokens)
	}
	if response.TotalTokens != 1500 {
		t.Errorf("Expected 1500 total tokens, got %d", response.TotalTokens)
	}
}

func TestGetProjectStatsHandlerNotFound(t *testing.T) {
	// エラーを返すモックサービス
	mockService := &MockSessionService{
		err: fmt.Errorf("project not found"),
	}
	handler := NewHandler(mockService)

	// 存在しないプロジェクトでリクエスト
	req := httptest.NewRequest(http.MethodGet, "/api/projects/nonexistent/stats", nil)
	req.SetPathValue("name", "nonexistent")
	w := httptest.NewRecorder()

	// ハンドラー実行
	handler.getProjectStatsHandler(w, req)

	// レスポンス検証（404を期待）
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestGetProjectTimelineHandler(t *testing.T) {
	// モックサービスを使用
	mockService := &MockSessionService{
		timeline: &TimeSeriesResponse{
			Period: "day",
			Data: []TimeSeriesDataPoint{
				{
					PeriodStart:      time.Date(2026, 1, 21, 0, 0, 0, 0, time.UTC),
					PeriodEnd:        time.Date(2026, 1, 21, 0, 0, 0, 0, time.UTC),
					SessionCount:     1,
					TotalInputTokens: 1000,
					TotalOutputTokens: 500,
					TotalTokens:      1500,
				},
				{
					PeriodStart:      time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC),
					PeriodEnd:        time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC),
					SessionCount:     1,
					TotalInputTokens: 1000,
					TotalOutputTokens: 500,
					TotalTokens:      1500,
				},
			},
		},
	}
	handler := NewHandler(mockService)

	// リクエスト作成（デフォルトはday）
	req := httptest.NewRequest(http.MethodGet, "/api/projects/test-project/timeline", nil)
	req.SetPathValue("name", "test-project")
	w := httptest.NewRecorder()

	// ハンドラー実行
	handler.getProjectTimelineHandler(w, req)

	// レスポンス検証
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response TimeSeriesResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Period != "day" {
		t.Errorf("Expected period 'day', got %s", response.Period)
	}

	if len(response.Data) != 2 {
		t.Fatalf("Expected 2 data points, got %d", len(response.Data))
	}
}

func TestGetProjectTimelineHandlerWithPeriod(t *testing.T) {
	// モックサービスを使用
	mockService := &MockSessionService{
		timeline: &TimeSeriesResponse{
			Period: "month",
			Data: []TimeSeriesDataPoint{
				{
					PeriodStart:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
					PeriodEnd:        time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC),
					SessionCount:     1,
					TotalInputTokens: 1000,
					TotalOutputTokens: 500,
					TotalTokens:      1500,
				},
			},
		},
	}
	handler := NewHandler(mockService)

	// period=monthでリクエスト
	req := httptest.NewRequest(http.MethodGet, "/api/projects/test-project/timeline?period=month", nil)
	req.SetPathValue("name", "test-project")
	w := httptest.NewRecorder()

	// ハンドラー実行
	handler.getProjectTimelineHandler(w, req)

	// レスポンス検証
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response TimeSeriesResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Period != "month" {
		t.Errorf("Expected period 'month', got %s", response.Period)
	}
}
