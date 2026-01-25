package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/a-tak/ccloganalysis/internal/db"
	"github.com/a-tak/ccloganalysis/internal/parser"
	"github.com/a-tak/ccloganalysis/internal/scanner"
)

// stringPtr はテスト用のヘルパー関数で、文字列のポインタを返す
func stringPtr(s string) *string {
	return &s
}

func TestListGroupsHandler(t *testing.T) {
	t.Run("グループ一覧を取得できる", func(t *testing.T) {
		mockService := &MockSessionService{
			ProjectGroups: []ProjectGroupResponse{
				{
					ID:        1,
					Name:      "test-repo",
					GitRoot:   stringPtr("{git-root}/test-repo"),
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				{
					ID:        2,
					Name:      "another-repo",
					GitRoot:   stringPtr("{git-root}/another-repo"),
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			},
		}

		mockDB := &db.DB{}
	mockParser := parser.NewParser("/tmp")
	mockScanManager := scanner.NewScanManager(mockDB, mockParser)
	handler := NewHandler(mockService, mockScanManager)
		req := httptest.NewRequest("GET", "/api/groups", nil)
		w := httptest.NewRecorder()

		handler.listGroupsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response ProjectGroupListResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response.Groups) != 2 {
			t.Errorf("Expected 2 groups, got %d", len(response.Groups))
		}

		if response.Groups[0].Name != "test-repo" {
			t.Errorf("Expected first group name 'test-repo', got '%s'", response.Groups[0].Name)
		}
	})

	t.Run("空のグループリストを返す", func(t *testing.T) {
		mockService := &MockSessionService{
			ProjectGroups: []ProjectGroupResponse{},
		}

		mockDB := &db.DB{}
	mockParser := parser.NewParser("/tmp")
	mockScanManager := scanner.NewScanManager(mockDB, mockParser)
	handler := NewHandler(mockService, mockScanManager)
		req := httptest.NewRequest("GET", "/api/groups", nil)
		w := httptest.NewRecorder()

		handler.listGroupsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response ProjectGroupListResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response.Groups) != 0 {
			t.Errorf("Expected 0 groups, got %d", len(response.Groups))
		}
	})
}

func TestGetGroupHandler(t *testing.T) {
	t.Run("グループ詳細を取得できる", func(t *testing.T) {
		mockService := &MockSessionService{
			ProjectGroupDetail: &ProjectGroupDetailResponse{
				ID:        1,
				Name:      "test-repo",
				GitRoot:   stringPtr("{git-root}/test-repo"),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Projects: []ProjectResponse{
					{Name: "project-1", DecodedPath: "{project-path}/project1", SessionCount: 5},
					{Name: "project-2", DecodedPath: "{project-path}/project2", SessionCount: 3},
				},
			},
		}

		mockDB := &db.DB{}
	mockParser := parser.NewParser("/tmp")
	mockScanManager := scanner.NewScanManager(mockDB, mockParser)
	handler := NewHandler(mockService, mockScanManager)
		req := httptest.NewRequest("GET", "/api/groups/1", nil)
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()

		handler.getGroupHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response ProjectGroupDetailResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Name != "test-repo" {
			t.Errorf("Expected group name 'test-repo', got '%s'", response.Name)
		}

		if len(response.Projects) != 2 {
			t.Errorf("Expected 2 projects, got %d", len(response.Projects))
		}
	})

	t.Run("無効なグループIDでエラーを返す", func(t *testing.T) {
		mockService := &MockSessionService{}
		mockDB := &db.DB{}
	mockParser := parser.NewParser("/tmp")
	mockScanManager := scanner.NewScanManager(mockDB, mockParser)
	handler := NewHandler(mockService, mockScanManager)
		req := httptest.NewRequest("GET", "/api/groups/invalid", nil)
		req.SetPathValue("id", "invalid")
		w := httptest.NewRecorder()

		handler.getGroupHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("存在しないグループIDで404を返す", func(t *testing.T) {
		mockService := &MockSessionService{
			err: fmt.Errorf("group not found"),
		}
		mockDB := &db.DB{}
	mockParser := parser.NewParser("/tmp")
	mockScanManager := scanner.NewScanManager(mockDB, mockParser)
	handler := NewHandler(mockService, mockScanManager)
		req := httptest.NewRequest("GET", "/api/groups/999", nil)
		req.SetPathValue("id", "999")
		w := httptest.NewRecorder()

		handler.getGroupHandler(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})
}

func TestGetGroupStatsHandler(t *testing.T) {
	t.Run("グループ統計を取得できる", func(t *testing.T) {
		mockService := &MockSessionService{
			ProjectGroupStats: &ProjectGroupStatsResponse{
				TotalProjects:     2,
				TotalSessions:     8,
				TotalInputTokens:  4000,
				TotalOutputTokens: 2000,
				AvgTokens:         750,
				FirstSession:      time.Now().Add(-7 * 24 * time.Hour),
				LastSession:       time.Now(),
				ErrorRate:         0.125,
			},
		}

		mockDB := &db.DB{}
	mockParser := parser.NewParser("/tmp")
	mockScanManager := scanner.NewScanManager(mockDB, mockParser)
	handler := NewHandler(mockService, mockScanManager)
		req := httptest.NewRequest("GET", "/api/groups/1/stats", nil)
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()

		handler.getGroupStatsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response ProjectGroupStatsResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.TotalProjects != 2 {
			t.Errorf("Expected 2 projects, got %d", response.TotalProjects)
		}

		if response.TotalSessions != 8 {
			t.Errorf("Expected 8 sessions, got %d", response.TotalSessions)
		}

		if response.ErrorRate != 0.125 {
			t.Errorf("Expected error rate 0.125, got %.3f", response.ErrorRate)
		}
	})

	t.Run("無効なグループIDでエラーを返す", func(t *testing.T) {
		mockService := &MockSessionService{}
		mockDB := &db.DB{}
	mockParser := parser.NewParser("/tmp")
	mockScanManager := scanner.NewScanManager(mockDB, mockParser)
	handler := NewHandler(mockService, mockScanManager)
		req := httptest.NewRequest("GET", "/api/groups/invalid/stats", nil)
		req.SetPathValue("id", "invalid")
		w := httptest.NewRecorder()

		handler.getGroupStatsHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("存在しないグループIDで404を返す", func(t *testing.T) {
		mockService := &MockSessionService{
			err: fmt.Errorf("group not found"),
		}
		mockDB := &db.DB{}
	mockParser := parser.NewParser("/tmp")
	mockScanManager := scanner.NewScanManager(mockDB, mockParser)
	handler := NewHandler(mockService, mockScanManager)
		req := httptest.NewRequest("GET", "/api/groups/999/stats", nil)
		req.SetPathValue("id", "999")
		w := httptest.NewRecorder()

		handler.getGroupStatsHandler(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})
}
