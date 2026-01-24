package parser

import "time"

// LogEntry represents a single line in the JSONL log file
type LogEntry struct {
	Type       string    `json:"type"`
	Timestamp  time.Time `json:"timestamp"`
	SessionID  string    `json:"sessionId"`
	UUID       string    `json:"uuid"`
	ParentUUID *string   `json:"parentUuid,omitempty"`
	Cwd        string    `json:"cwd"`
	Version    string    `json:"version"`
	GitBranch  string    `json:"gitBranch"`
	Message    *Message  `json:"message,omitempty"`
	RequestID  string    `json:"requestId,omitempty"`
}

// Message represents the message content in assistant/user entries
type Message struct {
	Model   string    `json:"model,omitempty"`
	ID      string    `json:"id,omitempty"`
	Role    string    `json:"role"`
	Content []Content `json:"content"`
	Usage   *Usage    `json:"usage,omitempty"`
}

// Content represents a content block (text or tool_use)
type Content struct {
	Type   string      `json:"type"`
	Text   string      `json:"text,omitempty"`
	ID     string      `json:"id,omitempty"`
	Name   string      `json:"name,omitempty"`
	Input  interface{} `json:"input,omitempty"`
	Caller *Caller     `json:"caller,omitempty"`

	// For tool_result
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   string `json:"content,omitempty"`
	IsError   bool   `json:"is_error,omitempty"`
}

// Caller represents the caller info for tool_use
type Caller struct {
	Type string `json:"type"`
}

// Usage represents token usage information
type Usage struct {
	InputTokens              int          `json:"input_tokens"`
	OutputTokens             int          `json:"output_tokens"`
	CacheCreationInputTokens int          `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int          `json:"cache_read_input_tokens"`
	CacheCreation            *CacheDetail `json:"cache_creation,omitempty"`
	ServiceTier              string       `json:"service_tier"`
}

// CacheDetail represents cache creation details
type CacheDetail struct {
	Ephemeral5mInputTokens int `json:"ephemeral_5m_input_tokens"`
	Ephemeral1hInputTokens int `json:"ephemeral_1h_input_tokens"`
}

// ToolUseResult represents the result of a tool use
type ToolUseResult struct {
	Stdout      string `json:"stdout,omitempty"`
	Stderr      string `json:"stderr,omitempty"`
	Interrupted bool   `json:"interrupted"`
	IsImage     bool   `json:"isImage"`
}

// Session represents a parsed session with aggregated data
type Session struct {
	ID           string
	ProjectPath  string
	GitBranch    string
	StartTime    time.Time
	EndTime      time.Time
	Entries      []LogEntry
	TotalTokens  TokenSummary
	ModelUsage   map[string]TokenSummary
	ToolCalls    []ToolCall
	ErrorCount   int
}

// TokenSummary holds aggregated token counts
type TokenSummary struct {
	InputTokens              int
	OutputTokens             int
	CacheCreationInputTokens int
	CacheReadInputTokens     int
}

// ToolCall represents a single tool invocation
type ToolCall struct {
	Timestamp time.Time
	Name      string
	Input     interface{}
	IsError   bool
	Result    string
}
