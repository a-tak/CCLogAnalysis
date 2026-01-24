package api

import "time"

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
	ID          string    `json:"id"`
	ProjectName string    `json:"projectName"`
	GitBranch   string    `json:"gitBranch"`
	StartTime   time.Time `json:"startTime"`
	EndTime     time.Time `json:"endTime"`
	TotalTokens int       `json:"totalTokens"`
	ErrorCount  int       `json:"errorCount"`
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
