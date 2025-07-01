package integration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eoinhurrell/mdnotes/internal/processor"
	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// TestFragmentLinkProcessing tests comprehensive fragment link support
func TestFragmentLinkProcessing(t *testing.T) {
	// Create a temporary test vault with fragment links
	tmpDir, err := os.MkdirTemp("", "fragment_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test files
	testFiles := map[string]string{
		"index.md": `---
title: Main Index
---

# Main Index

## Wiki Links with Fragments
- [[research#Introduction]] - Link to introduction section
- [[research#^block123]] - Link to specific block
- [[notes/deep-dive#Theory Section]] - Link to section with spaces
- [[notes/deep-dive#^ref456]] - Link to block reference

## Markdown Links with Fragments  
- [Research Intro](research.md#Introduction) - Markdown link to section
- [Deep Theory](notes/deep-dive.md#Theory%20Section) - URL-encoded section
- [Block Reference](research.md#^block123) - Markdown link to block

## Embed Links with Fragments
- ![[research#Introduction]] - Embed introduction section
- ![[research#^block123]] - Embed specific block
`,
		"research.md": `---
title: Research Notes
---

# Research Notes

## Introduction
This is the introduction section.

## Methods
Research methodology here.

Some content here. ^block123

## Results
The research results.
`,
		"notes/deep-dive.md": `---
title: Deep Dive Analysis
---

# Deep Dive Analysis

## Theory Section
Theoretical background and analysis.

## Implementation
Implementation details here. ^ref456

## Conclusion
Final thoughts and conclusions.
`,
	}

	// Write test files
	for relativePath, content := range testFiles {
		filePath := filepath.Join(tmpDir, relativePath)
		require.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0755))
		require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))
	}

	t.Run("parse fragment links correctly", func(t *testing.T) {
		// Test link parsing
		indexPath := filepath.Join(tmpDir, "index.md")
		content, err := os.ReadFile(indexPath)
		require.NoError(t, err)

		linkParser := processor.NewLinkParser()
		links := linkParser.Extract(string(content))

		// Count different types of fragment links
		var wikiFragmentLinks, markdownFragmentLinks, embedFragmentLinks []vault.Link
		var headingFragments, blockFragments []vault.Link

		for _, link := range links {
			if link.HasFragment() {
				switch link.Type {
				case vault.WikiLink:
					wikiFragmentLinks = append(wikiFragmentLinks, link)
				case vault.MarkdownLink:
					markdownFragmentLinks = append(markdownFragmentLinks, link)
				case vault.EmbedLink:
					embedFragmentLinks = append(embedFragmentLinks, link)
				}

				if link.IsHeadingFragment() {
					headingFragments = append(headingFragments, link)
				} else if link.IsBlockFragment() {
					blockFragments = append(blockFragments, link)
				}
			}
		}

		// Verify we found the expected number of fragment links
		assert.Equal(t, 4, len(wikiFragmentLinks), "Should find 4 wiki links with fragments")
		assert.Equal(t, 3, len(markdownFragmentLinks), "Should find 3 markdown links with fragments")
		assert.Equal(t, 2, len(embedFragmentLinks), "Should find 2 embed links with fragments")

		// Verify fragment types
		assert.Equal(t, 5, len(headingFragments), "Should find 5 heading fragments")
		assert.Equal(t, 4, len(blockFragments), "Should find 4 block fragments")

		// Test specific fragment parsing
		for _, link := range links {
			if link.Target == "research" && link.Fragment == "Introduction" {
				assert.True(t, link.IsHeadingFragment(), "Introduction should be a heading fragment")
				assert.False(t, link.IsBlockFragment(), "Introduction should not be a block fragment")
				assert.Equal(t, "research#Introduction", link.FullTarget(), "FullTarget should include fragment")
			}

			if link.Target == "research" && link.Fragment == "^block123" {
				assert.False(t, link.IsHeadingFragment(), "^block123 should not be a heading fragment")
				assert.True(t, link.IsBlockFragment(), "^block123 should be a block fragment")
				assert.Equal(t, "research#^block123", link.FullTarget(), "FullTarget should include block reference")
			}

			if link.Target == "notes/deep-dive.md" && link.Fragment == "Theory Section" {
				assert.True(t, link.IsHeadingFragment(), "Theory Section should be a heading fragment")
				assert.Equal(t, "notes/deep-dive.md#Theory Section", link.FullTarget(), "FullTarget should handle spaces in fragments")
			}
		}

		t.Logf("Found %d total links, %d with fragments (%d headings, %d blocks)",
			len(links), len(wikiFragmentLinks)+len(markdownFragmentLinks)+len(embedFragmentLinks),
			len(headingFragments), len(blockFragments))
	})

	t.Run("rename file with fragment links updates correctly", func(t *testing.T) {
		// Test renaming a file that has incoming fragment links
		originalPath := filepath.Join(tmpDir, "research.md")
		newPath := filepath.Join(tmpDir, "research-updated.md")

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

		// Verify fragment links were updated
		assert.Greater(t, result.LinksUpdated, 0, "Should have updated fragment links")

		// Read the updated index file
		indexPath := filepath.Join(tmpDir, "index.md")
		updatedContent, err := os.ReadFile(indexPath)
		require.NoError(t, err)
		updatedContentStr := string(updatedContent)

		// Verify fragment links were updated correctly
		assert.Contains(t, updatedContentStr, "[[research-updated#Introduction]]", "Wiki fragment link should be updated")
		assert.Contains(t, updatedContentStr, "[[research-updated#^block123]]", "Wiki block reference should be updated")
		assert.Contains(t, updatedContentStr, "[Research Intro](research-updated.md#Introduction)", "Markdown fragment link should be updated")
		assert.Contains(t, updatedContentStr, "[Block Reference](research-updated.md#^block123)", "Markdown block reference should be updated")
		assert.Contains(t, updatedContentStr, "![[research-updated#Introduction]]", "Embed fragment link should be updated")
		assert.Contains(t, updatedContentStr, "![[research-updated#^block123]]", "Embed block reference should be updated")

		// Verify old links are gone
		assert.NotContains(t, updatedContentStr, "[[research#Introduction]]", "Old wiki fragment link should be gone")
		assert.NotContains(t, updatedContentStr, "[Research Intro](research.md#Introduction)", "Old markdown fragment link should be gone")
		assert.NotContains(t, updatedContentStr, "![[research#Introduction]]", "Old embed fragment link should be gone")

		// Verify that links to other files with fragments were not affected
		assert.Contains(t, updatedContentStr, "[[notes/deep-dive#Theory Section]]", "Other fragment links should remain unchanged")
		assert.Contains(t, updatedContentStr, "[Deep Theory](notes/deep-dive.md#Theory%20Section)", "Other markdown fragment links should remain unchanged")

		t.Logf("Fragment link update test: %d files modified, %d links updated",
			result.FilesModified, result.LinksUpdated)
	})

	t.Run("fragment links with URL encoding", func(t *testing.T) {
		// Test URL-encoded fragment handling
		testContent := `# Test URL Encoding

- [Spaced Section](file.md#Section%20With%20Spaces)
- [Special Chars](file.md#Section%20%28with%29%20%5Bbrackets%5D)
- [Encoded Block](file.md#%5Eblock123)
`

		linkParser := processor.NewLinkParser()
		links := linkParser.Extract(testContent)

		var fragmentLinks []vault.Link
		for _, link := range links {
			if link.HasFragment() {
				fragmentLinks = append(fragmentLinks, link)
			}
		}

		require.Equal(t, 3, len(fragmentLinks), "Should find 3 fragment links with URL encoding")

		// Verify decoding
		for _, link := range fragmentLinks {
			switch {
			case strings.Contains(link.RawText, "Spaced%20Section"):
				assert.Equal(t, "Section With Spaces", link.Fragment, "Should decode spaces in fragment")
				assert.Equal(t, "url", link.Encoding, "Should detect URL encoding")

			case strings.Contains(link.RawText, "%28with%29"):
				assert.Equal(t, "Section (with) [brackets]", link.Fragment, "Should decode special characters")
				assert.Equal(t, "url", link.Encoding, "Should detect URL encoding")

			case strings.Contains(link.RawText, "%5Eblock123"):
				assert.Equal(t, "^block123", link.Fragment, "Should decode block reference marker")
				assert.True(t, link.IsBlockFragment(), "Should recognize as block fragment after decoding")
				assert.Equal(t, "url", link.Encoding, "Should detect URL encoding")
			}
		}
	})
}
