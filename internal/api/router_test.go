package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/a-tak/ccloganalysis/internal/db"
	"github.com/a-tak/ccloganalysis/internal/parser"
	"github.com/a-tak/ccloganalysis/internal/scanner"
)

// MockSessionService is a mock implementation of SessionService for testing
type MockSessionService struct {
	projects           []ProjectResponse
	sessions           []SessionSummary
	session            *SessionDetailResponse
	analyze            *AnalyzeResponse
	stats              *ProjectStatsResponse
	timeline           *TimeSeriesResponse
	ProjectGroups        []ProjectGroupResponse
	ProjectGroupDetail   *ProjectGroupDetailResponse
	ProjectGroupStats    *ProjectGroupStatsResponse
	ProjectGroupTimeline *TimeSeriesResponse
	TotalStats           *TotalStatsResponse
	TotalTimeline        *TimeSeriesResponse
	DailyStats           *DailyStatsResponse
	ShouldError          bool
	err                  error
}

func (m *MockSessionService) ListProjects() ([]ProjectResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.projects, nil
}

func (m *MockSessionService) ListSessions(projectName string) ([]SessionSummary, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.sessions, nil
}

func (m *MockSessionService) GetSession(projectName, sessionID string) (*SessionDetailResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.session, nil
}

func (m *MockSessionService) Analyze(projectNames []string) (*AnalyzeResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.analyze, nil
}

func (m *MockSessionService) GetProjectStats(projectName string) (*ProjectStatsResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.stats, nil
}

func (m *MockSessionService) GetProjectTimeline(projectName, period string, limit int) (*TimeSeriesResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.timeline, nil
}

func (m *MockSessionService) ListProjectGroups() ([]ProjectGroupResponse, error) {
	if m.ShouldError || m.err != nil {
		return nil, m.err
	}
	return m.ProjectGroups, nil
}

func (m *MockSessionService) GetProjectGroup(groupID int64) (*ProjectGroupDetailResponse, error) {
	if m.ShouldError || m.err != nil {
		return nil, m.err
	}
	return m.ProjectGroupDetail, nil
}

func (m *MockSessionService) GetProjectGroupStats(groupID int64) (*ProjectGroupStatsResponse, error) {
	if m.ShouldError || m.err != nil {
		return nil, m.err
	}
	return m.ProjectGroupStats, nil
}

func (m *MockSessionService) GetProjectGroupTimeline(groupID int64, period string, limit int) (*TimeSeriesResponse, error) {
	if m.ShouldError || m.err != nil {
		return nil, m.err
	}
	return m.ProjectGroupTimeline, nil
}

func (m *MockSessionService) GetTotalStats() (*TotalStatsResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.TotalStats, nil
}

func (m *MockSessionService) GetTotalTimeline(period string, limit int) (*TimeSeriesResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.TotalTimeline, nil
}

func (m *MockSessionService) GetDailyStats(date string) (*DailyStatsResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.DailyStats, nil
}

func TestHealthHandler(t *testing.T) {
	mockDB := &db.DB{}
	mockParser := parser.NewParser("/tmp")
	mockScanManager := scanner.NewScanManager(mockDB, mockParser)
	handler := NewHandler(nil, mockScanManager)
	router := handler.Routes()

	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", resp.Status)
	}
}

func TestListProjectsHandler(t *testing.T) {
	mockService := &MockSessionService{
		projects: []ProjectResponse{
			{Name: "project-1", DecodedPath: "{project-path}/project-1", SessionCount: 5},
			{Name: "project-2", DecodedPath: "{project-path}/project-2", SessionCount: 3},
		},
	}

	mockDB := &db.DB{}
	mockParser := parser.NewParser("/tmp")
	mockScanManager := scanner.NewScanManager(mockDB, mockParser)
	handler := NewHandler(mockService, mockScanManager)
	router := handler.Routes()

	req := httptest.NewRequest("GET", "/api/projects", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp ProjectListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(resp.Projects) != 2 {
		t.Errorf("Expected 2 projects, got %d", len(resp.Projects))
	}

	if resp.Projects[0].Name != "project-1" {
		t.Errorf("Expected project name 'project-1', got '%s'", resp.Projects[0].Name)
	}
}

func TestListSessionsHandler(t *testing.T) {
	startTime := time.Date(2026, 1, 11, 3, 24, 10, 0, time.UTC)
	endTime := time.Date(2026, 1, 11, 3, 30, 0, 0, time.UTC)

	mockService := &MockSessionService{
		sessions: []SessionSummary{
			{
				ID:          "session-001",
				ProjectName: "project-1",
				GitBranch:   "main",
				StartTime:   startTime,
				EndTime:     endTime,
				TotalTokens: 500,
				ErrorCount:  0,
			},
		},
	}

	mockDB := &db.DB{}
	mockParser := parser.NewParser("/tmp")
	mockScanManager := scanner.NewScanManager(mockDB, mockParser)
	handler := NewHandler(mockService, mockScanManager)
	router := handler.Routes()

	req := httptest.NewRequest("GET", "/api/sessions", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp SessionListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(resp.Sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(resp.Sessions))
	}

	if resp.Sessions[0].ID != "session-001" {
		t.Errorf("Expected session ID 'session-001', got '%s'", resp.Sessions[0].ID)
	}
}

func TestGetSessionHandler(t *testing.T) {
	startTime := time.Date(2026, 1, 11, 3, 24, 10, 0, time.UTC)
	endTime := time.Date(2026, 1, 11, 3, 30, 0, 0, time.UTC)

	mockService := &MockSessionService{
		session: &SessionDetailResponse{
			ID:          "session-001",
			ProjectName: "project-1",
			ProjectPath: "/Users/user/projects/my-project",
			GitBranch:   "main",
			StartTime:   startTime,
			EndTime:     endTime,
			Duration:    "5m 50s",
			TotalTokens: TokenSummaryResponse{
				InputTokens:  300,
				OutputTokens: 200,
				TotalTokens:  500,
			},
			ModelUsage: []ModelUsageResponse{
				{
					Model: "claude-sonnet-4-20250514",
					Tokens: TokenSummaryResponse{
						InputTokens:  200,
						OutputTokens: 150,
						TotalTokens:  350,
					},
				},
			},
			ToolCalls:  []ToolCallResponse{},
			Messages:   []MessageResponse{},
			ErrorCount: 0,
		},
	}

	mockDB := &db.DB{}
	mockParser := parser.NewParser("/tmp")
	mockScanManager := scanner.NewScanManager(mockDB, mockParser)
	handler := NewHandler(mockService, mockScanManager)
	router := handler.Routes()

	req := httptest.NewRequest("GET", "/api/sessions/project-1/session-001", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp SessionDetailResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.ID != "session-001" {
		t.Errorf("Expected session ID 'session-001', got '%s'", resp.ID)
	}

	if resp.TotalTokens.TotalTokens != 500 {
		t.Errorf("Expected total tokens 500, got %d", resp.TotalTokens.TotalTokens)
	}
}

func TestAnalyzeHandler(t *testing.T) {
	mockService := &MockSessionService{
		analyze: &AnalyzeResponse{
			Status:         "completed",
			SessionsFound:  10,
			SessionsParsed: 10,
			ErrorCount:     0,
		},
	}

	mockDB := &db.DB{}
	mockParser := parser.NewParser("/tmp")
	mockScanManager := scanner.NewScanManager(mockDB, mockParser)
	handler := NewHandler(mockService, mockScanManager)
	router := handler.Routes()

	req := httptest.NewRequest("POST", "/api/analyze", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp AnalyzeResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", resp.Status)
	}

	if resp.SessionsParsed != 10 {
		t.Errorf("Expected 10 sessions parsed, got %d", resp.SessionsParsed)
	}
}
