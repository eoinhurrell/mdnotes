package rename

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a temporary test vault
func createTestVault(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "mdnotes-rename-test-*")
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

func TestRenameCommand_Basic(t *testing.T) {
	tmpDir := createTestVault(t)
	
	// Create a source file
	sourceContent := `---
title: Original Note
tags: [test]
---

# Original Note

This is the original content.`
	
	sourceFile := createTestFile(t, tmpDir, "original.md", sourceContent)
	
	// Create a file that references the original
	referencingContent := `# Referencing Note

This note links to [[original]] and also [link text](original.md).

Also an embed: ![[original]]`
	
	createTestFile(t, tmpDir, "referencing.md", referencingContent)
	
	cmd := NewRenameCommand()
	
	// Rename the file
	args := []string{
		sourceFile,
		"renamed.md",
		"--vault", tmpDir,
	}
	
	err := runCommand(t, cmd, args)
	assert.NoError(t, err)
	
	// Verify the file was renamed
	renamedFile := filepath.Join(tmpDir, "renamed.md")
	_, err = os.Stat(renamedFile)
	assert.NoError(t, err, "Renamed file should exist")
	
	// Verify original file no longer exists
	_, err = os.Stat(sourceFile)
	assert.True(t, os.IsNotExist(err), "Original file should not exist")
	
	// Verify references were updated
	referencingPath := filepath.Join(tmpDir, "referencing.md")
	updatedContent, err := os.ReadFile(referencingPath)
	require.NoError(t, err)
	
	contentStr := string(updatedContent)
	assert.Contains(t, contentStr, "[[renamed]]")
	assert.Contains(t, contentStr, "[link text](renamed.md)")
	assert.Contains(t, contentStr, "![[renamed]]")
	
	// Should not contain old references
	assert.NotContains(t, contentStr, "[[original]]")
	assert.NotContains(t, contentStr, "(original.md)")
}

func TestRenameCommand_WithTemplate(t *testing.T) {
	tmpDir := createTestVault(t)
	
	// Create a source file with frontmatter for template
	sourceContent := `---
title: Test Note
created: 2023-01-15
---

# Test Note

Content for template test.`
	
	sourceFile := createTestFile(t, tmpDir, "messy filename.md", sourceContent)
	
	cmd := NewRenameCommand()
	
	// Rename using template (no target name provided)
	args := []string{
		sourceFile,
		"--vault", tmpDir,
		"--template", "{{created|date:20060102}}-{{filename|slug}}.md",
	}
	
	err := runCommand(t, cmd, args)
	assert.NoError(t, err)
	
	// Verify the file was renamed according to template
	expectedName := "20230115-messy-filename.md"
	renamedFile := filepath.Join(tmpDir, expectedName)
	_, err = os.Stat(renamedFile)
	assert.NoError(t, err, "File should be renamed according to template")
}

func TestRenameCommand_DirectoryChange(t *testing.T) {
	tmpDir := createTestVault(t)
	
	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	err := os.MkdirAll(subDir, 0755)
	require.NoError(t, err)
	
	// Create source file
	sourceContent := `# Source Note

This note will be moved to a subdirectory.`
	
	sourceFile := createTestFile(t, tmpDir, "source.md", sourceContent)
	
	// Create a referencing file
	referencingContent := `# Main Note

References: [[source]] and [Source](source.md)`
	
	createTestFile(t, tmpDir, "main.md", referencingContent)
	
	cmd := NewRenameCommand()
	
	// Move file to subdirectory
	targetPath := filepath.Join(subDir, "moved.md")
	args := []string{
		sourceFile,
		targetPath,
		"--vault", tmpDir,
	}
	
	err = runCommand(t, cmd, args)
	assert.NoError(t, err)
	
	// Verify file exists in new location
	_, err = os.Stat(targetPath)
	assert.NoError(t, err, "File should exist in new location")
	
	// Verify references were updated with new path
	mainFile := filepath.Join(tmpDir, "main.md")
	updatedContent, err := os.ReadFile(mainFile)
	require.NoError(t, err)
	
	contentStr := string(updatedContent)
	assert.Contains(t, contentStr, "[[subdir/moved]]")
	assert.Contains(t, contentStr, "[Source](subdir/moved.md)")
}

func TestRenameCommand_FilenameOnly(t *testing.T) {
	tmpDir := createTestVault(t)
	
	// Create source file
	sourceContent := `# Old Name

This is a filename-only rename test.`
	
	sourceFile := createTestFile(t, tmpDir, "old-name.md", sourceContent)
	
	// Create referencing files with different link formats
	ref1Content := `# Reference 1

Links: [[old-name]] and [[old-name|Custom Alias]]`
	
	ref2Content := `# Reference 2

Links: [Old Name](old-name.md) and ![[old-name]]`
	
	createTestFile(t, tmpDir, "ref1.md", ref1Content)
	createTestFile(t, tmpDir, "ref2.md", ref2Content)
	
	cmd := NewRenameCommand()
	
	// Rename file (same directory)
	args := []string{
		sourceFile,
		"new-name.md",
		"--vault", tmpDir,
	}
	
	err := runCommand(t, cmd, args)
	assert.NoError(t, err)
	
	// Verify all references were updated correctly
	ref1Path := filepath.Join(tmpDir, "ref1.md")
	ref1Updated, err := os.ReadFile(ref1Path)
	require.NoError(t, err)
	
	ref1Str := string(ref1Updated)
	assert.Contains(t, ref1Str, "[[new-name]]")
	assert.Contains(t, ref1Str, "[[new-name|Custom Alias]]")
	
	ref2Path := filepath.Join(tmpDir, "ref2.md")
	ref2Updated, err := os.ReadFile(ref2Path)
	require.NoError(t, err)
	
	ref2Str := string(ref2Updated)
	assert.Contains(t, ref2Str, "[Old Name](new-name.md)")
	assert.Contains(t, ref2Str, "![[new-name]]")
}

func TestRenameCommand_NoReferences(t *testing.T) {
	tmpDir := createTestVault(t)
	
	// Create source file
	sourceContent := `# Orphan Note

This note has no references to it.`
	
	sourceFile := createTestFile(t, tmpDir, "orphan.md", sourceContent)
	
	// Create another file with no references to orphan
	otherContent := `# Other Note

This note doesn't reference the orphan file.`
	
	createTestFile(t, tmpDir, "other.md", otherContent)
	
	cmd := NewRenameCommand()
	
	args := []string{
		sourceFile,
		"renamed-orphan.md",
		"--vault", tmpDir,
	}
	
	err := runCommand(t, cmd, args)
	assert.NoError(t, err)
	
	// Verify the file was renamed successfully
	renamedFile := filepath.Join(tmpDir, "renamed-orphan.md")
	_, err = os.Stat(renamedFile)
	assert.NoError(t, err)
	
	// Verify original file no longer exists
	_, err = os.Stat(sourceFile)
	assert.True(t, os.IsNotExist(err))
}

func TestRenameCommand_InvalidSource(t *testing.T) {
	tmpDir := createTestVault(t)
	
	cmd := NewRenameCommand()
	
	// Try to rename non-existent file
	args := []string{
		filepath.Join(tmpDir, "nonexistent.md"),
		"new-name.md",
		"--vault", tmpDir,
	}
	
	err := runCommand(t, cmd, args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestRenameCommand_TargetExists(t *testing.T) {
	tmpDir := createTestVault(t)
	
	// Create source file
	sourceFile := createTestFile(t, tmpDir, "source.md", "# Source")
	
	// Create target file that already exists
	createTestFile(t, tmpDir, "target.md", "# Target")
	
	cmd := NewRenameCommand()
	
	// Try to rename to existing file
	args := []string{
		sourceFile,
		"target.md",
		"--vault", tmpDir,
	}
	
	err := runCommand(t, cmd, args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestRenameCommand_CaseInsensitiveRename(t *testing.T) {
	tmpDir := createTestVault(t)
	
	// Create a source file with capital letters
	sourceContent := `---
title: Birdman
tags: [movie]
---

# Birdman

This is a movie note.`
	
	sourceFile := createTestFile(t, tmpDir, "Birdman.md", sourceContent)
	
	// Create a referencing file
	referencingContent := `# Movie List

Movies I've watched:
- [[Birdman]] - Great film
- [Birdman Review](Birdman.md)`
	
	createTestFile(t, tmpDir, "movies.md", referencingContent)
	
	cmd := NewRenameCommand()
	
	// Rename to lowercase version (case-only change)
	args := []string{
		sourceFile,
		"birdman.md",
		"--vault", tmpDir,
	}
	
	err := runCommand(t, cmd, args)
	assert.NoError(t, err, "Case-insensitive rename should succeed")
	
	// Verify the file was renamed to lowercase
	renamedFile := filepath.Join(tmpDir, "birdman.md")
	_, err = os.Stat(renamedFile)
	assert.NoError(t, err, "Renamed file should exist with lowercase name")
	
	// On case-insensitive filesystems, check if the file now has the new name
	// by reading the directory and checking the actual filename
	entries, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	
	foundOriginalCase := false
	foundNewCase := false
	for _, entry := range entries {
		if entry.Name() == "Birdman.md" {
			foundOriginalCase = true
		}
		if entry.Name() == "birdman.md" {
			foundNewCase = true
		}
	}
	
	// On case-insensitive filesystems, we should only find the new case
	assert.False(t, foundOriginalCase, "Original case filename should not be found in directory listing")
	assert.True(t, foundNewCase, "New case filename should be found in directory listing")
	
	// Verify references were updated
	moviesPath := filepath.Join(tmpDir, "movies.md")
	updatedContent, err := os.ReadFile(moviesPath)
	require.NoError(t, err)
	
	contentStr := string(updatedContent)
	assert.Contains(t, contentStr, "[[birdman]]", "Wiki link should be updated to lowercase")
	assert.Contains(t, contentStr, "[Birdman Review](birdman.md)", "Markdown link should be updated to lowercase")
	
	// Should not contain old references
	assert.NotContains(t, contentStr, "[[Birdman]]", "Old wiki link should be replaced")
	assert.NotContains(t, contentStr, "(Birdman.md)", "Old markdown link should be replaced")
}

func TestRenameCommand_PreservesContent(t *testing.T) {
	tmpDir := createTestVault(t)
	
	// Create source file with complex content
	originalContent := `---
title: Complex Note
tags: [test, complex]
created: 2023-01-01
---

# Complex Note

This note has:
- **Bold text**
- *Italic text*
- ` + "`code`" + `
- [External link](https://example.com)

## Code Block

` + "```go" + `
func main() {
    fmt.Println("Hello, World!")
}
` + "```" + `

## Table

| Column 1 | Column 2 |
|----------|----------|
| Value 1  | Value 2  |`
	
	sourceFile := createTestFile(t, tmpDir, "complex.md", originalContent)
	
	cmd := NewRenameCommand()
	
	args := []string{
		sourceFile,
		"renamed-complex.md",
		"--vault", tmpDir,
	}
	
	err := runCommand(t, cmd, args)
	assert.NoError(t, err)
	
	// Verify content is preserved exactly
	renamedFile := filepath.Join(tmpDir, "renamed-complex.md")
	renamedContent, err := os.ReadFile(renamedFile)
	require.NoError(t, err)
	
	assert.Equal(t, originalContent, string(renamedContent))
}

// Benchmark tests
func TestRenameCommand_DatestringPreservation(t *testing.T) {
	tmpDir := createTestVault(t)
	
	// Create a source file that already has a datestring prefix
	sourceContent := `---
title: The Queen
created: 2025-06-01
---

# The Queen

This file already has a datestring prefix.`
	
	sourceFile := createTestFile(t, tmpDir, "20250601223002-The Queen.md", sourceContent)
	
	cmd := NewRenameCommand()
	
	// Rename using default template (should preserve existing datestring)
	args := []string{
		sourceFile,
		"--vault", tmpDir,
	}
	
	err := runCommand(t, cmd, args)
	assert.NoError(t, err)
	
	// The file should be renamed to preserve the existing datestring and use underscores
	expectedName := "20250601223002-the_queen.md"
	renamedFile := filepath.Join(tmpDir, expectedName)
	_, err = os.Stat(renamedFile)
	assert.NoError(t, err, "File should be renamed with preserved datestring and underscores")
	
	// Verify original file no longer exists
	_, err = os.Stat(sourceFile)
	assert.True(t, os.IsNotExist(err), "Original file should not exist")
	
	// Should NOT create a double datestring like "20250601223002-20250601223002-the-queen.md"
	wrongFile := filepath.Join(tmpDir, "20250601223002-20250601223002-the-queen.md")
	_, err = os.Stat(wrongFile)
	assert.True(t, os.IsNotExist(err), "Should not create double datestring")
}

func TestRenameCommand_DatestringWithoutPrefix(t *testing.T) {
	tmpDir := createTestVault(t)
	
	// Create a source file WITHOUT datestring prefix
	sourceContent := `---
title: Regular Note
created: 2025-06-01
---

# Regular Note

This file doesn't have a datestring prefix.`
	
	sourceFile := createTestFile(t, tmpDir, "Regular Note.md", sourceContent)
	
	cmd := NewRenameCommand()
	
	// Rename using default template (should create new datestring)
	args := []string{
		sourceFile,
		"--vault", tmpDir,
	}
	
	err := runCommand(t, cmd, args)
	assert.NoError(t, err)
	
	// Find the renamed file (should have a new datestring)
	files, err := filepath.Glob(filepath.Join(tmpDir, "*-regular_note.md"))
	require.NoError(t, err)
	require.Len(t, files, 1, "Should find exactly one renamed file")
	
	renamedFile := files[0]
	renamedName := filepath.Base(renamedFile)
	
	// Should be in format YYYYMMDDHHMMSS-regular_note.md with underscores
	assert.Regexp(t, `^\d{14}-regular_note\.md$`, renamedName)
	assert.Contains(t, renamedName, "regular_note") // underscores not hyphens
}

func BenchmarkRenameCommand_FilenameOnly(b *testing.B) {
	tmpDir := createTestVault(&testing.T{})
	defer os.RemoveAll(tmpDir)
	
	// Create many files with references
	for i := 0; i < 50; i++ {
		content := `# Test Note

This references [[target]] file.`
		createTestFile(&testing.T{}, tmpDir, "ref"+string(rune(i))+".md", content)
	}
	
	cmd := NewRenameCommand()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create target file for each benchmark iteration
		targetFile := createTestFile(&testing.T{}, tmpDir, "target.md", "# Target")
		
		args := []string{
			targetFile,
			"renamed-target.md",
			"--vault", tmpDir,
		}
		
		runCommand(&testing.T{}, cmd, args)
		
		// Clean up for next iteration
		os.Rename(filepath.Join(tmpDir, "renamed-target.md"), targetFile)
	}
}