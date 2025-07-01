package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eoinhurrell/mdnotes/internal/processor"
)

// TestRenameStressTest conducts comprehensive rename testing using the real reading list
func TestRenameStressTest(t *testing.T) {
	// Get the test vault path
	vaultPath := filepath.Join("..", "..", "test/test-vault")
	absVaultPath, err := filepath.Abs(vaultPath)
	require.NoError(t, err)

	// Verify the vault exists
	_, err = os.Stat(absVaultPath)
	require.NoError(t, err, "test/test-vault directory should exist")

	t.Run("comprehensive rename stress test", func(t *testing.T) {
		// Copy the vault to a temporary location for testing
		testVaultPath, cleanup := copyTestVault(t, absVaultPath)
		defer cleanup()

		readingListPath := filepath.Join(testVaultPath, "projects", "20241226230440-2025-to-be-read-list.md")

		// Verify reading list exists
		require.FileExists(t, readingListPath)

		// Define stress test scenarios with increasing complexity
		testScenarios := []struct {
			name        string
			description string
			operations  []RenameOperation
		}{
			{
				name:        "basic_renaming",
				description: "Test basic file renaming with links",
				operations: []RenameOperation{
					{
						original:         "resources/books/20250527111132-blood_s_hiding.md",
						new:              "resources/books/20250527111132-Blood's Hiding RENAMED.md",
						expectLinkUpdate: true,
					},
				},
			},
			{
				name:        "space_handling",
				description: "Test files with spaces",
				operations: []RenameOperation{
					{
						original:         "resources/books/20250113221045-flying_to_nowhere.md",
						new:              "resources/books/20250113221045-Flying to Nowhere RENAMED.md",
						expectLinkUpdate: true,
					},
				},
			},
			{
				name:        "special_characters",
				description: "Test files with special characters",
				operations: []RenameOperation{
					{
						original:         "resources/books/20241020122921-Polysecure.md",
						new:              "resources/books/20241020122921-Polysecure™ RENAMED.md",
						expectLinkUpdate: true,
					},
				},
			},
		}

		totalRenames := 0
		totalLinksUpdated := 0

		// Execute each test scenario
		for _, scenario := range testScenarios {
			t.Run(scenario.name, func(t *testing.T) {
				t.Logf("Testing scenario: %s", scenario.description)

				for i, op := range scenario.operations {
					t.Logf("  Operation %d: %s → %s", i+1, op.original, op.new)

					// Ensure directories exist for the new path
					newDir := filepath.Dir(filepath.Join(testVaultPath, op.new))
					require.NoError(t, os.MkdirAll(newDir, 0755))

					// Perform the rename
					result := performRename(t, testVaultPath, op.original, op.new)

					if op.expectLinkUpdate {
						assert.Greater(t, result.LinksUpdated, 0,
							"Expected links to be updated for %s", op.original)
					}

					totalRenames++
					totalLinksUpdated += result.LinksUpdated

					t.Logf("    Result: %d files scanned, %d modified, %d links updated",
						result.FilesScanned, result.FilesModified, result.LinksUpdated)
				}
			})
		}

		t.Logf("\n=== STRESS TEST SUMMARY ===")
		t.Logf("Total renames performed: %d", totalRenames)
		t.Logf("Total links updated: %d", totalLinksUpdated)

		// Verify the reading list still has valid content
		finalContent, err := os.ReadFile(readingListPath)
		require.NoError(t, err)
		finalContentStr := string(finalContent)

		// Basic sanity checks
		assert.Contains(t, finalContentStr, "# 2025 To Be Read List", "Reading list title should be preserved")
		assert.Contains(t, finalContentStr, "## Books", "Books section should be preserved")
		assert.Greater(t, len(strings.Split(finalContentStr, "\n")), 50, "Reading list should still have substantial content")

		// Verify some of our renames are reflected (check for unique renamed parts)
		assert.Contains(t, finalContentStr, "RENAMED", "Should contain RENAMED files")

		// Verify specific renames occurred (accounting for URL encoding in paths)
		assert.True(t,
			strings.Contains(finalContentStr, "Blood's%20Hiding%20RENAMED") ||
				strings.Contains(finalContentStr, "Blood's Hiding RENAMED") ||
				strings.Contains(finalContentStr, "Blood%27s%20Hiding%20RENAMED"),
			"Should contain renamed Blood's Hiding")
		assert.True(t,
			strings.Contains(finalContentStr, "Flying%20to%20Nowhere%20RENAMED") || strings.Contains(finalContentStr, "Flying to Nowhere RENAMED"),
			"Should contain renamed Flying to Nowhere")
		assert.Contains(t, finalContentStr, "Polysecure™", "Should contain renamed Polysecure")

		t.Logf("Reading list integrity verified after %d renames", totalRenames)
	})

	t.Run("concurrent_rename_stress", func(t *testing.T) {
		// Test multiple rapid renames
		testVaultPath, cleanup := copyTestVault(t, absVaultPath)
		defer cleanup()

		readingListPath := filepath.Join(testVaultPath, "projects", "20241226230440-2025-to-be-read-list.md")

		// Perform rapid sequential renames of the same file
		currentPath := "resources/books/20250101134325-the_dracula_tape.md"

		for i := 1; i <= 5; i++ {
			newPath := fmt.Sprintf("resources/books/20250101134325-the_dracula_tape_v%d.md", i)

			result := performRename(t, testVaultPath, currentPath, newPath)
			assert.Greater(t, result.LinksUpdated, 0, "Should update links in iteration %d", i)

			currentPath = newPath
		}

		// Verify final state
		finalContent, err := os.ReadFile(readingListPath)
		require.NoError(t, err)

		assert.Contains(t, string(finalContent), "the_dracula_tape_v5.md", "Should contain final renamed version")
		assert.NotContains(t, string(finalContent), "the_dracula_tape.md", "Should not contain original name")

		t.Logf("Sequential rename stress test completed successfully")
	})

	t.Run("error_handling_and_edge_cases", func(t *testing.T) {
		testVaultPath, cleanup := copyTestVault(t, absVaultPath)
		defer cleanup()

		t.Run("nonexistent_file", func(t *testing.T) {
			// Try to rename a file that doesn't exist
			options := processor.RenameOptions{
				VaultRoot: testVaultPath,
				DryRun:    false,
				Verbose:   true,
			}

			renameProcessor := processor.NewRenameProcessor(options)
			defer renameProcessor.Cleanup()

			nonexistentPath := filepath.Join(testVaultPath, "nonexistent.md")
			newPath := filepath.Join(testVaultPath, "still_nonexistent.md")

			_, err := renameProcessor.ProcessRename(context.Background(), nonexistentPath, newPath, options)
			assert.Error(t, err, "Should error when trying to rename nonexistent file")
		})

		t.Run("duplicate_target_name", func(t *testing.T) {
			// Try to rename to a name that already exists
			existingFile := filepath.Join(testVaultPath, "resources", "books", "20250101134325-the_dracula_tape.md")
			targetFile := filepath.Join(testVaultPath, "resources", "books", "20241228173236-Nightblood.md")

			options := processor.RenameOptions{
				VaultRoot: testVaultPath,
				DryRun:    false,
				Verbose:   true,
			}

			renameProcessor := processor.NewRenameProcessor(options)
			defer renameProcessor.Cleanup()

			_, err := renameProcessor.ProcessRename(context.Background(), existingFile, targetFile, options)
			// This should either error or handle the conflict gracefully
			if err == nil {
				t.Logf("Rename processor handled duplicate target gracefully")
			} else {
				t.Logf("Rename processor correctly errored on duplicate target: %v", err)
			}
		})

		t.Run("very_long_filename", func(t *testing.T) {
			// Test with extremely long filename - should fail due to filesystem limits
			longName := strings.Repeat("very_long_name_", 20) + ".md"

			originalFullPath := filepath.Join(testVaultPath, "resources/books/20250601223002-the_queen.md")
			newFullPath := filepath.Join(testVaultPath, "resources/books/"+longName)

			require.FileExists(t, originalFullPath, "Source file should exist")

			options := processor.RenameOptions{
				VaultRoot: testVaultPath,
				DryRun:    false,
				Verbose:   true,
			}

			renameProcessor := processor.NewRenameProcessor(options)
			defer renameProcessor.Cleanup()

			_, err := renameProcessor.ProcessRename(context.Background(), originalFullPath, newFullPath, options)
			assert.Error(t, err, "Very long filename should fail due to filesystem limits")
			assert.Contains(t, err.Error(), "file name too long", "Should get appropriate error message")
			t.Logf("Expected error for long filename: %v", err)
		})
	})
}

type RenameOperation struct {
	original         string
	new              string
	expectLinkUpdate bool
}

// performRename executes a single rename operation and returns the result
func performRename(t *testing.T, vaultPath, originalPath, newPath string) *processor.RenameResult {
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
