package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// runMdnotesCommand runs the mdnotes binary with the given arguments
func runMdnotesCommand(args ...string) ([]byte, error) {
	// Get the binary path relative to the test directory
	binaryPath := filepath.Join("..", "..", "mdnotes")
	
	// Check if binary exists, if not try to build it
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		// Try to build the binary
		buildCmd := exec.Command("go", "build", "-o", "mdnotes", "./cmd")
		buildCmd.Dir = filepath.Join("..", "..")
		if buildErr := buildCmd.Run(); buildErr != nil {
			return nil, buildErr
		}
	}
	
	cmd := exec.Command(binaryPath, args...)
	return cmd.CombinedOutput()
}

// createTestVault creates a temporary test vault with sample files
func createTestVault(files map[string]string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "mdnotes-test-vault-*")
	if err != nil {
		return "", err
	}
	
	for filename, content := range files {
		filePath := filepath.Join(tmpDir, filename)
		
		// Create directory if needed
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", err
		}
		
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return "", err
		}
	}
	
	return tmpDir, nil
}

// createLargeTestVault creates a larger test vault for performance testing
func createLargeTestVault(numFiles int) (string, error) {
	tmpDir, err := os.MkdirTemp("", "mdnotes-large-test-vault-*")
	if err != nil {
		return "", err
	}
	
	// Create subdirectories
	dirs := []string{"notes", "projects", "archive", "templates", "inbox"}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			return "", err
		}
	}
	
	// Create files distributed across directories
	for i := 0; i < numFiles; i++ {
		dir := dirs[i%len(dirs)]
		filename := filepath.Join(tmpDir, dir, "note"+string(rune(i%26+'a'))+string(rune((i/26)%26+'a'))+".md")
		
		content := generateTestContent(i)
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			return "", err
		}
	}
	
	return tmpDir, nil
}

// generateTestContent generates realistic test content for performance testing
func generateTestContent(index int) string {
	templates := []string{
		`---
title: Note %d
tags: [test, note%d]
created: 2024-01-%02d
priority: %d
---

# Note %d

This is test note number %d. It contains some sample content to make it realistic.

## Overview

This note demonstrates:
- Basic markdown structure
- Frontmatter with various field types
- Links to other notes: [[note%d]]
- External links: [Example](https://example.com)

## Content

Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.

### Subsection

More content here with different structures.

## References

- Related note: [[note%d]]
- Another link: [Link text](note%d.md)
`,
		`---
title: Project %d
type: project
status: active
created: 2024-01-%02d
tags: [project, work]
---

# Project %d

This is a project note with different structure.

## Goals

1. Goal one
2. Goal two  
3. Goal three

## Tasks

- [ ] Task 1
- [ ] Task 2
- [x] Completed task

## Notes

See also: [[note%d]] for related information.

Project reference: [[project%d]]
`,
	}
	
	template := templates[index%len(templates)]
	day := (index % 28) + 1
	priority := (index % 3) + 1
	relatedIndex := (index + 1) % 100
	
	return fmt.Sprintf(template, index, index, day, priority, index, index, relatedIndex, relatedIndex+1, relatedIndex+2, index, relatedIndex, index+1)
}

