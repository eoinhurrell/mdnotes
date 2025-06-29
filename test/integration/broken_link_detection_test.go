package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/eoinhurrell/mdnotes/internal/processor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBrokenLinkDetection validates that the rename system correctly handles broken vs valid links
func TestBrokenLinkDetection(t *testing.T) {
	// Get the test vault path
	vaultPath := filepath.Join("..", "..", "test-vault")
	absVaultPath, err := filepath.Abs(vaultPath)
	require.NoError(t, err)

	t.Run("distinguish_valid_vs_broken_links", func(t *testing.T) {
		// Copy the vault to a temporary location for testing
		testVaultPath, cleanup := copyTestVault(t, absVaultPath)
		defer cleanup()

		readingListPath := filepath.Join(testVaultPath, "projects", "20241226230440-2025-to-be-read-list.md")

		t.Logf("=== BROKEN LINK DETECTION TEST ===")

		// Analyze the current state
		content, err := os.ReadFile(readingListPath)
		require.NoError(t, err)
		contentStr := string(content)

		// Check what files actually exist
		bigKidsFile := filepath.Join(testVaultPath, "resources", "books", "20250525145132-big_kids.md")
		bigKidsLinkFile := filepath.Join(testVaultPath, "resources", "books", "20250525145132-Big Kids.md")
		bloodFile := filepath.Join(testVaultPath, "resources", "books", "20250527111132-blood_s_hiding.md")

		bigKidsExists := fileExists(bigKidsFile)
		bigKidsLinkExists := fileExists(bigKidsLinkFile)
		bloodExists := fileExists(bloodFile)

		t.Logf("File existence check:")
		t.Logf("  big_kids.md (actual): %t", bigKidsExists)
		t.Logf("  Big Kids.md (link target): %t", bigKidsLinkExists)
		t.Logf("  blood_s_hiding.md: %t", bloodExists)

		// Test 1: Broken link scenario (Big Kids)
		t.Run("broken_link_not_updated", func(t *testing.T) {
			// This link points to a file that doesn't exist
			// Link: [Big Kids](resources/books/20250525145132-Big%20Kids.md)
			// Actual file: 20250525145132-big_kids.md

			assert.True(t, bigKidsExists, "Actual file should exist")
			assert.False(t, bigKidsLinkExists, "Link target should NOT exist (broken link)")
			assert.Contains(t, contentStr, "Big%20Kids.md", "Should contain broken link")

			// Attempt to rename the actual file
			originalPath := "resources/books/20250525145132-big_kids.md"
			newPath := "resources/books/20250525145132-big_kids_RENAMED.md"

			result := performRenameOperation(t, testVaultPath, originalPath, newPath)

			// Should NOT update the broken link
			assert.Equal(t, 0, result.LinksUpdated, "Broken links should NOT be updated")

			// Verify the broken link remains unchanged
			updatedContent, err := os.ReadFile(readingListPath)
			require.NoError(t, err)
			updatedStr := string(updatedContent)

			assert.Contains(t, updatedStr, "Big%20Kids.md", "Broken link should remain unchanged")
			assert.NotContains(t, updatedStr, "big_kids_RENAMED", "Should not contain new filename")

			t.Logf("✅ Broken link correctly NOT updated")
		})

		// Test 2: Valid link scenario (Blood's Hiding)
		t.Run("valid_link_updated", func(t *testing.T) {
			// This link points to a file that exists
			// Link: [Blood's Hiding](resources/books/20250527111132-blood_s_hiding.md)
			// Actual file: 20250527111132-blood_s_hiding.md

			assert.True(t, bloodExists, "Blood file should exist")
			assert.Contains(t, contentStr, "blood_s_hiding.md", "Should contain valid link")

			// Rename the actual file
			originalPath := "resources/books/20250527111132-blood_s_hiding.md"
			newPath := "resources/books/20250527111132-blood_s_hiding_RENAMED.md"

			result := performRenameOperation(t, testVaultPath, originalPath, newPath)

			// Should update the valid link
			assert.Greater(t, result.LinksUpdated, 0, "Valid links should be updated")

			// Verify the link was updated
			updatedContent, err := os.ReadFile(readingListPath)
			require.NoError(t, err)
			updatedStr := string(updatedContent)

			assert.Contains(t, updatedStr, "blood_s_hiding_RENAMED", "Should contain new filename")
			assert.NotContains(t, updatedStr, "[Blood's Hiding](resources/books/20250527111132-blood_s_hiding.md)", "Old link should be gone")

			t.Logf("✅ Valid link correctly updated")
		})
	})

	t.Run("create_and_test_correct_links", func(t *testing.T) {
		// Create a test that demonstrates the correct way to handle this
		testVaultPath, cleanup := copyTestVault(t, absVaultPath)
		defer cleanup()

		// Create a test document with correct links
		testDoc := filepath.Join(testVaultPath, "test-correct-links.md")
		testContent := `# Test Correct Links

## Valid Links (point to actual files)
- [Blood's Hiding - Valid](resources/books/20250527111132-blood_s_hiding.md)
- [Nightblood - Valid](resources/books/20241228173236-Nightblood.md)

## Broken Links (point to non-existent files)  
- [Big Kids - Broken](resources/books/20250525145132-Big%20Kids.md)
- [Fragments - Broken](resources/books/20250103125238-Fragments%20of%20Horror.md)

## What the links SHOULD be
- [Big Kids - Correct](resources/books/20250525145132-big_kids.md)
- [Fragments - Correct](resources/books/20250103125238-Fragments of Horror.md)
`
		require.NoError(t, os.WriteFile(testDoc, []byte(testContent), 0644))

		// Test renaming blood_s_hiding file
		result := performRenameOperation(t, testVaultPath,
			"resources/books/20250527111132-blood_s_hiding.md",
			"resources/books/20250527111132-blood_hiding_fixed.md")

		// Only the valid links should be updated (may be multiple files)
		assert.Greater(t, result.LinksUpdated, 0, "Should update valid links")
		assert.Equal(t, 2, result.LinksUpdated, "Should update exactly 2 valid links (test file + reading list)")

		// Verify the results
		updatedContent, err := os.ReadFile(testDoc)
		require.NoError(t, err)
		updatedStr := string(updatedContent)

		// Valid link should be updated
		assert.Contains(t, updatedStr, "blood_hiding_fixed.md", "Valid link should be updated")

		// Broken links should remain unchanged
		assert.Contains(t, updatedStr, "Big%20Kids.md", "Broken link should remain")
		assert.Contains(t, updatedStr, "Fragments%20of%20Horror.md", "Broken link should remain")

		// Correct links should remain unchanged
		assert.Contains(t, updatedStr, "big_kids.md", "Correct link should remain")
		assert.Contains(t, updatedStr, "Fragments of Horror.md", "Correct link should remain")

		t.Logf("✅ Correct link behavior validated")
	})
}

// performRenameOperation executes a single rename operation
func performRenameOperation(t *testing.T, vaultPath, originalPath, newPath string) *processor.RenameResult {
	originalFullPath := filepath.Join(vaultPath, originalPath)
	newFullPath := filepath.Join(vaultPath, newPath)

	// Verify source file exists
	require.FileExists(t, originalFullPath, "Source file should exist: %s", originalPath)

	options := processor.RenameOptions{
		VaultRoot: vaultPath,
		DryRun:    false,
		Verbose:   true,
	}

	renameProcessor := processor.NewRenameProcessor(options)
	defer renameProcessor.Cleanup()

	result, err := renameProcessor.ProcessRename(context.Background(), originalFullPath, newFullPath, options)
	require.NoError(t, err, "Rename should succeed")

	// Verify the rename was successful
	assert.FileExists(t, newFullPath, "Target file should exist after rename")
	assert.NoFileExists(t, originalFullPath, "Source file should not exist after rename")

	return result
}
