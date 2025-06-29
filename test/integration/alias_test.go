package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCommandAliases tests that all command aliases work correctly
func TestCommandAliases(t *testing.T) {
	// Create a temporary test vault
	tmpDir, err := os.MkdirTemp("", "mdnotes-alias-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.md")
	content := `---
title: Test Note
---

# Test Note

This is a test note.`

	err = os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	tests := []struct {
		name     string
		command  []string
		expected string
	}{
		{
			name:     "fm alias for frontmatter",
			command:  []string{"fm", "ensure", testFile, "--field", "created", "--default", "2024-01-01", "--dry-run"},
			expected: "Dry run completed",
		},
		{
			name:     "e alias for frontmatter ensure",
			command:  []string{"e", testFile, "--field", "modified", "--default", "2024-01-02", "--dry-run"},
			expected: "Dry run completed",
		},
		{
			name:     "a alias for analyze",
			command:  []string{"a", "health", tmpDir, "--quiet"},
			expected: "Score:",
		},
		{
			name:     "f alias for headings fix",
			command:  []string{"f", testFile, "--dry-run"},
			expected: "",
		},
		{
			name:     "c alias for links check",
			command:  []string{"c", tmpDir},
			expected: "completed",
		},
		{
			name:     "q alias for frontmatter query",
			command:  []string{"q", tmpDir, "--where", "title = 'Test Note'"},
			expected: "test.md",
		},
		{
			name:     "s alias for frontmatter set",
			command:  []string{"s", testFile, "--field", "status", "--value", "active", "--dry-run"},
			expected: "Dry run completed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run the command using the built binary
			output, err := runMdnotesCommand(tt.command...)

			// All commands should succeed
			assert.NoError(t, err, "Command should not return an error: %s", string(output))

			// Check for expected output if specified
			if tt.expected != "" {
				assert.Contains(t, string(output), tt.expected, "Output should contain expected text")
			}
		})
	}
}

// TestGroupAliases tests that group aliases work correctly
func TestGroupAliases(t *testing.T) {
	// Create a temporary test vault
	tmpDir, err := os.MkdirTemp("", "mdnotes-group-alias-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a test file with URL for linkding
	testFile := filepath.Join(tmpDir, "test.md")
	content := `---
title: Test Note
url: https://example.com
---

# Test Note

This is a test note with a URL.`

	err = os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	tests := []struct {
		name     string
		command  []string
		expected string
	}{
		{
			name:     "fm ensure group alias",
			command:  []string{"fm", "ensure", testFile, "--field", "created", "--default", "2024-01-01", "--dry-run"},
			expected: "Dry run completed",
		},
		{
			name:     "fm query group alias",
			command:  []string{"fm", "query", tmpDir, "--where", "title = 'Test Note'"},
			expected: "test.md",
		},
		{
			name:     "fm set group alias",
			command:  []string{"fm", "set", testFile, "--field", "status", "--value", "active", "--dry-run"},
			expected: "Dry run completed",
		},
		{
			name:     "a health group alias",
			command:  []string{"a", "health", tmpDir, "--quiet"},
			expected: "Score:",
		},
		{
			name:     "a stats group alias",
			command:  []string{"a", "stats", tmpDir},
			expected: "Files:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run the command using the built binary
			output, err := runMdnotesCommand(tt.command...)

			// All commands should succeed
			assert.NoError(t, err, "Command should not return an error: %s", string(output))

			// Check for expected output if specified
			if tt.expected != "" {
				assert.Contains(t, string(output), tt.expected, "Output should contain expected text")
			}
		})
	}
}

// TestAliasConsistency ensures that aliases produce the same output as full commands
func TestAliasConsistency(t *testing.T) {
	// Create a temporary test vault
	tmpDir, err := os.MkdirTemp("", "mdnotes-consistency-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.md")
	content := `---
title: Test Note
tags: [test]
---

# Test Note

This is a test note.`

	err = os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	tests := []struct {
		name     string
		fullCmd  []string
		aliasCmd []string
	}{
		{
			name:     "frontmatter ensure vs fm ensure",
			fullCmd:  []string{"frontmatter", "ensure", testFile, "--field", "created", "--default", "2024-01-01", "--dry-run"},
			aliasCmd: []string{"fm", "ensure", testFile, "--field", "created", "--default", "2024-01-01", "--dry-run"},
		},
		{
			name:     "frontmatter ensure vs e shortcut",
			fullCmd:  []string{"frontmatter", "ensure", testFile, "--field", "modified", "--default", "2024-01-02", "--dry-run"},
			aliasCmd: []string{"e", testFile, "--field", "modified", "--default", "2024-01-02", "--dry-run"},
		},
		{
			name:     "analyze health vs a health",
			fullCmd:  []string{"analyze", "health", tmpDir, "--quiet"},
			aliasCmd: []string{"a", "health", tmpDir, "--quiet"},
		},
		{
			name:     "frontmatter query vs q",
			fullCmd:  []string{"frontmatter", "query", tmpDir, "--where", "title = 'Test Note'"},
			aliasCmd: []string{"q", tmpDir, "--where", "title = 'Test Note'"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run both commands
			fullOutput, fullErr := runMdnotesCommand(tt.fullCmd...)
			aliasOutput, aliasErr := runMdnotesCommand(tt.aliasCmd...)

			// Both should succeed or fail the same way
			assert.Equal(t, fullErr != nil, aliasErr != nil, "Both commands should have same error status")

			// Outputs should be identical (ignoring timing differences)
			if fullErr == nil && aliasErr == nil {
				// For some commands, output might contain timestamps or other variable data
				// So we check that both contain the same key elements
				assert.Contains(t, string(aliasOutput), extractKeyContent(string(fullOutput)),
					"Alias output should contain same key content as full command")
			}
		})
	}
}

// extractKeyContent extracts the main content from command output, ignoring timestamps
func extractKeyContent(output string) string {
	// This is a simple extraction - in a real implementation,
	// you might want more sophisticated parsing
	if len(output) > 20 {
		return output[:20]
	}
	return output
}
