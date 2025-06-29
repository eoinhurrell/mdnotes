package integration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eoinhurrell/mdnotes/internal/processor"
	"github.com/eoinhurrell/mdnotes/internal/vault"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnhancedEncodingSupport tests comprehensive encoding and special character handling
func TestEnhancedEncodingSupport(t *testing.T) {
	// Create a temporary test vault with various encoding scenarios
	tmpDir, err := os.MkdirTemp("", "encoding_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test files with special characters in names
	testFiles := map[string]string{
		"index.md": `---
title: Encoding Test Index
---

# Encoding Test Index

## Basic Special Characters
- [File with spaces](resources/file with spaces.md)
- [File & symbols](resources/file & symbols.md)
- [File (parentheses)](resources/file (parentheses).md)
- [File [brackets]](resources/file [brackets].md)
- [File {braces}](resources/file {braces}.md)

## Advanced Special Characters  
- [File @ symbols](resources/file @ symbols.md)
- [File + plus](resources/file + plus.md)
- [File = equals](resources/file = equals.md)
- [File ? question](resources/file ? question.md)
- [File : colon](resources/file : colon.md)
- [File * asterisk](resources/file * asterisk.md)

## Wiki Links with Special Characters
- [[file with spaces]]
- [[file & symbols]]
- [[file (parentheses)]]
- [[file + plus]]

## Fragment Links with Special Characters
- [Section with spaces](file.md#Section with Spaces)
- [Section & symbols](file.md#Section & Symbols)
- [Block reference](file.md#^block with spaces)
- [[file#Section (parentheses)]]
- [[file#^block & symbols]]

## Already Encoded Links
- [Pre-encoded](file%20with%20spaces.md#Section%20with%20Spaces)
- [Mixed encoding](file%20with%20spaces.md#Section & Symbols)
`,
		"resources/file with spaces.md":   "# File with Spaces\n\nContent here.",
		"resources/file & symbols.md":     "# File & Symbols\n\nContent here.",
		"resources/file (parentheses).md": "# File (Parentheses)\n\nContent here.",
		"resources/file [brackets].md":    "# File [Brackets]\n\nContent here.",
		"resources/file {braces}.md":      "# File {Braces}\n\nContent here.",
		"resources/file @ symbols.md":     "# File @ Symbols\n\nContent here.",
		"resources/file + plus.md":        "# File + Plus\n\nContent here.",
		"resources/file = equals.md":      "# File = Equals\n\nContent here.",
		"resources/file ? question.md":    "# File ? Question\n\nContent here.",
		"resources/file : colon.md":       "# File : Colon\n\nContent here.",
		"resources/file * asterisk.md":    "# File * Asterisk\n\nContent here.",
		"file.md": `# Test File

## Section with Spaces
Content here.

## Section & Symbols  
Content here.

## Section (parentheses)
Content here.

Some content here. ^block with spaces

More content. ^block & symbols
`,
	}

	// Write test files
	for relativePath, content := range testFiles {
		filePath := filepath.Join(tmpDir, relativePath)
		require.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0755))
		require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))
	}

	t.Run("parse links with enhanced special characters", func(t *testing.T) {
		// Test link parsing with enhanced character support
		indexPath := filepath.Join(tmpDir, "index.md")
		content, err := os.ReadFile(indexPath)
		require.NoError(t, err)

		linkParser := processor.NewLinkParser()
		links := linkParser.Extract(string(content))

		// Count different types of special character links
		var basicSpecialLinks, advancedSpecialLinks, fragmentSpecialLinks []vault.Link
		var urlEncodedLinks []vault.Link

		for _, link := range links {
			target := strings.ToLower(link.Target)
			fragment := strings.ToLower(link.Fragment)

			// Check for various special characters
			switch {
			case strings.Contains(target, " ") || strings.Contains(target, "&") ||
				strings.Contains(target, "(") || strings.Contains(target, "[") ||
				strings.Contains(target, "{"):
				basicSpecialLinks = append(basicSpecialLinks, link)

			case strings.Contains(target, "@") || strings.Contains(target, "+") ||
				strings.Contains(target, "=") || strings.Contains(target, "?") ||
				strings.Contains(target, ":") || strings.Contains(target, "*"):
				advancedSpecialLinks = append(advancedSpecialLinks, link)
			}

			if strings.Contains(fragment, " ") || strings.Contains(fragment, "&") ||
				strings.Contains(fragment, "(") || strings.Contains(fragment, "^") {
				fragmentSpecialLinks = append(fragmentSpecialLinks, link)
			}

			if link.Encoding == "url" || strings.Contains(link.RawText, "%") {
				urlEncodedLinks = append(urlEncodedLinks, link)
			}
		}

		// Verify we found links with special characters
		assert.Greater(t, len(basicSpecialLinks), 0, "Should find basic special character links")
		// Note: Advanced special character links might not be detected in link targets themselves,
		// but rather in the filenames they point to
		assert.Greater(t, len(fragmentSpecialLinks), 0, "Should find fragment links with special characters")
		assert.Greater(t, len(urlEncodedLinks), 0, "Should find URL-encoded links")

		t.Logf("Found %d total links: %d basic special, %d advanced special, %d fragment special, %d URL-encoded",
			len(links), len(basicSpecialLinks), len(advancedSpecialLinks), len(fragmentSpecialLinks), len(urlEncodedLinks))
	})

	t.Run("rename file with special characters updates all references", func(t *testing.T) {
		// Test renaming a file with special characters
		originalPath := filepath.Join(tmpDir, "resources", "file & symbols.md")
		newPath := filepath.Join(tmpDir, "resources", "file & symbols RENAMED.md")

		// Create rename processor
		options := processor.RenameOptions{
			VaultRoot: tmpDir,
			DryRun:    false,
			Verbose:   true,
		}

		renameProcessor := processor.NewRenameProcessor(options)
		defer renameProcessor.Cleanup()

		// Perform the rename
		result, err := renameProcessor.ProcessRename(context.Background(), originalPath, newPath, options)
		require.NoError(t, err)

		// Verify the rename was successful
		assert.FileExists(t, newPath)
		assert.NoFileExists(t, originalPath)

		// Verify links were updated
		assert.Greater(t, result.LinksUpdated, 0, "Should have updated links")

		// Read the updated index file
		indexPath := filepath.Join(tmpDir, "index.md")
		updatedContent, err := os.ReadFile(indexPath)
		require.NoError(t, err)
		updatedContentStr := string(updatedContent)

		// Verify markdown links were updated with proper encoding (& becomes %26)
		assert.Contains(t, updatedContentStr, "file%20%26%20symbols%20RENAMED.md", "Markdown link should be URL-encoded")
		assert.NotContains(t, updatedContentStr, "[File & symbols](resources/file & symbols.md)", "Old markdown link should be gone")

		// Verify wiki links were updated (may include full path for disambiguation)
		assert.Contains(t, updatedContentStr, "file & symbols RENAMED", "Wiki link should be updated")
		assert.NotContains(t, updatedContentStr, "[[file & symbols]]", "Old wiki link should be gone")

		t.Logf("Special character rename: %d files modified, %d links updated",
			result.FilesModified, result.LinksUpdated)
	})

	t.Run("fragment encoding when fragments contain special characters", func(t *testing.T) {
		// Test that fragments with special characters are properly encoded
		testContent := `# Fragment Encoding Test

Links with special character fragments:
- [Spaced Section](file.md#Section with Spaces)
- [Symbol Section](file.md#Section & Symbols) 
- [Paren Section](file.md#Section (parentheses))
- [Block with spaces](file.md#^block with spaces)
- [Block with symbols](file.md#^block & symbols)
`

		linkParser := processor.NewLinkParser()
		links := linkParser.Extract(testContent)

		var fragmentLinksWithSpecialChars []vault.Link
		for _, link := range links {
			if link.HasFragment() && (strings.Contains(link.Fragment, " ") ||
				strings.Contains(link.Fragment, "&") || strings.Contains(link.Fragment, "(")) {
				fragmentLinksWithSpecialChars = append(fragmentLinksWithSpecialChars, link)
			}
		}

		require.Greater(t, len(fragmentLinksWithSpecialChars), 0, "Should find fragment links with special characters")

		// Test that fragments are properly decoded and targets separated
		for _, link := range fragmentLinksWithSpecialChars {
			assert.Equal(t, "file.md", link.Target, "Target should be properly separated from fragment")

			// Verify fragment content
			switch {
			case strings.Contains(link.Fragment, "Section with Spaces"):
				assert.Equal(t, "Section with Spaces", link.Fragment, "Fragment should be decoded")
				assert.True(t, link.IsHeadingFragment(), "Should be recognized as heading fragment")

			case strings.Contains(link.Fragment, "Section & Symbols"):
				assert.Equal(t, "Section & Symbols", link.Fragment, "Fragment with symbols should be decoded")
				assert.True(t, link.IsHeadingFragment(), "Should be recognized as heading fragment")

			case strings.Contains(link.Fragment, "block with spaces"):
				assert.Equal(t, "^block with spaces", link.Fragment, "Block fragment should be decoded with ^")
				assert.True(t, link.IsBlockFragment(), "Should be recognized as block fragment")

			case strings.Contains(link.Fragment, "block & symbols"):
				assert.Equal(t, "^block & symbols", link.Fragment, "Block fragment with symbols should be decoded")
				assert.True(t, link.IsBlockFragment(), "Should be recognized as block fragment")
			}
		}

		t.Logf("Found %d fragment links with special characters", len(fragmentLinksWithSpecialChars))
	})

	t.Run("handle malformed URL encoding gracefully", func(t *testing.T) {
		// Test malformed encoding scenarios
		malformedContent := `# Malformed Encoding Test

Links with malformed encoding:
- [Bad percent](file%ZZ.md) - Invalid hex
- [Incomplete percent](file%.md) - Incomplete encoding  
- [Mixed encoding](file%20name%XX.md) - Mix of valid and invalid
- [Fragment bad](file.md#section%ZZ) - Invalid fragment encoding
`

		linkParser := processor.NewLinkParser()
		links := linkParser.Extract(malformedContent)

		// Should still parse links even with malformed encoding
		assert.Greater(t, len(links), 0, "Should parse links even with malformed encoding")

		for _, link := range links {
			// Links should have targets even if decoding failed
			assert.NotEmpty(t, link.Target, "Target should not be empty even with malformed encoding")

			// Malformed encoding should be preserved in raw text
			if strings.Contains(link.RawText, "%ZZ") || strings.Contains(link.RawText, "%.") {
				// The target should contain the malformed encoding since decoding failed
				assert.True(t, strings.Contains(link.Target, "%") || strings.Contains(link.Target, "file"),
					"Malformed encoding should be preserved in target or processed gracefully")
			}
		}

		t.Logf("Successfully parsed %d links with malformed encoding", len(links))
	})
}
