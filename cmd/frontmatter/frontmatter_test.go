package frontmatter

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a temporary test vault
func createTestVault(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "mdnotes-test-*")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})
	return tmpDir
}

// Helper function to create a test file with content
func createTestFile(t *testing.T, dir, filename, content string) string {
	filePath := filepath.Join(dir, filename)
	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)
	return filePath
}

// Test helper to run command with arguments
func runCommand(t *testing.T, cmd *cobra.Command, args []string) error {
	cmd.SetArgs(args)
	return cmd.Execute()
}

func TestEnsureCommand_Basic(t *testing.T) {
	tmpDir := createTestVault(t)

	// Create a test file without frontmatter
	content := `# Test Note

This is a test note without frontmatter.`

	testFile := createTestFile(t, tmpDir, "test.md", content)

	// Create ensure command
	cmd := NewEnsureCommand()

	// Test adding a simple field
	args := []string{
		"--field", "tags",
		"--default", "[]",
		tmpDir,
	}

	err := runCommand(t, cmd, args)
	assert.NoError(t, err)

	// Verify the file was updated
	updatedContent, err := os.ReadFile(testFile)
	require.NoError(t, err)

	contentStr := string(updatedContent)
	assert.Contains(t, contentStr, "---")
	assert.Contains(t, contentStr, "tags: []")
}

func TestEnsureCommand_WithExistingFrontmatter(t *testing.T) {
	tmpDir := createTestVault(t)

	// Create a test file with existing frontmatter
	content := `---
title: Existing Title
created: 2023-01-01
---

# Test Note

This note already has frontmatter.`

	testFile := createTestFile(t, tmpDir, "existing.md", content)

	cmd := NewEnsureCommand()

	// Add a new field without overriding existing ones
	args := []string{
		"--field", "tags",
		"--default", "[]",
		"--field", "modified",
		"--default", "{{current_date}}",
		tmpDir,
	}

	err := runCommand(t, cmd, args)
	assert.NoError(t, err)

	// Verify the file was updated correctly
	updatedContent, err := os.ReadFile(testFile)
	require.NoError(t, err)

	contentStr := string(updatedContent)
	assert.Contains(t, contentStr, "title: Existing Title") // Original field preserved
	assert.Contains(t, contentStr, "created: 2023-01-01")   // Original field preserved
	assert.Contains(t, contentStr, "tags: []")              // New field added
	assert.Contains(t, contentStr, "modified:")             // New field added
}

func TestEnsureCommand_TemplateVariables(t *testing.T) {
	tmpDir := createTestVault(t)

	content := `# Template Test

Testing template variables.`

	testFile := createTestFile(t, tmpDir, "template-test.md", content)

	cmd := NewEnsureCommand()

	args := []string{
		"--field", "filename",
		"--default", "{{filename}}",
		"--field", "slug",
		"--default", "{{filename|slug}}",
		"--field", "created",
		"--default", "{{current_date}}",
		tmpDir,
	}

	err := runCommand(t, cmd, args)
	assert.NoError(t, err)

	// Verify template variables were processed
	updatedContent, err := os.ReadFile(testFile)
	require.NoError(t, err)

	contentStr := string(updatedContent)
	assert.Contains(t, contentStr, "filename: template-test")
	assert.Contains(t, contentStr, "slug: template-test")
	assert.Contains(t, contentStr, "created: \""+time.Now().Format("2006-01-02")+"\"")
}

// TODO: Add dry-run test when we can properly set up parent command structure
// func TestEnsureCommand_DryRun(t *testing.T) { ... }

func TestEnsureCommand_MultipleFields(t *testing.T) {
	tmpDir := createTestVault(t)

	content := `# Multiple Fields Test`
	testFile := createTestFile(t, tmpDir, "multi.md", content)

	cmd := NewEnsureCommand()

	args := []string{
		"--field", "tags",
		"--default", "[]",
		"--field", "priority",
		"--default", "3",
		"--field", "status",
		"--default", "draft",
		"--field", "created",
		"--default", "{{current_date}}",
		tmpDir,
	}

	err := runCommand(t, cmd, args)
	assert.NoError(t, err)

	// Verify all fields were added
	updatedContent, err := os.ReadFile(testFile)
	require.NoError(t, err)

	contentStr := string(updatedContent)
	assert.Contains(t, contentStr, "tags: []")
	assert.Contains(t, contentStr, "priority: \"3\"")
	assert.Contains(t, contentStr, "status: draft")
	assert.Contains(t, contentStr, "created:")
}

func TestEnsureCommand_NullDefault(t *testing.T) {
	tmpDir := createTestVault(t)

	content := `# Null Test`
	testFile := createTestFile(t, tmpDir, "null.md", content)

	cmd := NewEnsureCommand()

	args := []string{
		"--field", "optional_field",
		"--default", "null",
		tmpDir,
	}

	err := runCommand(t, cmd, args)
	assert.NoError(t, err)

	// Verify null field was added
	updatedContent, err := os.ReadFile(testFile)
	require.NoError(t, err)

	contentStr := string(updatedContent)
	assert.Contains(t, contentStr, "optional_field: null")
}

func TestEnsureCommand_InvalidArgs(t *testing.T) {
	cmd := NewEnsureCommand()

	// Test mismatched field and default counts
	args := []string{
		"--field", "tags",
		"--field", "status",
		"--default", "[]", // Only one default for two fields
		"/tmp/test",
	}

	err := runCommand(t, cmd, args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "number of fields")
}

func TestEnsureCommand_NonexistentDirectory(t *testing.T) {
	cmd := NewEnsureCommand()

	args := []string{
		"--field", "tags",
		"--default", "[]",
		"/nonexistent/directory",
	}

	err := runCommand(t, cmd, args)
	assert.Error(t, err)
}

func TestSetCommand_Basic(t *testing.T) {
	tmpDir := createTestVault(t)

	content := `---
title: Original Title
status: draft
---

# Test Note`

	testFile := createTestFile(t, tmpDir, "set-test.md", content)

	cmd := NewSetCommand()

	args := []string{
		"--field", "status",
		"--value", "published",
		"--field", "modified",
		"--value", "{{current_date}}",
		tmpDir,
	}

	err := runCommand(t, cmd, args)
	assert.NoError(t, err)

	// Verify the fields were set
	updatedContent, err := os.ReadFile(testFile)
	require.NoError(t, err)

	contentStr := string(updatedContent)
	assert.Contains(t, contentStr, "status: published")
	// Template variables are not expanded by the set command, it sets literal values
	assert.Contains(t, contentStr, "modified: '{{current_date}}'")
}

func TestCheckCommand_Basic(t *testing.T) {
	tmpDir := createTestVault(t)

	// Create file with valid frontmatter
	validContent := `---
title: Valid Note
tags: [test, valid]
priority: 5
published: true
created: 2023-01-01
---

# Valid Note`

	createTestFile(t, tmpDir, "valid.md", validContent)

	// Create file with invalid frontmatter
	invalidContent := `---
title: Invalid Note
tags: "should be array"
priority: "not a number"
---

# Invalid Note`

	createTestFile(t, tmpDir, "invalid.md", invalidContent)

	cmd := NewCheckCommand()

	args := []string{
		"--required", "title",
		"--required", "tags",
		"--type", "tags:array",
		"--type", "priority:number",
		"--type", "published:boolean",
		tmpDir,
	}

	err := runCommand(t, cmd, args)
	// Should return error because invalid.md has validation issues
	assert.Error(t, err)
}

func TestQueryCommand_Basic(t *testing.T) {
	tmpDir := createTestVault(t)

	// Create test files with different frontmatter
	file1 := `---
title: Draft Article
status: draft
priority: 5
tags: [work, article]
---

# Draft Article`

	file2 := `---
title: Published Post
status: published
priority: 3
tags: [blog, published]
---

# Published Post`

	createTestFile(t, tmpDir, "draft.md", file1)
	createTestFile(t, tmpDir, "published.md", file2)

	cmd := NewQueryCommand()

	// Test simple where query
	args := []string{
		"--where", "status = 'draft'",
		tmpDir,
	}

	err := runCommand(t, cmd, args)
	assert.NoError(t, err)
}

func TestCastCommand_Basic(t *testing.T) {
	tmpDir := createTestVault(t)

	content := `---
title: Cast Test
created: "2023-01-01"
priority: "5"
published: "true"
tags: "tag1,tag2,tag3"
---

# Cast Test`

	testFile := createTestFile(t, tmpDir, "cast.md", content)

	cmd := NewCastCommand()

	args := []string{
		"--field", "created",
		"--type", "created:date",
		"--field", "priority",
		"--type", "priority:number",
		"--field", "published",
		"--type", "published:boolean",
		"--field", "tags",
		"--type", "tags:array",
		tmpDir,
	}

	err := runCommand(t, cmd, args)
	assert.NoError(t, err)

	// Verify types were cast correctly
	updatedContent, err := os.ReadFile(testFile)
	require.NoError(t, err)

	contentStr := string(updatedContent)
	assert.Contains(t, contentStr, "created: 2023-01-01")      // Date without quotes
	assert.Contains(t, contentStr, "priority: 5")              // Number without quotes
	assert.Contains(t, contentStr, "published: true")          // Boolean without quotes
	assert.Contains(t, contentStr, "- tag1") // YAML array format
}

func TestSyncCommand_Basic(t *testing.T) {
	tmpDir := createTestVault(t)

	content := `---
title: Sync Test
---

# Sync Test`

	testFile := createTestFile(t, tmpDir, "sync-test.md", content)

	// Set file modification time to a known value
	pastTime := time.Now().Add(-24 * time.Hour)
	err := os.Chtimes(testFile, pastTime, pastTime)
	require.NoError(t, err)

	cmd := NewSyncCommand()

	args := []string{
		"--field", "modified",
		"--source", "file-mtime",
		tmpDir,
	}

	err = runCommand(t, cmd, args)
	assert.NoError(t, err)

	// Verify the modification time was synced
	updatedContent, err := os.ReadFile(testFile)
	require.NoError(t, err)

	contentStr := string(updatedContent)
	assert.Contains(t, contentStr, "modified:")
}

// Benchmark tests
func BenchmarkEnsureCommand(b *testing.B) {
	tmpDir := createTestVault(&testing.T{})
	defer os.RemoveAll(tmpDir)

	// Create multiple test files
	for i := 0; i < 100; i++ {
		content := `# Test Note ` + string(rune(i)) + `

This is a test note for benchmarking.`
		createTestFile(&testing.T{}, tmpDir, "test"+string(rune(i))+".md", content)
	}

	cmd := NewEnsureCommand()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		args := []string{
			"--field", "tags",
			"--default", "[]",
			tmpDir,
		}
		runCommand(&testing.T{}, cmd, args)
	}
}
