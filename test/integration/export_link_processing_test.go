package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExportLinkProcessing tests the complete link processing workflow
func TestExportLinkProcessing(t *testing.T) {
	// Create test vault with various link types
	vaultFiles := map[string]string{
		"notes/internal-note.md": `---
title: Internal Note
---

# Internal Note

This note is included in exports.`,

		"notes/external-note.md": `---
title: External Note  
---

# External Note

This note is NOT included in exports.`,

		"blog/published-post.md": `---
title: Published Blog Post
tags: [published, blog]
url: https://example.com/blog-post
---

# Published Blog Post

This post has various types of links:

## Internal Links (should be preserved)
- [[internal-note]] - wiki link to exported note
- [Internal Note](../notes/internal-note.md) - markdown link to exported note

## External Links (should be processed based on strategy)
- [[external-note]] - wiki link to non-exported note
- [External Note](../notes/external-note.md) - markdown link to non-exported note
- [[missing-note|Custom Display]] - wiki link with custom text

## URL Links (should be preserved)
- [Example Website](https://example.com) - external URL
- [Email Contact](mailto:contact@example.com) - email link

## Assets
- ![[diagram.png]] - embedded image
- ![Screenshot](../assets/screenshot.png) - markdown image`,

		"blog/draft-post.md": `---
title: Draft Post
tags: [draft]
source: https://source.com/reference
---

# Draft Post

Links in draft: [[external-note]] and [missing](missing.md)`,

		"assets/diagram.png": "fake-image-content",
	}

	vaultPath, err := createTestVault(vaultFiles)
	require.NoError(t, err)
	defer os.RemoveAll(vaultPath)

	t.Run("Export_With_Remove_Strategy", func(t *testing.T) {
		outputDir, err := os.MkdirTemp("", "export-link-remove-*")
		require.NoError(t, err)
		defer os.RemoveAll(outputDir)

		// Export only published posts with remove strategy
		output, err := runMdnotesCommand("export", outputDir, vaultPath,
			"--query", "tags contains 'published'",
			"--link-strategy", "remove")
		assert.NoError(t, err)

		outputStr := string(output)
		assert.Contains(t, outputStr, "Export completed successfully")
		assert.Contains(t, outputStr, "Exported 1 files")
		assert.Contains(t, outputStr, "External links removed:")
		assert.Contains(t, outputStr, "Files with links processed: 1")

		// Check that the exported file has processed links
		exportedFile := filepath.Join(outputDir, "blog", "published-post.md")
		assert.FileExists(t, exportedFile)

		content, err := os.ReadFile(exportedFile)
		require.NoError(t, err)
		contentStr := string(content)

		// Since we're only exporting published posts, internal-note is NOT exported
		// so links to it should be converted to plain text (treated as external)
		assert.Contains(t, contentStr, "internal-note - wiki link to exported note")
		assert.Contains(t, contentStr, "Internal Note - markdown link to exported note")

		// External links should be converted to plain text
		assert.Contains(t, contentStr, "external-note - wiki link to non-exported note")
		assert.Contains(t, contentStr, "External Note - markdown link to non-exported note")
		assert.Contains(t, contentStr, "Custom Display - wiki link with custom text")

		// URL links should be preserved
		assert.Contains(t, contentStr, "[Example Website](https://example.com)")
		assert.Contains(t, contentStr, "[Email Contact](mailto:contact@example.com)")

		// Assets should be unchanged for now (Phase 3 feature)
		assert.Contains(t, contentStr, "![[diagram.png]]")
		assert.Contains(t, contentStr, "![Screenshot](../assets/screenshot.png)")
	})

	t.Run("Export_With_URL_Strategy", func(t *testing.T) {
		outputDir, err := os.MkdirTemp("", "export-link-url-*")
		require.NoError(t, err)
		defer os.RemoveAll(outputDir)

		// Export published post with URL strategy
		output, err := runMdnotesCommand("export", outputDir, vaultPath,
			"--query", "tags contains 'published'",
			"--link-strategy", "url")
		assert.NoError(t, err)

		outputStr := string(output)
		assert.Contains(t, outputStr, "Export completed successfully")
		assert.Contains(t, outputStr, "External links converted:")

		// Check that the exported file has URL conversions
		exportedFile := filepath.Join(outputDir, "blog", "published-post.md")
		content, err := os.ReadFile(exportedFile)
		require.NoError(t, err)
		contentStr := string(content)

		// External links should be converted to URLs from frontmatter
		assert.Contains(t, contentStr, "[external-note](https://example.com/blog-post)")
		assert.Contains(t, contentStr, "[External Note](https://example.com/blog-post)")
		assert.Contains(t, contentStr, "[Custom Display](https://example.com/blog-post)")
	})

	t.Run("Export_With_Different_URL_Field", func(t *testing.T) {
		outputDir, err := os.MkdirTemp("", "export-link-source-*")
		require.NoError(t, err)
		defer os.RemoveAll(outputDir)

		// Export draft post (has 'source' field instead of 'url')
		output, err := runMdnotesCommand("export", outputDir, vaultPath,
			"--query", "tags contains 'draft'",
			"--link-strategy", "url")
		assert.NoError(t, err)

		outputStr := string(output)
		assert.Contains(t, outputStr, "External links converted:")

		// Check that links use the 'source' field URL
		exportedFile := filepath.Join(outputDir, "blog", "draft-post.md")
		content, err := os.ReadFile(exportedFile)
		require.NoError(t, err)
		contentStr := string(content)

		assert.Contains(t, contentStr, "[external-note](https://source.com/reference)")
		assert.Contains(t, contentStr, "[missing](https://source.com/reference)")
	})

	t.Run("Export_Without_Link_Processing", func(t *testing.T) {
		outputDir, err := os.MkdirTemp("", "export-no-links-*")
		require.NoError(t, err)
		defer os.RemoveAll(outputDir)

		// Export without link processing
		output, err := runMdnotesCommand("export", outputDir, vaultPath,
			"--query", "tags contains 'published'",
			"--process-links=false")
		assert.NoError(t, err)

		outputStr := string(output)
		assert.Contains(t, outputStr, "Export completed successfully")
		// Should not contain link processing statistics
		assert.NotContains(t, outputStr, "External links")
		assert.NotContains(t, outputStr, "Links processed")

		// Check that links are unchanged
		exportedFile := filepath.Join(outputDir, "blog", "published-post.md")
		content, err := os.ReadFile(exportedFile)
		require.NoError(t, err)
		contentStr := string(content)

		// All links should be exactly as in the original
		assert.Contains(t, contentStr, "[[external-note]]")
		assert.Contains(t, contentStr, "[External Note](../notes/external-note.md)")
		assert.Contains(t, contentStr, "[[missing-note|Custom Display]]")
	})

	t.Run("Export_All_With_Link_Processing", func(t *testing.T) {
		outputDir, err := os.MkdirTemp("", "export-all-links-*")
		require.NoError(t, err)
		defer os.RemoveAll(outputDir)

		// Export all files (no query) with link processing
		output, err := runMdnotesCommand("export", outputDir, vaultPath,
			"--link-strategy", "remove")
		assert.NoError(t, err)

		outputStr := string(output)
		assert.Contains(t, outputStr, "Export completed successfully")
		assert.Contains(t, outputStr, "Exported 4 files")              // 4 markdown files
		assert.Contains(t, outputStr, "Files with links processed: 2") // Both blog posts have links

		// Verify both blog files are processed
		publishedFile := filepath.Join(outputDir, "blog", "published-post.md")
		assert.FileExists(t, publishedFile)

		draftFile := filepath.Join(outputDir, "blog", "draft-post.md")
		assert.FileExists(t, draftFile)

		// Verify internal notes are preserved
		internalFile := filepath.Join(outputDir, "notes", "internal-note.md")
		assert.FileExists(t, internalFile)

		externalFile := filepath.Join(outputDir, "notes", "external-note.md")
		assert.FileExists(t, externalFile)
	})

	t.Run("Dry_Run_With_Link_Processing", func(t *testing.T) {
		outputDir, err := os.MkdirTemp("", "export-dry-links-*")
		require.NoError(t, err)
		defer os.RemoveAll(outputDir)

		// Dry run with link processing
		output, err := runMdnotesCommand("export", outputDir, vaultPath,
			"--query", "tags contains 'published'",
			"--link-strategy", "remove",
			"--dry-run")
		assert.NoError(t, err)

		outputStr := string(output)
		assert.Contains(t, outputStr, "Export Summary (Dry Run)")
		assert.Contains(t, outputStr, "Would export 1 files")
		assert.Contains(t, outputStr, "Link processing (would be performed):")
		assert.Contains(t, outputStr, "External links removed:")
		assert.Contains(t, outputStr, "Files with links processed: 1")

		// Verify no files were actually created
		entries, err := os.ReadDir(outputDir)
		require.NoError(t, err)
		assert.Empty(t, entries, "Output directory should be empty in dry-run mode")
	})
}

// TestExportLinkProcessingEdgeCases tests edge cases in link processing
func TestExportLinkProcessingEdgeCases(t *testing.T) {
	t.Run("Invalid_Link_Strategy", func(t *testing.T) {
		vaultPath, err := createTestVault(map[string]string{
			"test.md": "# Test",
		})
		require.NoError(t, err)
		defer os.RemoveAll(vaultPath)

		outputDir, err := os.MkdirTemp("", "export-invalid-strategy-*")
		require.NoError(t, err)
		defer os.RemoveAll(outputDir)

		output, err := runMdnotesCommand("export", outputDir, vaultPath,
			"--link-strategy", "invalid")
		assert.Error(t, err)
		outputStr := string(output)
		assert.Contains(t, outputStr, "invalid link strategy")
	})

	t.Run("Links_Without_Frontmatter_URL", func(t *testing.T) {
		vaultFiles := map[string]string{
			"no-url.md": `---
title: No URL Field
---

# No URL

External link: [[missing-note]]`,
		}

		vaultPath, err := createTestVault(vaultFiles)
		require.NoError(t, err)
		defer os.RemoveAll(vaultPath)

		outputDir, err := os.MkdirTemp("", "export-no-url-*")
		require.NoError(t, err)
		defer os.RemoveAll(outputDir)

		// With URL strategy but no URL in frontmatter, should fall back to remove
		output, err := runMdnotesCommand("export", outputDir, vaultPath,
			"--link-strategy", "url")
		assert.NoError(t, err)

		outputStr := string(output)
		assert.Contains(t, outputStr, "External links removed:")

		// Check that link was converted to plain text (fallback behavior)
		exportedFile := filepath.Join(outputDir, "no-url.md")
		content, err := os.ReadFile(exportedFile)
		require.NoError(t, err)
		contentStr := string(content)

		assert.Contains(t, contentStr, "missing-note")
		assert.NotContains(t, contentStr, "[[missing-note]]")
	})

	t.Run("Files_With_No_Links", func(t *testing.T) {
		vaultFiles := map[string]string{
			"no-links.md": `---
title: No Links
---

# No Links

This file has no links at all.
Just plain text content.`,
		}

		vaultPath, err := createTestVault(vaultFiles)
		require.NoError(t, err)
		defer os.RemoveAll(vaultPath)

		outputDir, err := os.MkdirTemp("", "export-no-links-*")
		require.NoError(t, err)
		defer os.RemoveAll(outputDir)

		output, err := runMdnotesCommand("export", outputDir, vaultPath,
			"--link-strategy", "remove")
		assert.NoError(t, err)

		outputStr := string(output)
		assert.Contains(t, outputStr, "Export completed successfully")
		// Should not show link processing statistics when no links are processed
		assert.NotContains(t, outputStr, "External links")
		assert.NotContains(t, outputStr, "Files with links processed")
	})

	t.Run("Complex_Link_Scenarios", func(t *testing.T) {
		vaultFiles := map[string]string{
			"exported.md": `# Exported Note`,
			"complex.md": `---
title: Complex Links
website: https://website.com
---

# Complex Links

Multiple scenarios:
- [[exported]] - internal (should preserve)
- [[missing1]] and [[missing2|Display]] - external (should process)
- [Link1](missing1.md) and [Link2](missing2.md) - external (should process)
- [Google](https://google.com) - URL (should preserve)
- Normal text without links
- [[exported|Custom Name]] - internal with alias (should preserve)`,
		}

		vaultPath, err := createTestVault(vaultFiles)
		require.NoError(t, err)
		defer os.RemoveAll(vaultPath)

		outputDir, err := os.MkdirTemp("", "export-complex-*")
		require.NoError(t, err)
		defer os.RemoveAll(outputDir)

		output, err := runMdnotesCommand("export", outputDir, vaultPath,
			"--link-strategy", "url")
		assert.NoError(t, err)

		outputStr := string(output)
		assert.Contains(t, outputStr, "External links converted: 4")
		assert.Contains(t, outputStr, "Files with links processed: 1")

		// Check the processed content
		exportedFile := filepath.Join(outputDir, "complex.md")
		content, err := os.ReadFile(exportedFile)
		require.NoError(t, err)
		contentStr := string(content)

		// Internal links preserved
		assert.Contains(t, contentStr, "[[exported]]")
		assert.Contains(t, contentStr, "[[exported|Custom Name]]")

		// External links converted to URLs
		assert.Contains(t, contentStr, "[missing1](https://website.com)")
		assert.Contains(t, contentStr, "[Display](https://website.com)")
		assert.Contains(t, contentStr, "[Link1](https://website.com)")
		assert.Contains(t, contentStr, "[Link2](https://website.com)")

		// URL links preserved
		assert.Contains(t, contentStr, "[Google](https://google.com)")

		// Normal text unchanged
		assert.Contains(t, contentStr, "Normal text without links")
	})
}
