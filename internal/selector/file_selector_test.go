package selector

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eoinhurrell/mdnotes/internal/vault"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a temporary test directory
func createTestDir(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "file-selector-test-*")
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

func TestFileSelector_AutoDetectSingleFile(t *testing.T) {
	tmpDir := createTestDir(t)
	
	// Create a test file
	content := `---
title: Test Note
tags: [test]
---

# Test Note
This is a test note.`
	
	testFile := createTestFile(t, tmpDir, "test.md", content)
	
	selector := NewFileSelector()
	result, err := selector.SelectFiles(testFile, AutoDetect)
	
	assert.NoError(t, err)
	assert.Len(t, result.Files, 1)
	assert.Equal(t, AutoDetect, result.Mode)
	assert.Contains(t, result.Source, "file:")
	assert.Equal(t, "Test Note", result.Files[0].Frontmatter["title"])
}

func TestFileSelector_AutoDetectDirectory(t *testing.T) {
	tmpDir := createTestDir(t)
	
	// Create multiple test files
	createTestFile(t, tmpDir, "file1.md", `---
title: File 1
---
# File 1`)
	
	createTestFile(t, tmpDir, "file2.md", `---
title: File 2
---
# File 2`)
	
	// Create a subdirectory with a file
	subDir := filepath.Join(tmpDir, "subdir")
	require.NoError(t, os.MkdirAll(subDir, 0755))
	createTestFile(t, subDir, "file3.md", `---
title: File 3
---
# File 3`)
	
	selector := NewFileSelector()
	result, err := selector.SelectFiles(tmpDir, AutoDetect)
	
	assert.NoError(t, err)
	assert.Len(t, result.Files, 3)
	assert.Equal(t, AutoDetect, result.Mode)
	assert.Contains(t, result.Source, "directory:")
}

func TestFileSelector_WithQuery(t *testing.T) {
	tmpDir := createTestDir(t)
	
	// Create test files with different tags
	createTestFile(t, tmpDir, "draft.md", `---
title: Draft Note
status: draft
---
# Draft`)
	
	createTestFile(t, tmpDir, "published.md", `---
title: Published Note
status: published
---
# Published`)
	
	createTestFile(t, tmpDir, "review.md", `---
title: Review Note
status: review
---
# Review`)
	
	selector := NewFileSelector().WithQuery("status = 'draft'")
	result, err := selector.SelectFiles(tmpDir, AutoDetect)
	
	assert.NoError(t, err)
	assert.Len(t, result.Files, 1)
	assert.Equal(t, "Draft Note", result.Files[0].Frontmatter["title"])
	assert.Contains(t, result.Source, "filtered by query")
}

func TestFileSelector_FromQuery(t *testing.T) {
	tmpDir := createTestDir(t)
	
	// Create test files
	createTestFile(t, tmpDir, "urgent.md", `---
title: Urgent Task
priority: 5
---
# Urgent`)
	
	createTestFile(t, tmpDir, "normal.md", `---
title: Normal Task
priority: 3
---
# Normal`)
	
	createTestFile(t, tmpDir, "low.md", `---
title: Low Priority
priority: 1
---
# Low`)
	
	selector := NewFileSelector().WithQuery("priority > 3")
	result, err := selector.SelectFiles(tmpDir, FilesFromQuery)
	
	assert.NoError(t, err)
	assert.Len(t, result.Files, 1)
	assert.Equal(t, FilesFromQuery, result.Mode)
	assert.Equal(t, "Urgent Task", result.Files[0].Frontmatter["title"])
	assert.Contains(t, result.Source, "query:")
}

func TestFileSelector_FromStdin(t *testing.T) {
	tmpDir := createTestDir(t)
	
	// Create test files
	file1 := createTestFile(t, tmpDir, "file1.md", `---
title: File 1
---
# File 1`)
	
	file2 := createTestFile(t, tmpDir, "file2.md", `---
title: File 2
---
# File 2`)
	
	// Mock stdin with file paths
	stdinContent := file1 + "\n" + file2 + "\n"
	reader := strings.NewReader(stdinContent)
	
	selector := NewFileSelector()
	result, err := selector.selectFromReader(reader, "test-stdin", FilesFromStdin)
	
	assert.NoError(t, err)
	assert.Len(t, result.Files, 2)
	assert.Contains(t, result.Source, "test-stdin")
}

func TestFileSelector_FromFile(t *testing.T) {
	tmpDir := createTestDir(t)
	
	// Create test markdown files
	file1 := createTestFile(t, tmpDir, "file1.md", `---
title: File 1
---
# File 1`)
	
	file2 := createTestFile(t, tmpDir, "file2.md", `---
title: File 2
---
# File 2`)
	
	// Create a file list
	listFile := createTestFile(t, tmpDir, "filelist.txt", file1+"\n"+file2+"\n# Comment\n\n")
	
	selector := NewFileSelector().WithSourceFile(listFile)
	result, err := selector.SelectFiles("", FilesFromFile)
	
	assert.NoError(t, err)
	assert.Len(t, result.Files, 2)
	assert.Equal(t, FilesFromFile, result.Mode)
	assert.Contains(t, result.Source, "file:")
}

func TestFileSelector_WithIgnorePatterns(t *testing.T) {
	tmpDir := createTestDir(t)
	
	// Create files that should be ignored
	createTestFile(t, tmpDir, "normal.md", `# Normal`)
	createTestFile(t, tmpDir, "temp.tmp", `# Temp`)
	
	// Create .obsidian directory with file
	obsidianDir := filepath.Join(tmpDir, ".obsidian")
	require.NoError(t, os.MkdirAll(obsidianDir, 0755))
	createTestFile(t, obsidianDir, "config.md", `# Config`)
	
	selector := NewFileSelector().WithIgnorePatterns([]string{".obsidian/*", "*.tmp"})
	result, err := selector.SelectFiles(tmpDir, AutoDetect)
	
	assert.NoError(t, err)
	assert.Len(t, result.Files, 1) // Only normal.md should be included
	assert.Equal(t, "normal.md", filepath.Base(result.Files[0].Path))
}

func TestFileSelector_ParseErrors(t *testing.T) {
	tmpDir := createTestDir(t)
	
	// Create a file with invalid YAML frontmatter
	createTestFile(t, tmpDir, "invalid.md", `---
title: "Invalid YAML
tags: [unclosed
---
# Invalid`)
	
	createTestFile(t, tmpDir, "valid.md", `---
title: Valid
---
# Valid`)
	
	selector := NewFileSelector()
	result, err := selector.SelectFiles(tmpDir, AutoDetect)
	
	assert.NoError(t, err)
	assert.Len(t, result.Files, 1) // Only valid file loaded successfully
	assert.Len(t, result.ParseErrors, 1) // Invalid file should have parse error
	assert.Contains(t, result.ParseErrors[0].Path, "invalid.md")
}

func TestFileSelector_NonMarkdownFile(t *testing.T) {
	tmpDir := createTestDir(t)
	
	// Create a non-markdown file
	txtFile := createTestFile(t, tmpDir, "test.txt", "This is not markdown")
	
	selector := NewFileSelector()
	_, err := selector.SelectFiles(txtFile, AutoDetect)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must have .md extension")
}

func TestFileSelector_NonExistentPath(t *testing.T) {
	selector := NewFileSelector()
	_, err := selector.SelectFiles("/nonexistent/path", AutoDetect)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path error")
}

func TestSelectionResult_GetSelectionSummary(t *testing.T) {
	result := &SelectionResult{
		Files:       make([]*vault.VaultFile, 5),
		ParseErrors: make([]vault.ParseError, 2),
		Mode:        AutoDetect,
		Source:      "directory: /test",
	}
	
	summary := result.GetSelectionSummary()
	assert.Contains(t, summary, "Selected 5 files")
	assert.Contains(t, summary, "directory: /test")
	assert.Contains(t, summary, "2 parse errors")
}