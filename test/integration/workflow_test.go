package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompleteWorkflow tests a complete real-world workflow
func TestCompleteWorkflow(t *testing.T) {
	// Create a test vault with realistic content
	vaultFiles := map[string]string{
		"inbox/quick-note.md": `---
url: https://example.com/article
---

# Quick Note

This is a quick capture that needs processing.

## TODO
- Process this note
- Add proper tags
`,
		"projects/project-alpha.md": `---
title: Project Alpha
type: project
status: planning
---

# Project Alpha

A new project that needs organization.

## Goals
- Goal 1
- Goal 2

## References
- [[quick-note]] - related resource
`,
		"notes/reference-note.md": `---
title: Reference Note
tags: [reference]
created: 2024-01-01
---

# Reference Note

This is a well-structured reference note.

Links to project: [[project-alpha]]
`,
		"archive/old-note.md": `# Old Note

This note has no frontmatter and needs updating.

Reference: [[reference-note]]
`,
	}

	vaultPath, err := createTestVault(vaultFiles)
	require.NoError(t, err)
	defer os.RemoveAll(vaultPath)

	t.Run("Step1_AnalyzeVault", func(t *testing.T) {
		// First, analyze vault health to understand current state
		output, err := runMdnotesCommand("analyze", "health", vaultPath)
		assert.NoError(t, err)
		assert.Contains(t, string(output), "Score:")

		// Should identify missing frontmatter
		assert.Contains(t, string(output), "frontmatter")
	})

	t.Run("Step2_EnsureFrontmatter", func(t *testing.T) {
		// Ensure all files have basic frontmatter
		output, err := runMdnotesCommand("frontmatter", "ensure", vaultPath,
			"--field", "created", "--default", "{{current_date}}",
			"--field", "modified", "--default", "{{file_mtime}}")
		assert.NoError(t, err)

		// Should have processed files
		assert.Contains(t, string(output), "Processed:")
	})

	t.Run("Step3_FixHeadings", func(t *testing.T) {
		// Fix heading structure
		output, err := runMdnotesCommand("headings", "fix", vaultPath, "--ensure-h1-title")
		assert.NoError(t, err)

		// Should complete without errors
		assert.NotContains(t, string(output), "Error")
	})

	t.Run("Step4_CheckLinks", func(t *testing.T) {
		// Check for broken links
		output, err := runMdnotesCommand("links", "check", vaultPath)
		assert.NoError(t, err)

		// Should complete the check
		assert.Contains(t, string(output), "complete")
	})

	t.Run("Step5_QueryContent", func(t *testing.T) {
		// Query for project files
		output, err := runMdnotesCommand("frontmatter", "query", vaultPath,
			"--where", "type = 'project'")
		assert.NoError(t, err)

		// Should find the project file
		assert.Contains(t, string(output), "project-alpha")
	})

	t.Run("Step6_AnalyzeLinks", func(t *testing.T) {
		// Analyze link structure
		output, err := runMdnotesCommand("analyze", "links", vaultPath)
		assert.NoError(t, err)

		// Should show link analysis
		assert.Contains(t, string(output), "Link")
	})

	t.Run("Step7_FinalHealthCheck", func(t *testing.T) {
		// Final health check should show improvement
		output, err := runMdnotesCommand("analyze", "health", vaultPath, "--quiet")
		assert.NoError(t, err)

		// Should have a health score
		assert.Contains(t, string(output), "Score:")
	})
}

// TestInboxWorkflow tests the INBOX processing workflow
func TestInboxWorkflow(t *testing.T) {
	// Create vault with INBOX content
	vaultFiles := map[string]string{
		"INBOX.md": `# INBOX

## Quick Notes
- [[https://example.com]] - interesting article
- Meeting notes from today
- Book recommendation: "Test Book"

## TODO
- Process yesterday's notes
- Review project status
- Call client about proposal

## URLs to Process
- https://github.com/example/repo
- https://blog.example.com/post
`,
		"notes/processed-note.md": `---
title: Processed Note
status: complete
---

# Processed Note

This note is already processed.
`,
	}

	vaultPath, err := createTestVault(vaultFiles)
	require.NoError(t, err)
	defer os.RemoveAll(vaultPath)

	t.Run("AnalyzeInbox", func(t *testing.T) {
		// Analyze INBOX content
		output, err := runMdnotesCommand("analyze", "inbox", vaultPath)
		assert.NoError(t, err)

		// Should identify INBOX content
		assert.Contains(t, string(output), "INBOX")
		assert.Contains(t, string(output), "items")
	})

	t.Run("ProcessFrontmatter", func(t *testing.T) {
		// Add frontmatter to INBOX file
		output, err := runMdnotesCommand("frontmatter", "ensure",
			filepath.Join(vaultPath, "INBOX.md"),
			"--field", "type", "--default", "inbox",
			"--field", "status", "--default", "needs-processing")
		assert.NoError(t, err)

		// Should add fields
		assert.Contains(t, string(output), "Processed:")
	})

	t.Run("QueryUnprocessed", func(t *testing.T) {
		// Query for files that need processing
		output, err := runMdnotesCommand("frontmatter", "query", vaultPath,
			"--where", "status = 'needs-processing'")
		assert.NoError(t, err)

		// Should find the INBOX file
		assert.Contains(t, string(output), "INBOX")
	})
}

// TestFileOrganizationWorkflow tests the file organization workflow
func TestFileOrganizationWorkflow(t *testing.T) {
	// Create files that need organization
	vaultFiles := map[string]string{
		"Messy File Name.md": `---
title: Messy File Name
type: note
created: 2024-01-15
---

# Messy File Name

This file has a messy name that should be organized.
`,
		"another messy file.md": `---
title: Another Messy File  
type: project
created: 2024-01-20
---

# Another Messy File

Another file with inconsistent naming.
`,
		"notes/reference.md": `---
title: Reference
---

# Reference

This note references:
- [[Messy File Name]]
- [[another messy file]]
`,
	}

	vaultPath, err := createTestVault(vaultFiles)
	require.NoError(t, err)
	defer os.RemoveAll(vaultPath)

	t.Run("AnalyzeBeforeRename", func(t *testing.T) {
		// Check current state
		output, err := runMdnotesCommand("analyze", "stats", vaultPath)
		assert.NoError(t, err)
		assert.Contains(t, string(output), "Files:")
	})

	t.Run("RenameWithTemplate", func(t *testing.T) {
		// Rename a file using template
		messyFile := filepath.Join(vaultPath, "Messy File Name.md")
		output, err := runMdnotesCommand("rename", messyFile, "--vault", vaultPath,
			"--template", "{{created|date:20060102}}-{{filename|slug}}.md")
		assert.NoError(t, err)

		// Should rename and update links
		assert.Contains(t, string(output), "Renamed:")
		assert.Contains(t, string(output), "Updated")
	})

	t.Run("VerifyLinksUpdated", func(t *testing.T) {
		// Check that links were updated in referencing files
		output, err := runMdnotesCommand("links", "check", vaultPath)
		assert.NoError(t, err)

		// Should not have broken links
		assert.NotContains(t, string(output), "broken")
	})

	t.Run("FinalLinkAnalysis", func(t *testing.T) {
		// Analyze final link structure
		_, err := runMdnotesCommand("analyze", "links", vaultPath, "--quiet")
		assert.NoError(t, err)

		// Should complete successfully
	})
}

// TestDryRunWorkflow tests that dry-run mode works consistently across commands
func TestDryRunWorkflow(t *testing.T) {
	vaultFiles := map[string]string{
		"test.md": `---
title: Test Note
---

# Test Note

Content here.
`,
	}

	vaultPath, err := createTestVault(vaultFiles)
	require.NoError(t, err)
	defer os.RemoveAll(vaultPath)

	dryRunCommands := [][]string{
		{"frontmatter", "ensure", vaultPath, "--field", "new_field", "--default", "value", "--dry-run"},
		{"frontmatter", "set", filepath.Join(vaultPath, "test.md"), "--field", "status", "--value", "active", "--dry-run"},
		{"headings", "fix", vaultPath, "--dry-run"},
		{"rename", filepath.Join(vaultPath, "test.md"), "new-name.md", "--vault", vaultPath, "--dry-run"},
	}

	for _, cmd := range dryRunCommands {
		t.Run("DryRun_"+strings.Join(cmd[:2], "_"), func(t *testing.T) {
			output, err := runMdnotesCommand(cmd...)
			assert.NoError(t, err, "Dry-run command should succeed")

			// Dry-run output should indicate what would be done
			outputStr := string(output)
			assert.True(t,
				strings.Contains(outputStr, "Would") ||
					strings.Contains(outputStr, "would") ||
					strings.Contains(outputStr, "Preview") ||
					len(outputStr) == 0, // Some commands might produce no output in dry-run
				"Dry-run should indicate what would be done: %s", outputStr)
		})
	}

	// Verify no actual changes were made
	t.Run("VerifyNoChanges", func(t *testing.T) {
		// Original file should still exist and be unchanged
		content, err := os.ReadFile(filepath.Join(vaultPath, "test.md"))
		assert.NoError(t, err)

		// Should not contain new_field that was added in dry-run
		assert.NotContains(t, string(content), "new_field")
		assert.NotContains(t, string(content), "status: active")
	})
}
