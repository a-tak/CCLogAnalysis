package api

import "time"

// SessionService defines the interface for session operations
type SessionService interface {
	ListProjects() ([]ProjectResponse, error)
	ListSessions(projectName string) ([]SessionSummary, error)
	GetSession(projectName, sessionID string) (*SessionDetailResponse, error)
	Analyze(projectNames []string) (*AnalyzeResponse, error)
	GetProjectStats(projectName string) (*ProjectStatsResponse, error)
	GetProjectTimeline(projectName, period string, limit int) (*TimeSeriesResponse, error)
	ListProjectGroups() ([]ProjectGroupResponse, error)
	GetProjectGroup(groupID int64) (*ProjectGroupDetailResponse, error)
	GetProjectGroupStats(groupID int64) (*ProjectGroupStatsResponse, error)
	GetProjectGroupTimeline(groupID int64, period string, limit int) (*TimeSeriesResponse, error)
	GetTotalStats() (*TotalStatsResponse, error)
	GetTotalTimeline(period string, limit int) (*TimeSeriesResponse, error)
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status string `json:"status"`
}

// ProjectResponse represents a project in the API response
type ProjectResponse struct {
	Name         string `json:"name"`
	DecodedPath  string `json:"decodedPath"`
	SessionCount int    `json:"sessionCount"`
}

// ProjectListResponse represents the list of projects
type ProjectListResponse struct {
	Projects []ProjectResponse `json:"projects"`
}

// SessionSummary represents a session in list view
type SessionSummary struct {
	ID               string    `json:"id"`
	ProjectName      string    `json:"projectName"`
	GitBranch        string    `json:"gitBranch"`
	StartTime        time.Time `json:"startTime"`
	EndTime          time.Time `json:"endTime"`
	TotalTokens      int       `json:"totalTokens"`
	ErrorCount       int       `json:"errorCount"`
	FirstUserMessage string    `json:"firstUserMessage"`
}

// SessionListResponse represents the list of sessions
type SessionListResponse struct {
	Sessions []SessionSummary `json:"sessions"`
}

// TokenSummaryResponse represents token usage in API response
type TokenSummaryResponse struct {
	InputTokens              int `json:"inputTokens"`
	OutputTokens             int `json:"outputTokens"`
	CacheCreationInputTokens int `json:"cacheCreationInputTokens"`
	CacheReadInputTokens     int `json:"cacheReadInputTokens"`
	TotalTokens              int `json:"totalTokens"`
}

// ModelUsageResponse represents per-model token usage
type ModelUsageResponse struct {
	Model  string               `json:"model"`
	Tokens TokenSummaryResponse `json:"tokens"`
}

// ToolCallResponse represents a tool call in API response
type ToolCallResponse struct {
	Timestamp time.Time   `json:"timestamp"`
	Name      string      `json:"name"`
	Input     interface{} `json:"input"`
	IsError   bool        `json:"isError"`
}

// MessageResponse represents a message in conversation history
type MessageResponse struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Model     string      `json:"model,omitempty"`
	Content   interface{} `json:"content"`
	IsError   bool        `json:"isError,omitempty"`
}

// SessionDetailResponse represents detailed session info
type SessionDetailResponse struct {
	ID          string               `json:"id"`
	ProjectName string               `json:"projectName"`
	ProjectPath string               `json:"projectPath"`
	GitBranch   string               `json:"gitBranch"`
	StartTime   time.Time            `json:"startTime"`
	EndTime     time.Time            `json:"endTime"`
	Duration    string               `json:"duration"`
	TotalTokens TokenSummaryResponse `json:"totalTokens"`
	ModelUsage  []ModelUsageResponse `json:"modelUsage"`
	ToolCalls   []ToolCallResponse   `json:"toolCalls"`
	Messages    []MessageResponse    `json:"messages"`
	ErrorCount  int                  `json:"errorCount"`
}

// AnalyzeRequest represents the analyze request body
type AnalyzeRequest struct {
	ProjectNames []string `json:"projectNames,omitempty"`
	Force        bool     `json:"force"`
}

// AnalyzeResponse represents the analyze response
type AnalyzeResponse struct {
	Status          string `json:"status"`
	SessionsFound   int    `json:"sessionsFound"`
	SessionsParsed  int    `json:"sessionsParsed"`
	ErrorCount      int    `json:"errorCount,omitempty"`
	Message         string `json:"message,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// ProjectStatsResponse represents project-level statistics
type ProjectStatsResponse struct {
	TotalSessions            int       `json:"totalSessions"`
	TotalInputTokens         int       `json:"totalInputTokens"`
	TotalOutputTokens        int       `json:"totalOutputTokens"`
	TotalCacheCreationTokens int       `json:"totalCacheCreationTokens"`
	TotalCacheReadTokens     int       `json:"totalCacheReadTokens"`
	TotalTokens              int       `json:"totalTokens"`
	AvgTokens                float64   `json:"avgTokens"`
	FirstSession             time.Time `json:"firstSession"`
	LastSession              time.Time `json:"lastSession"`
	ErrorRate                float64   `json:"errorRate"`
}

// BranchStatsResponse represents statistics per branch
type BranchStatsResponse struct {
	Branch                   string    `json:"branch"`
	SessionCount             int       `json:"sessionCount"`
	TotalInputTokens         int       `json:"totalInputTokens"`
	TotalOutputTokens        int       `json:"totalOutputTokens"`
	TotalCacheCreationTokens int       `json:"totalCacheCreationTokens"`
	TotalCacheReadTokens     int       `json:"totalCacheReadTokens"`
	TotalTokens              int       `json:"totalTokens"`
	LastActivity             time.Time `json:"lastActivity"`
}

// TimeSeriesDataPoint represents a single data point in time series
type TimeSeriesDataPoint struct {
	PeriodStart              time.Time `json:"periodStart"`
	PeriodEnd                time.Time `json:"periodEnd"`
	SessionCount             int       `json:"sessionCount"`
	TotalInputTokens         int       `json:"totalInputTokens"`
	TotalOutputTokens        int       `json:"totalOutputTokens"`
	TotalCacheCreationTokens int       `json:"totalCacheCreationTokens"`
	TotalCacheReadTokens     int       `json:"totalCacheReadTokens"`
	TotalTokens              int       `json:"totalTokens"`
}

// TimeSeriesResponse represents time-series statistics response
type TimeSeriesResponse struct {
	Period string                 `json:"period"`
	Data   []TimeSeriesDataPoint `json:"data"`
}

// ProjectGroupResponse represents a project group
type ProjectGroupResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	GitRoot   *string   `json:"gitRoot,omitempty"` // NULL可能
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ProjectGroupListResponse represents the list of project groups
type ProjectGroupListResponse struct {
	Groups []ProjectGroupResponse `json:"groups"`
}

// ProjectGroupDetailResponse represents detailed project group info with member projects
type ProjectGroupDetailResponse struct {
	ID        int64             `json:"id"`
	Name      string            `json:"name"`
	GitRoot   *string           `json:"gitRoot,omitempty"` // NULL可能
	CreatedAt time.Time         `json:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt"`
	Projects  []ProjectResponse `json:"projects"`
}

// ProjectGroupStatsResponse represents project group statistics
type ProjectGroupStatsResponse struct {
	TotalProjects            int       `json:"totalProjects"`
	TotalSessions            int       `json:"totalSessions"`
	TotalInputTokens         int       `json:"totalInputTokens"`
	TotalOutputTokens        int       `json:"totalOutputTokens"`
	TotalCacheCreationTokens int       `json:"totalCacheCreationTokens"`
	TotalCacheReadTokens     int       `json:"totalCacheReadTokens"`
	AvgTokens                float64   `json:"avgTokens"`
	FirstSession             time.Time `json:"firstSession"`
	LastSession              time.Time `json:"lastSession"`
	ErrorRate                float64   `json:"errorRate"`
}

// ScanStatusResponse represents the scan status
type ScanStatusResponse struct {
	Status            string  `json:"status"`
	ProjectsProcessed int     `json:"projectsProcessed"`
	SessionsFound     int     `json:"sessionsFound"`
	SessionsSynced    int     `json:"sessionsSynced"`
	SessionsSkipped   int     `json:"sessionsSkipped"`
	ErrorCount        int     `json:"errorCount"`
	StartedAt         string  `json:"startedAt"`
	CompletedAt       *string `json:"completedAt,omitempty"`
	LastError         string  `json:"lastError,omitempty"`
}

// TotalStatsResponse represents total statistics across all projects
type TotalStatsResponse struct {
	TotalGroups              int       `json:"totalGroups"`
	TotalProjects            int       `json:"totalProjects"`
	TotalSessions            int       `json:"totalSessions"`
	TotalInputTokens         int       `json:"totalInputTokens"`
	TotalOutputTokens        int       `json:"totalOutputTokens"`
	TotalCacheCreationTokens int       `json:"totalCacheCreationTokens"`
	TotalCacheReadTokens     int       `json:"totalCacheReadTokens"`
	TotalTokens              int       `json:"totalTokens"`
	AvgTokens                float64   `json:"avgTokens"`
	FirstSession             time.Time `json:"firstSession"`
	LastSession              time.Time `json:"lastSession"`
	ErrorRate                float64   `json:"errorRate"`
}
