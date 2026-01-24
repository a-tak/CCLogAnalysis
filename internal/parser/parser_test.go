package parser

import (
	"path/filepath"
	"testing"
	"time"
)

func TestParseFile_BasicSession(t *testing.T) {
	testFile := filepath.Join("testdata", "sample_session.jsonl")
	parser := NewParser(".")

	session, err := parser.ParseFile(testFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Test session ID
	if session.ID != "test-session-001" {
		t.Errorf("Expected session ID 'test-session-001', got '%s'", session.ID)
	}

	// Test git branch
	if session.GitBranch != "main" {
		t.Errorf("Expected git branch 'main', got '%s'", session.GitBranch)
	}

	// Test project path
	if session.ProjectPath != "/Users/user/projects/my-project" {
		t.Errorf("Expected project path '/Users/user/projects/my-project', got '%s'", session.ProjectPath)
	}

	// Test entry count
	if len(session.Entries) != 6 {
		t.Errorf("Expected 6 entries, got %d", len(session.Entries))
	}
}

func TestParseFile_TokenAggregation(t *testing.T) {
	testFile := filepath.Join("testdata", "sample_session.jsonl")
	parser := NewParser(".")

	session, err := parser.ParseFile(testFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Test total input tokens (100 + 200 + 50 = 350)
	expectedInputTokens := 350
	if session.TotalTokens.InputTokens != expectedInputTokens {
		t.Errorf("Expected total input tokens %d, got %d", expectedInputTokens, session.TotalTokens.InputTokens)
	}

	// Test total output tokens (50 + 30 + 20 = 100)
	expectedOutputTokens := 100
	if session.TotalTokens.OutputTokens != expectedOutputTokens {
		t.Errorf("Expected total output tokens %d, got %d", expectedOutputTokens, session.TotalTokens.OutputTokens)
	}

	// Test cache creation tokens (500 + 0 + 0 = 500)
	expectedCacheCreation := 500
	if session.TotalTokens.CacheCreationInputTokens != expectedCacheCreation {
		t.Errorf("Expected cache creation tokens %d, got %d", expectedCacheCreation, session.TotalTokens.CacheCreationInputTokens)
	}

	// Test cache read tokens (0 + 500 + 100 = 600)
	expectedCacheRead := 600
	if session.TotalTokens.CacheReadInputTokens != expectedCacheRead {
		t.Errorf("Expected cache read tokens %d, got %d", expectedCacheRead, session.TotalTokens.CacheReadInputTokens)
	}
}

func TestParseFile_StringContentFormat(t *testing.T) {
	testFile := filepath.Join("testdata", "string_content_session.jsonl")
	parser := NewParser(".")

	session, err := parser.ParseFile(testFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Test that entries were parsed
	if len(session.Entries) == 0 {
		t.Fatal("Expected at least 1 entry, got 0")
	}

	// Find user message
	var userEntry *LogEntry
	for i := range session.Entries {
		if session.Entries[i].Type == "user" && session.Entries[i].Message != nil {
			userEntry = &session.Entries[i]
			break
		}
	}

	if userEntry == nil {
		t.Fatal("Expected to find user message")
	}

	// Test that string content was converted to array
	if len(userEntry.Message.Content) != 1 {
		t.Errorf("Expected 1 content block, got %d", len(userEntry.Message.Content))
	}

	if userEntry.Message.Content[0].Type != "text" {
		t.Errorf("Expected content type 'text', got '%s'", userEntry.Message.Content[0].Type)
	}

	if userEntry.Message.Content[0].Text != "Warmup" {
		t.Errorf("Expected content text 'Warmup', got '%s'", userEntry.Message.Content[0].Text)
	}
}

func TestParseFile_ModelUsage(t *testing.T) {
	testFile := filepath.Join("testdata", "sample_session.jsonl")
	parser := NewParser(".")

	session, err := parser.ParseFile(testFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Test model count
	if len(session.ModelUsage) != 2 {
		t.Errorf("Expected 2 models, got %d", len(session.ModelUsage))
	}

	// Test Sonnet usage (2 messages: 100+200=300 input, 50+30=80 output)
	sonnetUsage, ok := session.ModelUsage["claude-sonnet-4-20250514"]
	if !ok {
		t.Fatal("Expected sonnet model usage to exist")
	}
	if sonnetUsage.InputTokens != 300 {
		t.Errorf("Expected Sonnet input tokens 300, got %d", sonnetUsage.InputTokens)
	}
	if sonnetUsage.OutputTokens != 80 {
		t.Errorf("Expected Sonnet output tokens 80, got %d", sonnetUsage.OutputTokens)
	}

	// Test Haiku usage (1 message: 50 input, 20 output)
	haikuUsage, ok := session.ModelUsage["claude-haiku-4-5-20251001"]
	if !ok {
		t.Fatal("Expected haiku model usage to exist")
	}
	if haikuUsage.InputTokens != 50 {
		t.Errorf("Expected Haiku input tokens 50, got %d", haikuUsage.InputTokens)
	}
	if haikuUsage.OutputTokens != 20 {
		t.Errorf("Expected Haiku output tokens 20, got %d", haikuUsage.OutputTokens)
	}
}

func TestParseFile_ToolCalls(t *testing.T) {
	testFile := filepath.Join("testdata", "sample_session.jsonl")
	parser := NewParser(".")

	session, err := parser.ParseFile(testFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Test tool call count
	if len(session.ToolCalls) != 1 {
		t.Errorf("Expected 1 tool call, got %d", len(session.ToolCalls))
	}

	// Test tool call details
	if len(session.ToolCalls) > 0 {
		toolCall := session.ToolCalls[0]
		if toolCall.Name != "Bash" {
			t.Errorf("Expected tool name 'Bash', got '%s'", toolCall.Name)
		}
	}
}

func TestParseFile_ErrorCount(t *testing.T) {
	testFile := filepath.Join("testdata", "session_with_error.jsonl")
	parser := NewParser(".")

	session, err := parser.ParseFile(testFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Test error count
	if session.ErrorCount != 1 {
		t.Errorf("Expected 1 error, got %d", session.ErrorCount)
	}
}

func TestParseFile_Timestamps(t *testing.T) {
	testFile := filepath.Join("testdata", "sample_session.jsonl")
	parser := NewParser(".")

	session, err := parser.ParseFile(testFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Test start time
	expectedStart, _ := time.Parse(time.RFC3339, "2026-01-11T03:24:10.137Z")
	if !session.StartTime.Equal(expectedStart) {
		t.Errorf("Expected start time %v, got %v", expectedStart, session.StartTime)
	}

	// Test end time
	expectedEnd, _ := time.Parse(time.RFC3339, "2026-01-11T03:24:40.000Z")
	if !session.EndTime.Equal(expectedEnd) {
		t.Errorf("Expected end time %v, got %v", expectedEnd, session.EndTime)
	}
}

func TestDecodeProjectPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "C--Users--username--projects--my-project",
			expected: "C/Users/username/projects/my-project",
		},
		{
			input:    "Users--username--my-repo",
			expected: "Users/username/my-repo",
		},
	}

	for _, tt := range tests {
		result := DecodeProjectPath(tt.input)
		if result != tt.expected {
			t.Errorf("DecodeProjectPath(%s) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

func TestParseFile_NonExistentFile(t *testing.T) {
	parser := NewParser(".")

	_, err := parser.ParseFile("nonexistent.jsonl")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}
