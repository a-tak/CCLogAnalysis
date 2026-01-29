package api

import "testing"

func TestExtractDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Standard Unix path",
			input:    "/home/user/projects/my-project",
			expected: "my-project",
		},
		{
			name:     "Standard Windows path",
			input:    "C:/Users/username/projects/my-project",
			expected: "my-project",
		},
		{
			name:     "Path with trailing slash",
			input:    "/home/user/projects/my-project/",
			expected: "my-project",
		},
		{
			name:     "Single folder",
			input:    "my-project",
			expected: "my-project",
		},
		{
			name:     "Empty path",
			input:    "",
			expected: "",
		},
		{
			name:     "Path with spaces",
			input:    "/home/user/my project name",
			expected: "my project name",
		},
		{
			name:     "Worktree path",
			input:    "/home/user/.claude/projects/CCLogAnalysis.worktrees/feature-branch",
			expected: "feature-branch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractDisplayName(tt.input)
			if result != tt.expected {
				t.Errorf("extractDisplayName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
