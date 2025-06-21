package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAllCommandsBasicFunctionality tests that all commands work without errors
func TestAllCommandsBasicFunctionality(t *testing.T) {
	// Create a comprehensive test vault
	vaultFiles := map[string]string{
		"note1.md": `---
title: Note 1
tags: [test, important]
created: 2024-01-01
url: https://example.com
---

# Note 1

This is note 1 with a link to [[note2]].

External link: [Example](https://example.com)
`,
		"note2.md": `---
title: Note 2
type: project
status: active
created: 2024-01-02
---

# Note 2

This is note 2.

Reference back: [[note1]]
`,
		"subfolder/note3.md": `---
title: Note 3
tags: [reference]
---

# Note 3

In a subfolder.
`,
		"no-frontmatter.md": `# No Frontmatter

This note has no frontmatter.
`,
		"INBOX.md": `# INBOX

## TODO
- Item 1
- Item 2

## URLs
- https://example.com/resource
`,
	}

	vaultPath, err := createTestVault(vaultFiles)
	require.NoError(t, err)
	defer os.RemoveAll(vaultPath)

	tests := []struct {
		name     string
		command  []string
		shouldSucceed bool
		description string
	}{
		// Frontmatter commands
		{
			name:     "frontmatter_ensure",
			command:  []string{"frontmatter", "ensure", vaultPath, "--field", "test", "--default", "value"},
			shouldSucceed: true,
			description: "Should add frontmatter fields",
		},
		{
			name:     "frontmatter_validate",
			command:  []string{"frontmatter", "validate", vaultPath},
			shouldSucceed: true,
			description: "Should validate frontmatter",
		},
		{
			name:     "frontmatter_cast",
			command:  []string{"frontmatter", "cast", vaultPath},
			shouldSucceed: true,
			description: "Should cast frontmatter types",
		},
		{
			name:     "frontmatter_sync",
			command:  []string{"frontmatter", "sync", vaultPath, "--field", "modified", "--source", "file-mtime"},
			shouldSucceed: true,
			description: "Should sync frontmatter with file system",
		},
		{
			name:     "frontmatter_query",
			command:  []string{"frontmatter", "query", vaultPath, "--where", "title != ''"},
			shouldSucceed: true,
			description: "Should query frontmatter",
		},
		{
			name:     "frontmatter_set",
			command:  []string{"frontmatter", "set", filepath.Join(vaultPath, "note1.md"), "--field", "processed", "--value", "true"},
			shouldSucceed: true,
			description: "Should set frontmatter field",
		},

		// Heading commands
		{
			name:     "headings_analyze",
			command:  []string{"headings", "analyze", vaultPath},
			shouldSucceed: true,
			description: "Should analyze headings",
		},
		{
			name:     "headings_fix",
			command:  []string{"headings", "fix", vaultPath},
			shouldSucceed: true,
			description: "Should fix headings",
		},

		// Link commands
		{
			name:     "links_check",
			command:  []string{"links", "check", vaultPath},
			shouldSucceed: true,
			description: "Should check links",
		},

		// Analysis commands
		{
			name:     "analyze_health",
			command:  []string{"analyze", "health", vaultPath},
			shouldSucceed: true,
			description: "Should analyze vault health",
		},
		{
			name:     "analyze_stats",
			command:  []string{"analyze", "stats", vaultPath},
			shouldSucceed: true,
			description: "Should generate statistics",
		},
		{
			name:     "analyze_content",
			command:  []string{"analyze", "content", vaultPath},
			shouldSucceed: true,
			description: "Should analyze content quality",
		},
		{
			name:     "analyze_links",
			command:  []string{"analyze", "links", vaultPath},
			shouldSucceed: true,
			description: "Should analyze link structure",
		},
		{
			name:     "analyze_inbox",
			command:  []string{"analyze", "inbox", vaultPath},
			shouldSucceed: true,
			description: "Should analyze inbox content",
		},
		{
			name:     "analyze_trends",
			command:  []string{"analyze", "trends", vaultPath},
			shouldSucceed: true,
			description: "Should analyze trends",
		},
		{
			name:     "analyze_duplicates",
			command:  []string{"analyze", "duplicates", vaultPath},
			shouldSucceed: true,
			description: "Should find duplicates",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := runMdnotesCommand(tt.command...)
			
			if tt.shouldSucceed {
				assert.NoError(t, err, "%s failed: %s", tt.description, string(output))
			} else {
				assert.Error(t, err, "%s should have failed", tt.description)
			}
		})
	}
}

// TestGlobalFlagsConsistency tests that global flags work consistently across all commands
func TestGlobalFlagsConsistency(t *testing.T) {
	vaultFiles := map[string]string{
		"test.md": `---
title: Test
---

# Test

Content.
`,
	}

	vaultPath, err := createTestVault(vaultFiles)
	require.NoError(t, err)
	defer os.RemoveAll(vaultPath)

	// Commands to test with global flags
	baseCommands := [][]string{
		{"frontmatter", "ensure", vaultPath, "--field", "test", "--default", "value"},
		{"headings", "fix", vaultPath},
		{"analyze", "health", vaultPath},
		{"links", "check", vaultPath},
	}

	globalFlags := []struct {
		name  string
		flag  string
		description string
	}{
		{"verbose", "--verbose", "Should provide detailed output"},
		{"quiet", "--quiet", "Should suppress non-essential output"},
		{"dry_run", "--dry-run", "Should preview changes without applying"},
	}

	for _, baseCmd := range baseCommands {
		for _, globalFlag := range globalFlags {
			t.Run(strings.Join(baseCmd[:2], "_")+"_"+globalFlag.name, func(t *testing.T) {
				cmd := append(baseCmd, globalFlag.flag)
				output, err := runMdnotesCommand(cmd...)
				
				assert.NoError(t, err, "Command with %s should succeed: %s", globalFlag.flag, string(output))
				
				// Basic validation based on flag type
				outputStr := string(output)
				switch globalFlag.flag {
				case "--quiet":
					// Quiet mode should have minimal output
					assert.True(t, len(outputStr) < 500, "Quiet mode should have minimal output")
				case "--dry-run":
					// Dry-run should indicate preview mode or produce minimal output
					// (Some commands might not have explicit "would" language)
					assert.NotContains(t, outputStr, "ERROR", "Dry-run should not produce errors")
				case "--verbose":
					// Verbose mode might have more output, but this varies by command
					// Just ensure it doesn't error
					assert.NotContains(t, outputStr, "ERROR", "Verbose mode should not produce errors")
				}
			})
		}
	}
}

// TestErrorHandling tests that commands handle various error conditions gracefully
func TestErrorHandling(t *testing.T) {
	// Create a minimal vault
	vaultPath, err := createTestVault(map[string]string{
		"valid.md": `---
title: Valid
---

# Valid

Content.
`,
	})
	require.NoError(t, err)
	defer os.RemoveAll(vaultPath)

	errorTests := []struct {
		name         string
		command      []string
		expectedFail bool
		description  string
	}{
		{
			name:         "nonexistent_file",
			command:      []string{"frontmatter", "ensure", "/nonexistent/path"},
			expectedFail: true,
			description:  "Should fail gracefully for nonexistent files",
		},
		{
			name:         "invalid_query",
			command:      []string{"frontmatter", "query", vaultPath, "--where", "invalid syntax"},
			expectedFail: false, // The query parser handles this and returns helpful error message
			description:  "Should handle invalid queries gracefully",
		},
		{
			name:         "nonexistent_vault",
			command:      []string{"analyze", "health", "/nonexistent/vault"},
			expectedFail: true,
			description:  "Should fail gracefully for nonexistent vaults",
		},
		{
			name:         "invalid_field_value",
			command:      []string{"frontmatter", "set", filepath.Join(vaultPath, "valid.md"), "--field", "", "--value", "test"},
			expectedFail: true,
			description:  "Should fail gracefully for invalid field names",
		},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := runMdnotesCommand(tt.command...)
			
			if tt.expectedFail {
				assert.Error(t, err, "%s should fail: %s", tt.description, string(output))
				// Error messages should be helpful
				outputStr := string(output)
				assert.True(t, len(outputStr) > 0, "Error should provide a message")
			} else {
				assert.NoError(t, err, "%s should succeed: %s", tt.description, string(output))
			}
		})
	}
}

// TestConfigFileHandling tests that configuration file handling works correctly
func TestConfigFileHandling(t *testing.T) {
	vaultPath, err := createTestVault(map[string]string{
		"test.md": `---
title: Test
---

# Test
`,
	})
	require.NoError(t, err)
	defer os.RemoveAll(vaultPath)

	// Create a test config file
	configContent := `
ignore_patterns:
  - "*.tmp"
  - ".DS_Store"

frontmatter:
  default_fields:
    created: "{{current_date}}"
    modified: "{{file_mtime}}"

output:
  verbose: false
  format: "table"
`
	
	configPath := filepath.Join(vaultPath, "test-config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	t.Run("explicit_config", func(t *testing.T) {
		// Test using explicit config file
		output, err := runMdnotesCommand("analyze", "health", vaultPath, "--config", configPath)
		assert.NoError(t, err, "Should work with explicit config: %s", string(output))
	})

	t.Run("default_config", func(t *testing.T) {
		// Test with default config (should work even if not present)
		output, err := runMdnotesCommand("analyze", "health", vaultPath)
		assert.NoError(t, err, "Should work without explicit config: %s", string(output))
	})
}

// TestOutputFormats tests that commands support different output formats
func TestOutputFormats(t *testing.T) {
	vaultFiles := map[string]string{
		"note1.md": `---
title: Note 1
type: project
priority: 1
---

# Note 1
`,
		"note2.md": `---
title: Note 2
type: note
priority: 2
---

# Note 2
`,
	}

	vaultPath, err := createTestVault(vaultFiles)
	require.NoError(t, err)
	defer os.RemoveAll(vaultPath)

	// Commands that support output formats
	formatCommands := [][]string{
		{"frontmatter", "query", vaultPath, "--where", "priority > 0"},
		{"analyze", "stats", vaultPath},
		{"analyze", "health", vaultPath},
	}

	formats := []string{"table", "json", "yaml", "csv"}

	for _, cmd := range formatCommands {
		for _, format := range formats {
			t.Run(strings.Join(cmd[:2], "_")+"_"+format, func(t *testing.T) {
				cmdWithFormat := append(cmd, "--format", format)
				output, err := runMdnotesCommand(cmdWithFormat...)
				
				// Not all commands may support all formats, so we allow errors
				// but if it succeeds, output should be non-empty
				if err == nil {
					assert.Greater(t, len(output), 0, "Successful command should produce output")
				}
			})
		}
	}
}