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

// TestRealVaultLinkUpdating tests the rename functionality with the actual test-vault
func TestRealVaultLinkUpdating(t *testing.T) {
	// Get the test vault path
	vaultPath := filepath.Join("..", "..", "test-vault")
	absVaultPath, err := filepath.Abs(vaultPath)
	require.NoError(t, err)

	// Verify the vault exists
	_, err = os.Stat(absVaultPath)
	require.NoError(t, err, "test-vault directory should exist")

	t.Run("rename book file and verify link updates", func(t *testing.T) {
		// Copy the vault to a temporary location for testing
		testVaultPath, cleanup := copyTestVault(t, absVaultPath)
		defer cleanup()

		// Original book file path
		originalBookPath := filepath.Join(testVaultPath, "resources", "books", "20250525145132-big_kids.md")
		readingListPath := filepath.Join(testVaultPath, "projects", "20241226230440-2025-to-be-read-list.md")

		// Verify files exist
		require.FileExists(t, originalBookPath)
		require.FileExists(t, readingListPath)

		// Read original reading list content
		originalContent, err := os.ReadFile(readingListPath)
		require.NoError(t, err)

		// Verify the original broken link exists
		originalContentStr := string(originalContent)
		assert.Contains(t, originalContentStr, "[Big Kids](resources/books/20250525145132-Big%20Kids.md)")

		// Parse the link to verify our enhanced parser works
		linkParser := processor.NewLinkParser()
		links := linkParser.Extract(originalContentStr)

		var bigKidsLinks []vault.Link
		for _, link := range links {
			if strings.Contains(link.Target, "20250525145132") || strings.Contains(link.Target, "Big Kids") {
				bigKidsLinks = append(bigKidsLinks, link)
			}
		}
		require.Greater(t, len(bigKidsLinks), 0, "Should find at least one link to Big Kids")

		// Test the enhanced link parsing
		for _, link := range bigKidsLinks {
			t.Logf("Found link: Type=%d, Target=%q, Text=%q, Encoding=%q",
				link.Type, link.Target, link.Text, link.Encoding)

			if link.Type == vault.MarkdownLink {
				// Verify URL encoding detection
				if strings.Contains(link.RawText, "%20") {
					assert.Equal(t, "url", link.Encoding, "Should detect URL encoding")
				}
				// Verify target decoding
				assert.Contains(t, link.Target, "Big Kids", "Target should be URL decoded")
			}
		}

		// Create rename processor
		options := processor.RenameOptions{
			VaultRoot: testVaultPath,
			DryRun:    false,
			Verbose:   true,
		}

		renameProcessor := processor.NewRenameProcessor(options)
		defer renameProcessor.Cleanup()

		// New book file path
		newBookPath := filepath.Join(testVaultPath, "resources", "books", "20250525145132-big-kids-renamed.md")

		// Perform the rename
		result, err := renameProcessor.ProcessRename(context.Background(), originalBookPath, newBookPath, options)
		require.NoError(t, err)

		// Verify the rename was successful
		assert.FileExists(t, newBookPath)
		assert.NoFileExists(t, originalBookPath)

		// IMPORTANT: This is a BROKEN LINK scenario!
		// The link points to "Big%20Kids.md" but the actual file is "big_kids.md"
		// The rename system should correctly NOT update broken links
		assert.Equal(t, 0, result.LinksUpdated, "Broken links should NOT be updated")

		t.Logf("Rename results: %d files scanned, %d files modified, %d links updated",
			result.FilesScanned, result.FilesModified, result.LinksUpdated)

		// Read updated reading list content
		updatedContent, err := os.ReadFile(readingListPath)
		require.NoError(t, err)
		updatedContentStr := string(updatedContent)

		// Verify the broken link was NOT updated (correct behavior)
		assert.Contains(t, updatedContentStr, "20250525145132-Big%20Kids.md",
			"Broken link should remain unchanged")
		assert.NotContains(t, updatedContentStr, "big-kids-renamed.md",
			"Broken link should not be 'fixed' automatically")

		// Test that other links are unchanged
		assert.Contains(t, updatedContentStr, "[Fragments of Horror](resources/books/20250103125238-Fragments%20of%20Horror.md)")
		assert.Contains(t, updatedContentStr, "[The Dracula Tape](resources/books/20250101134325-the_dracula_tape.md)")
	})

	t.Run("rename blood's hiding book with special characters", func(t *testing.T) {
		// Copy the vault to a temporary location for testing
		testVaultPath, cleanup := copyTestVault(t, absVaultPath)
		defer cleanup()

		// Original book file path (note the apostrophe)
		originalBookPath := filepath.Join(testVaultPath, "resources", "books", "20250527111132-blood_s_hiding.md")
		readingListPath := filepath.Join(testVaultPath, "projects", "20241226230440-2025-to-be-read-list.md")

		// Verify files exist
		require.FileExists(t, originalBookPath)
		require.FileExists(t, readingListPath)

		// Read original reading list content
		originalContent, err := os.ReadFile(readingListPath)
		require.NoError(t, err)
		originalContentStr := string(originalContent)

		// Verify the original link exists
		assert.Contains(t, originalContentStr, "[Blood's Hiding](resources/books/20250527111132-blood_s_hiding.md)")

		// Create rename processor
		options := processor.RenameOptions{
			VaultRoot: testVaultPath,
			DryRun:    false,
			Verbose:   true,
		}

		renameProcessor := processor.NewRenameProcessor(options)
		defer renameProcessor.Cleanup()

		// New book file path
		newBookPath := filepath.Join(testVaultPath, "resources", "books", "20250527111132-blood-hiding-renamed.md")

		// Perform the rename
		result, err := renameProcessor.ProcessRename(context.Background(), originalBookPath, newBookPath, options)
		require.NoError(t, err)

		// Verify the rename was successful
		assert.FileExists(t, newBookPath)
		assert.NoFileExists(t, originalBookPath)

		// Verify at least one link was updated
		assert.Greater(t, result.LinksUpdated, 0, "Should have updated at least one link")

		// Read updated reading list content
		updatedContent, err := os.ReadFile(readingListPath)
		require.NoError(t, err)
		updatedContentStr := string(updatedContent)

		// Verify the link was updated
		assert.Contains(t, updatedContentStr, "blood-hiding-renamed.md",
			"Reading list should contain updated filename")
		assert.NotContains(t, updatedContentStr, "blood_s_hiding.md",
			"Reading list should not contain old filename")

		t.Logf("Blood's Hiding rename: %d files modified, %d links updated",
			result.FilesModified, result.LinksUpdated)
	})

	t.Run("test link parsing with various encoding patterns", func(t *testing.T) {
		// Test the enhanced link parser on real vault content
		readingListPath := filepath.Join(absVaultPath, "projects", "20241226230440-2025-to-be-read-list.md")

		content, err := os.ReadFile(readingListPath)
		require.NoError(t, err)

		linkParser := processor.NewLinkParser()
		links := linkParser.Extract(string(content))

		// Analyze the different link patterns found
		var urlEncodedLinks, normalLinks, fragmentLinks []vault.Link

		for _, link := range links {
			if link.Type == vault.MarkdownLink && strings.HasPrefix(link.Target, "resources/books/") {
				if link.Encoding == "url" {
					urlEncodedLinks = append(urlEncodedLinks, link)
				} else {
					normalLinks = append(normalLinks, link)
				}

				if link.HasFragment() {
					fragmentLinks = append(fragmentLinks, link)
				}
			}
		}

		t.Logf("Found %d total links, %d book links (%d URL-encoded, %d normal, %d with fragments)",
			len(links), len(urlEncodedLinks)+len(normalLinks), len(urlEncodedLinks), len(normalLinks), len(fragmentLinks))

		// Verify we found URL-encoded links
		assert.Greater(t, len(urlEncodedLinks), 0, "Should find URL-encoded links in the real vault")

		// Test specific patterns
		for _, link := range urlEncodedLinks {
			assert.Equal(t, "url", link.Encoding, "URL-encoded links should be detected")
			assert.NotContains(t, link.Target, "%", "Target should be decoded")
			assert.Contains(t, link.RawText, "%", "Raw text should preserve encoding")
		}
	})
}

// copyTestVault creates a temporary copy of the test vault for testing
func copyTestVault(t *testing.T, originalVaultPath string) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "mdnotes_real_vault_test_*")
	require.NoError(t, err)

	// Copy the vault structure
	err = copyDir(originalVaultPath, tmpDir)
	require.NoError(t, err)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate the destination path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Copy file
		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return err
		}

		dstFile, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		// Copy the content
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		_, err = dstFile.Write(data)
		return err
	})
}
