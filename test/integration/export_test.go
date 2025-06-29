package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExportIntegration tests the complete export workflow
func TestExportIntegration(t *testing.T) {
	// Create test vault with realistic content
	vaultFiles := map[string]string{
		"areas/philosophy/stoicism.md": `---
title: Stoicism
type: philosophy
status: published
tags: [philosophy, stoicism, marcus-aurelius]
created: 2024-01-15
---

# Stoicism

Stoicism is a school of philosophy founded in Athens.

## Key Principles
- Focus on what you can control
- Accept what you cannot change
- Live according to nature

## Philosophers
- [[Marcus Aurelius]]
- [[Epictetus]]
- [[Seneca]]
`,
		"areas/philosophy/marcus-aurelius.md": `---
title: Marcus Aurelius
type: biography
status: published
tags: [philosophy, stoicism, emperor]
created: 2024-01-16
---

# Marcus Aurelius

Roman Emperor and Stoic philosopher.

## Background
Born in 121 AD, died in 180 AD.

## Works
- Meditations
`,
		"projects/blog-post.md": `---
title: Blog Post About Philosophy
type: blog
status: draft
tags: [writing, philosophy]
created: 2024-01-20
---

# Blog Post About Philosophy

This is a draft blog post about [[Stoicism]].

It's not ready for publication yet.
`,
		"resources/books/meditations.md": `---
title: Meditations
type: book
author: Marcus Aurelius
status: published
tags: [philosophy, stoicism, books]
created: 2024-01-10
---

# Meditations

Personal writings of Roman Emperor [[Marcus Aurelius]].

## Summary
Collection of personal notes and ideas.
`,
		"inbox/quick-note.md": `---
title: Quick Note
status: draft
tags: [inbox, temporary]
---

# Quick Note

Just a quick capture.
`,
	}

	vaultPath, err := createTestVault(vaultFiles)
	require.NoError(t, err)
	defer os.RemoveAll(vaultPath)

	t.Run("Export_All_Files", func(t *testing.T) {
		outputDir, err := os.MkdirTemp("", "export-all-*")
		require.NoError(t, err)
		defer os.RemoveAll(outputDir)

		output, err := runMdnotesCommand("export", outputDir, vaultPath)
		assert.NoError(t, err)
		assert.Contains(t, string(output), "Export completed successfully")
		assert.Contains(t, string(output), "Exported 5 files")

		// Verify all files were copied with directory structure preserved
		assert.FileExists(t, filepath.Join(outputDir, "areas", "philosophy", "stoicism.md"))
		assert.FileExists(t, filepath.Join(outputDir, "areas", "philosophy", "marcus-aurelius.md"))
		assert.FileExists(t, filepath.Join(outputDir, "projects", "blog-post.md"))
		assert.FileExists(t, filepath.Join(outputDir, "resources", "books", "meditations.md"))
		assert.FileExists(t, filepath.Join(outputDir, "inbox", "quick-note.md"))

		// Verify content is preserved
		content, err := os.ReadFile(filepath.Join(outputDir, "areas", "philosophy", "stoicism.md"))
		require.NoError(t, err)
		assert.Contains(t, string(content), "Stoicism is a school of philosophy")
		assert.Contains(t, string(content), "[[Marcus Aurelius]]")
	})

	t.Run("Export_Published_Only", func(t *testing.T) {
		outputDir, err := os.MkdirTemp("", "export-published-*")
		require.NoError(t, err)
		defer os.RemoveAll(outputDir)

		output, err := runMdnotesCommand("export", outputDir, vaultPath, "--query", "status = 'published'")
		assert.NoError(t, err)
		assert.Contains(t, string(output), "Export completed successfully")
		assert.Contains(t, string(output), "Exported 3 files")

		// Verify only published files were copied
		assert.FileExists(t, filepath.Join(outputDir, "areas", "philosophy", "stoicism.md"))
		assert.FileExists(t, filepath.Join(outputDir, "areas", "philosophy", "marcus-aurelius.md"))
		assert.FileExists(t, filepath.Join(outputDir, "resources", "books", "meditations.md"))

		// Verify draft files were not copied
		assert.NoFileExists(t, filepath.Join(outputDir, "projects", "blog-post.md"))
		assert.NoFileExists(t, filepath.Join(outputDir, "inbox", "quick-note.md"))
	})

	t.Run("Export_Philosophy_Only", func(t *testing.T) {
		outputDir, err := os.MkdirTemp("", "export-philosophy-*")
		require.NoError(t, err)
		defer os.RemoveAll(outputDir)

		output, err := runMdnotesCommand("export", outputDir, vaultPath, "--query", "tags contains 'philosophy'")
		assert.NoError(t, err)
		assert.Contains(t, string(output), "Export completed successfully")
		assert.Contains(t, string(output), "Exported 4 files")

		// Verify philosophy-related files were copied
		assert.FileExists(t, filepath.Join(outputDir, "areas", "philosophy", "stoicism.md"))
		assert.FileExists(t, filepath.Join(outputDir, "areas", "philosophy", "marcus-aurelius.md"))
		assert.FileExists(t, filepath.Join(outputDir, "projects", "blog-post.md"))
		assert.FileExists(t, filepath.Join(outputDir, "resources", "books", "meditations.md"))

		// Verify non-philosophy files were not copied
		assert.NoFileExists(t, filepath.Join(outputDir, "inbox", "quick-note.md"))
	})

	t.Run("Export_Dry_Run", func(t *testing.T) {
		outputDir, err := os.MkdirTemp("", "export-dry-*")
		require.NoError(t, err)
		defer os.RemoveAll(outputDir)

		output, err := runMdnotesCommand("export", outputDir, vaultPath, "--dry-run")
		assert.NoError(t, err)
		assert.Contains(t, string(output), "Export Summary (Dry Run)")
		assert.Contains(t, string(output), "Would export 5 files")
		assert.Contains(t, string(output), "Files scanned:  5")

		// Verify no files were actually copied
		entries, err := os.ReadDir(outputDir)
		require.NoError(t, err)
		assert.Empty(t, entries, "Output directory should be empty in dry-run mode")
	})

	t.Run("Export_Dry_Run_With_Query", func(t *testing.T) {
		outputDir, err := os.MkdirTemp("", "export-dry-query-*")
		require.NoError(t, err)
		defer os.RemoveAll(outputDir)

		output, err := runMdnotesCommand("export", outputDir, vaultPath, "--dry-run", "--query", "type = 'blog'")
		assert.NoError(t, err)
		assert.Contains(t, string(output), "Export Summary (Dry Run)")
		assert.Contains(t, string(output), "Would export 1 files")
		assert.Contains(t, string(output), "Files scanned:  5")
		assert.Contains(t, string(output), "Files selected: 1")

		// Verify no files were copied
		entries, err := os.ReadDir(outputDir)
		require.NoError(t, err)
		assert.Empty(t, entries)
	})

	t.Run("Export_Verbose_Output", func(t *testing.T) {
		outputDir, err := os.MkdirTemp("", "export-verbose-*")
		require.NoError(t, err)
		defer os.RemoveAll(outputDir)

		output, err := runMdnotesCommand("export", outputDir, vaultPath, "--verbose")
		assert.NoError(t, err)

		outputStr := string(output)
		assert.Contains(t, outputStr, "Exporting from:")
		assert.Contains(t, outputStr, "Exporting to:")
		assert.Contains(t, outputStr, "Processing details:")
		assert.Contains(t, outputStr, "Files scanned: 5")
		assert.Contains(t, outputStr, "Files exported: 5")
		assert.Contains(t, outputStr, "Processing time:")
		assert.Contains(t, outputStr, "Exported files:")

		// Should show all file paths
		assert.Contains(t, outputStr, "areas/philosophy/stoicism.md")
		assert.Contains(t, outputStr, "projects/blog-post.md")
	})

	t.Run("Export_Complex_Query", func(t *testing.T) {
		outputDir, err := os.MkdirTemp("", "export-complex-*")
		require.NoError(t, err)
		defer os.RemoveAll(outputDir)

		// Export philosophy files that are published
		output, err := runMdnotesCommand("export", outputDir, vaultPath,
			"--query", "tags contains 'philosophy' AND status = 'published'")
		assert.NoError(t, err)
		assert.Contains(t, string(output), "Exported 3 files")

		// Verify correct files were exported
		assert.FileExists(t, filepath.Join(outputDir, "areas", "philosophy", "stoicism.md"))
		assert.FileExists(t, filepath.Join(outputDir, "areas", "philosophy", "marcus-aurelius.md"))
		assert.FileExists(t, filepath.Join(outputDir, "resources", "books", "meditations.md"))

		// Verify blog post was not exported (has philosophy tag but is draft)
		assert.NoFileExists(t, filepath.Join(outputDir, "projects", "blog-post.md"))
	})

	t.Run("Export_No_Matching_Files", func(t *testing.T) {
		outputDir, err := os.MkdirTemp("", "export-no-match-*")
		require.NoError(t, err)
		defer os.RemoveAll(outputDir)

		output, err := runMdnotesCommand("export", outputDir, vaultPath,
			"--query", "tags contains 'nonexistent'", "--dry-run")
		assert.NoError(t, err)

		outputStr := string(output)
		assert.Contains(t, outputStr, "No files match the criteria")
		assert.Contains(t, outputStr, "Files scanned:  5")
		assert.Contains(t, outputStr, "Files selected: 0")
	})
}

// TestExportErrorCases tests various error conditions
func TestExportErrorCases(t *testing.T) {
	t.Run("Nonexistent_Vault", func(t *testing.T) {
		outputDir, err := os.MkdirTemp("", "export-error-*")
		require.NoError(t, err)
		defer os.RemoveAll(outputDir)

		output, err := runMdnotesCommand("export", outputDir, "/nonexistent/vault")
		assert.Error(t, err)
		assert.Contains(t, string(output), "vault path does not exist")
	})

	t.Run("Invalid_Query", func(t *testing.T) {
		vaultPath, err := createTestVault(map[string]string{
			"test.md": "# Test",
		})
		require.NoError(t, err)
		defer os.RemoveAll(vaultPath)

		outputDir, err := os.MkdirTemp("", "export-query-error-*")
		require.NoError(t, err)
		defer os.RemoveAll(outputDir)

		output, err := runMdnotesCommand("export", outputDir, vaultPath, "--query", "invalid query syntax !!!")
		assert.Error(t, err)
		assert.Contains(t, string(output), "parsing query")
	})

	t.Run("Output_Directory_Not_Empty", func(t *testing.T) {
		vaultPath, err := createTestVault(map[string]string{
			"test.md": "# Test",
		})
		require.NoError(t, err)
		defer os.RemoveAll(vaultPath)

		outputDir, err := os.MkdirTemp("", "export-not-empty-*")
		require.NoError(t, err)
		defer os.RemoveAll(outputDir)

		// Create a file in the output directory
		err = os.WriteFile(filepath.Join(outputDir, "existing.txt"), []byte("content"), 0644)
		require.NoError(t, err)

		output, err := runMdnotesCommand("export", outputDir, vaultPath)
		assert.Error(t, err)
		assert.Contains(t, string(output), "output directory is not empty")
	})
}

// TestExportPerformance tests export performance with larger vaults
func TestExportPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Create a larger vault for performance testing
	vaultFiles := make(map[string]string)
	for i := 0; i < 100; i++ {
		filename := filepath.Join("notes", "note"+string(rune(i%26+'a'))+string(rune((i/26)%26+'a'))+".md")
		content := generateTestContent(i)
		vaultFiles[filename] = content
	}

	vaultPath, err := createTestVault(vaultFiles)
	require.NoError(t, err)
	defer os.RemoveAll(vaultPath)

	t.Run("Export_100_Files", func(t *testing.T) {
		outputDir, err := os.MkdirTemp("", "export-perf-*")
		require.NoError(t, err)
		defer os.RemoveAll(outputDir)

		output, err := runMdnotesCommand("export", outputDir, vaultPath)
		assert.NoError(t, err)
		assert.Contains(t, string(output), "Exported 100 files")

		// Verify some files exist
		assert.FileExists(t, filepath.Join(outputDir, "notes", "noteaa.md"))
		assert.FileExists(t, filepath.Join(outputDir, "notes", "noteba.md"))
	})

	t.Run("Export_With_Query_Performance", func(t *testing.T) {
		outputDir, err := os.MkdirTemp("", "export-query-perf-*")
		require.NoError(t, err)
		defer os.RemoveAll(outputDir)

		// Export only project files
		output, err := runMdnotesCommand("export", outputDir, vaultPath, "--query", "type = 'project'")
		assert.NoError(t, err)

		// Should export roughly half the files (those with index % 2 == 1)
		outputStr := string(output)
		assert.Contains(t, outputStr, "Export completed successfully")

		// Verify some project files exist
		entries, err := os.ReadDir(filepath.Join(outputDir, "notes"))
		if err == nil {
			assert.Greater(t, len(entries), 0, "Should export some project files")
		}
	})
}

// TestExportEdgeCases tests edge cases and corner scenarios
func TestExportEdgeCases(t *testing.T) {
	t.Run("Empty_Vault", func(t *testing.T) {
		vaultPath, err := createTestVault(map[string]string{})
		require.NoError(t, err)
		defer os.RemoveAll(vaultPath)

		outputDir, err := os.MkdirTemp("", "export-empty-*")
		require.NoError(t, err)
		defer os.RemoveAll(outputDir)

		output, err := runMdnotesCommand("export", outputDir, vaultPath)
		assert.NoError(t, err)
		assert.Contains(t, string(output), "Exported 0 files")
	})

	t.Run("Files_With_Special_Characters", func(t *testing.T) {
		vaultFiles := map[string]string{
			"file with spaces.md":      "# File with spaces",
			"file-with-hyphens.md":     "# File with hyphens",
			"file_with_underscores.md": "# File with underscores",
			"unicode-file-名前.md":       "# Unicode file",
		}

		vaultPath, err := createTestVault(vaultFiles)
		require.NoError(t, err)
		defer os.RemoveAll(vaultPath)

		outputDir, err := os.MkdirTemp("", "export-special-*")
		require.NoError(t, err)
		defer os.RemoveAll(outputDir)

		output, err := runMdnotesCommand("export", outputDir, vaultPath)
		assert.NoError(t, err)
		assert.Contains(t, string(output), "Exported 4 files")

		// Verify all files were copied correctly
		assert.FileExists(t, filepath.Join(outputDir, "file with spaces.md"))
		assert.FileExists(t, filepath.Join(outputDir, "file-with-hyphens.md"))
		assert.FileExists(t, filepath.Join(outputDir, "file_with_underscores.md"))
		assert.FileExists(t, filepath.Join(outputDir, "unicode-file-名前.md"))
	})

	t.Run("Deep_Directory_Structure", func(t *testing.T) {
		vaultFiles := map[string]string{
			"a/b/c/d/e/f/deep.md": "# Deep file",
		}

		vaultPath, err := createTestVault(vaultFiles)
		require.NoError(t, err)
		defer os.RemoveAll(vaultPath)

		outputDir, err := os.MkdirTemp("", "export-deep-*")
		require.NoError(t, err)
		defer os.RemoveAll(outputDir)

		output, err := runMdnotesCommand("export", outputDir, vaultPath)
		assert.NoError(t, err)
		assert.Contains(t, string(output), "Exported 1 files")

		// Verify deep structure is preserved
		assert.FileExists(t, filepath.Join(outputDir, "a", "b", "c", "d", "e", "f", "deep.md"))
	})
}
