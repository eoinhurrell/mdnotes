package export

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a temporary test vault
func createTestVault(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "mdnotes-export-test-*")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})
	return tmpDir
}

// Helper function to create a test file with content
func createTestFile(t *testing.T, dir, filename, content string) string {
	filePath := filepath.Join(dir, filename)
	
	// Create directory if it contains path separators
	if strings.Contains(filename, string(filepath.Separator)) {
		fileDir := filepath.Dir(filePath)
		err := os.MkdirAll(fileDir, 0755)
		require.NoError(t, err)
	}
	
	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)
	return filePath
}

// Helper function to create a temporary output directory
func createOutputDir(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "mdnotes-export-output-*")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})
	return tmpDir
}

// runMdnotesCommand runs the mdnotes binary with the given arguments
func runMdnotesCommand(args ...string) (string, error) {
	// Get the binary path relative to the test directory
	binaryPath := filepath.Join("..", "..", "mdnotes")
	
	// Check if binary exists, if not try to build it
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		// Try to build the binary
		buildCmd := exec.Command("go", "build", "-o", "mdnotes", "./cmd")
		buildCmd.Dir = filepath.Join("..", "..")
		if buildErr := buildCmd.Run(); buildErr != nil {
			return "", buildErr
		}
	}
	
	cmd := exec.Command(binaryPath, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// Test helper to run command with arguments
func runExportCommand(t *testing.T, args []string) (string, error) {
	// We need to capture stdout since the display functions use fmt.Printf
	// For testing, we'll use the mdnotes binary directly
	return runMdnotesCommand(append([]string{"export"}, args...)...)
}

func TestExportCommand_Basic(t *testing.T) {
	vaultDir := createTestVault(t)
	outputDir := createOutputDir(t)
	
	// Create test files
	createTestFile(t, vaultDir, "note1.md", `---
title: Note 1
tags: [test]
---

# Note 1

This is the first note.`)
	
	createTestFile(t, vaultDir, "note2.md", `---
title: Note 2
type: project
---

# Note 2

This is a project note.`)
	
	createTestFile(t, vaultDir, "subfolder/note3.md", `---
title: Note 3
---

# Note 3

This note is in a subfolder.`)
	
	// Run export command
	args := []string{outputDir, vaultDir}
	output, err := runExportCommand(t, args)
	
	assert.NoError(t, err)
	assert.Contains(t, output, "Export completed successfully")
	assert.Contains(t, output, "Exported 3 files")
	
	// Verify files were copied
	assert.FileExists(t, filepath.Join(outputDir, "note1.md"))
	assert.FileExists(t, filepath.Join(outputDir, "note2.md"))
	assert.FileExists(t, filepath.Join(outputDir, "subfolder", "note3.md"))
	
	// Verify content is preserved
	content, err := os.ReadFile(filepath.Join(outputDir, "note1.md"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "This is the first note.")
}

func TestExportCommand_WithQuery(t *testing.T) {
	vaultDir := createTestVault(t)
	outputDir := createOutputDir(t)
	
	// Create test files with different tags
	createTestFile(t, vaultDir, "published.md", `---
title: Published Note
tags: [published, blog]
---

# Published Note

This note should be exported.`)
	
	createTestFile(t, vaultDir, "draft.md", `---
title: Draft Note
tags: [draft]
---

# Draft Note

This note should not be exported.`)
	
	createTestFile(t, vaultDir, "project.md", `---
title: Project Note
type: project
tags: [published, work]
---

# Project Note

This project note should be exported.`)
	
	// Run export command with query
	args := []string{outputDir, vaultDir, "--query", "tags contains 'published'"}
	output, err := runExportCommand(t, args)
	
	assert.NoError(t, err)
	assert.Contains(t, output, "Export completed successfully")
	assert.Contains(t, output, "Exported 2 files")
	
	// Verify only matching files were copied
	assert.FileExists(t, filepath.Join(outputDir, "published.md"))
	assert.FileExists(t, filepath.Join(outputDir, "project.md"))
	assert.NoFileExists(t, filepath.Join(outputDir, "draft.md"))
}

func TestExportCommand_DryRun(t *testing.T) {
	vaultDir := createTestVault(t)
	outputDir := createOutputDir(t)
	
	// Create test files
	createTestFile(t, vaultDir, "note1.md", `---
title: Note 1
---

# Note 1

Content here.`)
	
	createTestFile(t, vaultDir, "note2.md", `---
title: Note 2
---

# Note 2

More content.`)
	
	// Run export command with dry-run
	args := []string{outputDir, vaultDir, "--dry-run"}
	output, err := runExportCommand(t, args)
	
	assert.NoError(t, err)
	assert.Contains(t, output, "Export Summary (Dry Run)")
	assert.Contains(t, output, "Would export 2 files")
	assert.Contains(t, output, "Files scanned:  2")
	assert.Contains(t, output, "Files selected: 2")
	
	// Verify no files were actually copied
	assert.NoFileExists(t, filepath.Join(outputDir, "note1.md"))
	assert.NoFileExists(t, filepath.Join(outputDir, "note2.md"))
}

func TestExportCommand_DryRunWithQuery(t *testing.T) {
	vaultDir := createTestVault(t)
	outputDir := createOutputDir(t)
	
	// Create test files
	createTestFile(t, vaultDir, "active.md", `---
title: Active Note
status: active
---

# Active Note

This is active.`)
	
	createTestFile(t, vaultDir, "inactive.md", `---
title: Inactive Note
status: inactive
---

# Inactive Note

This is inactive.`)
	
	// Run export command with dry-run and query
	args := []string{outputDir, vaultDir, "--dry-run", "--query", "status = 'active'"}
	output, err := runExportCommand(t, args)
	
	assert.NoError(t, err)
	assert.Contains(t, output, "Export Summary (Dry Run)")
	assert.Contains(t, output, "Would export 1 files")
	assert.Contains(t, output, "Files scanned:  2")
	assert.Contains(t, output, "Files selected: 1")
	
	// Verify no files were copied
	assert.NoFileExists(t, filepath.Join(outputDir, "active.md"))
	assert.NoFileExists(t, filepath.Join(outputDir, "inactive.md"))
}

func TestExportCommand_VerboseOutput(t *testing.T) {
	vaultDir := createTestVault(t)
	outputDir := createOutputDir(t)
	
	// Create test files
	createTestFile(t, vaultDir, "test.md", `---
title: Test Note
---

# Test

Content.`)
	
	// Run export command with verbose
	args := []string{outputDir, vaultDir, "--verbose"}
	output, err := runExportCommand(t, args)
	
	assert.NoError(t, err)
	assert.Contains(t, output, "Exporting from:")
	assert.Contains(t, output, "Exporting to:")
	assert.Contains(t, output, "Processing details:")
	assert.Contains(t, output, "Exported files:")
	assert.Contains(t, output, "test.md")
}

func TestExportCommand_EmptyVault(t *testing.T) {
	vaultDir := createTestVault(t)
	outputDir := createOutputDir(t)
	
	// Don't create any files
	
	// Run export command
	args := []string{outputDir, vaultDir}
	output, err := runExportCommand(t, args)
	
	assert.NoError(t, err)
	assert.Contains(t, output, "Exported 0 files")
}

func TestExportCommand_NoMatchingFiles(t *testing.T) {
	vaultDir := createTestVault(t)
	outputDir := createOutputDir(t)
	
	// Create test files that won't match the query
	createTestFile(t, vaultDir, "note.md", `---
title: Note
tags: [personal]
---

# Note

Content.`)
	
	// Run export command with query that matches nothing
	args := []string{outputDir, vaultDir, "--query", "tags contains 'nonexistent'", "--dry-run"}
	output, err := runExportCommand(t, args)
	
	assert.NoError(t, err)
	assert.Contains(t, output, "No files match the criteria")
	assert.Contains(t, output, "Files scanned:  1")
	assert.Contains(t, output, "Files selected: 0")
}

func TestExportCommand_PreservesDirectoryStructure(t *testing.T) {
	vaultDir := createTestVault(t)
	outputDir := createOutputDir(t)
	
	// Create nested directory structure
	createTestFile(t, vaultDir, "level1/note1.md", `# Note 1`)
	createTestFile(t, vaultDir, "level1/level2/note2.md", `# Note 2`)
	createTestFile(t, vaultDir, "level1/level2/level3/note3.md", `# Note 3`)
	
	// Run export command
	args := []string{outputDir, vaultDir}
	output, err := runExportCommand(t, args)
	
	assert.NoError(t, err)
	assert.Contains(t, output, "Exported 3 files")
	
	// Verify directory structure is preserved
	assert.FileExists(t, filepath.Join(outputDir, "level1", "note1.md"))
	assert.FileExists(t, filepath.Join(outputDir, "level1", "level2", "note2.md"))
	assert.FileExists(t, filepath.Join(outputDir, "level1", "level2", "level3", "note3.md"))
}

func TestExportCommand_IgnorePatterns(t *testing.T) {
	vaultDir := createTestVault(t)
	outputDir := createOutputDir(t)
	
	// Create files that should be ignored
	createTestFile(t, vaultDir, "note.md", `# Note`)
	createTestFile(t, vaultDir, "temp.tmp", `temporary file`)
	createTestFile(t, vaultDir, ".obsidian/config.json", `{"setting": "value"}`)
	
	// Run export command
	args := []string{outputDir, vaultDir}
	output, err := runExportCommand(t, args)
	
	assert.NoError(t, err)
	assert.Contains(t, output, "Exported 1 files") // Only note.md should be exported
	
	// Verify ignored files were not copied
	assert.FileExists(t, filepath.Join(outputDir, "note.md"))
	assert.NoFileExists(t, filepath.Join(outputDir, "temp.tmp"))
	assert.NoFileExists(t, filepath.Join(outputDir, ".obsidian", "config.json"))
}

func TestExportCommand_InvalidPaths(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		errorMsg string
	}{
		{
			name:     "nonexistent vault",
			args:     []string{"/tmp/output", "/nonexistent/vault"},
			errorMsg: "vault path does not exist",
		},
		{
			name:     "output directory not empty",
			args:     []string{".", "/tmp"},
			errorMsg: "output directory is not empty",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := runExportCommand(t, tt.args)
			assert.Error(t, err)
			assert.Contains(t, output, tt.errorMsg)
		})
	}
}

func TestExportCommand_OutputDirectoryExists(t *testing.T) {
	vaultDir := createTestVault(t)
	outputDir := createOutputDir(t)
	
	// Create a file in the output directory
	createTestFile(t, outputDir, "existing.txt", "existing content")
	
	// Create test vault file
	createTestFile(t, vaultDir, "note.md", `# Note`)
	
	// Run export command - should fail because output dir is not empty
	args := []string{outputDir, vaultDir}
	output, err := runExportCommand(t, args)
	
	assert.Error(t, err)
	assert.Contains(t, output, "output directory is not empty")
}

func TestExportCommand_ComplexQuery(t *testing.T) {
	vaultDir := createTestVault(t)
	outputDir := createOutputDir(t)
	
	// Create test files with various frontmatter
	createTestFile(t, vaultDir, "published_blog.md", `---
title: Published Blog
type: blog
status: published
priority: 1
---

# Published Blog`)
	
	createTestFile(t, vaultDir, "draft_blog.md", `---
title: Draft Blog
type: blog
status: draft
priority: 2
---

# Draft Blog`)
	
	createTestFile(t, vaultDir, "published_note.md", `---
title: Published Note
type: note
status: published
priority: 3
---

# Published Note`)
	
	// Run export with complex query
	args := []string{outputDir, vaultDir, "--query", "type = 'blog' AND status = 'published'"}
	output, err := runExportCommand(t, args)
	
	assert.NoError(t, err)
	assert.Contains(t, output, "Exported 1 files")
	
	// Verify only the published blog was exported
	assert.FileExists(t, filepath.Join(outputDir, "published_blog.md"))
	assert.NoFileExists(t, filepath.Join(outputDir, "draft_blog.md"))
	assert.NoFileExists(t, filepath.Join(outputDir, "published_note.md"))
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 bytes"},
		{500, "500 bytes"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1536 * 1024 * 1024, "1.5 GB"},
	}
	
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatSize(tt.bytes)
			assert.Equal(t, tt.expected, result)
		})
	}
}