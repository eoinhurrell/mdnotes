package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/eoinhurrell/mdnotes/internal/processor"
	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// TestComplexLinkScenario tests the specific complex link scenario reported by the user
func TestComplexLinkScenario(t *testing.T) {
	// Create temporary vault structure
	tempDir, err := os.MkdirTemp("", "mdnotes_complex_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create directory structure
	resourcesDir := filepath.Join(tempDir, "resources", "books")
	projectsDir := filepath.Join(tempDir, "projects")

	if err := os.MkdirAll(resourcesDir, 0755); err != nil {
		t.Fatalf("Failed to create resources dir: %v", err)
	}
	if err := os.MkdirAll(projectsDir, 0755); err != nil {
		t.Fatalf("Failed to create projects dir: %v", err)
	}

	// Create the exact files from the user's scenario
	bigKidsPath := filepath.Join(resourcesDir, "20250525145132-Big Kids.md")
	readingListPath := filepath.Join(projectsDir, "reading-list.md")

	bigKidsContent := `# Big Kids

Book by Michael DeForge.
`

	// This is the exact complex line format that was failing
	readingListContent := `# Reading List

- [x] [Big Kids](resources/books/20250525145132-Big%20Kids.md), Michael DeForge ✅ 2025-05-16
- [ ] Another book
`

	if err := os.WriteFile(bigKidsPath, []byte(bigKidsContent), 0644); err != nil {
		t.Fatalf("Failed to create Big Kids file: %v", err)
	}

	if err := os.WriteFile(readingListPath, []byte(readingListContent), 0644); err != nil {
		t.Fatalf("Failed to create reading list file: %v", err)
	}

	t.Run("link detection works with complex line", func(t *testing.T) {
		readingListFile := &vault.VaultFile{
			Path:         readingListPath,
			RelativePath: "projects/reading-list.md",
			Body:         readingListContent,
		}

		linkParser := processor.NewLinkParser()
		linkParser.UpdateFile(readingListFile)

		if len(readingListFile.Links) != 1 {
			t.Errorf("Expected 1 link, got %d", len(readingListFile.Links))
		}

		if len(readingListFile.Links) > 0 {
			link := readingListFile.Links[0]
			// The enhanced parser correctly decodes URL-encoded targets
			expectedTarget := "resources/books/20250525145132-Big Kids.md"
			if link.Target != expectedTarget {
				t.Errorf("Expected target %q, got %q", expectedTarget, link.Target)
			}
			if link.Text != "Big Kids" {
				t.Errorf("Expected text %q, got %q", "Big Kids", link.Text)
			}
			// Verify the encoding was detected
			if link.Encoding != "url" {
				t.Errorf("Expected encoding %q, got %q", "url", link.Encoding)
			}
		}
	})

	t.Run("rename processor handles complex line with verbose logging", func(t *testing.T) {
		options := processor.RenameOptions{
			VaultRoot: tempDir,
			DryRun:    true,
			Verbose:   true,
		}

		renameProcessor := processor.NewRenameProcessor(options)
		defer renameProcessor.Cleanup()

		// Rename to something else
		newPath := filepath.Join(resourcesDir, "20250525145132-Renamed-Big-Kids.md")

		result, err := renameProcessor.ProcessRename(context.Background(), bigKidsPath, newPath, options)
		if err != nil {
			t.Fatalf("Failed to process rename: %v", err)
		}

		if result.FilesModified != 1 {
			t.Errorf("Expected 1 file to be modified, got %d", result.FilesModified)
		}

		if result.LinksUpdated != 1 {
			t.Errorf("Expected 1 link to be updated, got %d", result.LinksUpdated)
		}
	})

	t.Run("link updater handles complex line correctly", func(t *testing.T) {
		linkUpdater := processor.NewLinkUpdater()
		moves := []processor.FileMove{
			{From: "resources/books/20250525145132-Big Kids.md", To: "resources/books/20250525145132-Renamed-Big-Kids.md"},
		}

		testFile := &vault.VaultFile{
			Path:         readingListPath,
			RelativePath: "projects/reading-list.md",
			Body:         readingListContent,
		}

		linkParser := processor.NewLinkParser()
		linkParser.UpdateFile(testFile)
		modified := linkUpdater.UpdateFile(testFile, moves)

		if !modified {
			t.Error("Expected file to be modified")
		}

		expectedContent := `# Reading List

- [x] [Big Kids](resources/books/20250525145132-Renamed-Big-Kids.md), Michael DeForge ✅ 2025-05-16
- [ ] Another book
`

		if testFile.Body != expectedContent {
			t.Errorf("Content not updated correctly.\nExpected:\n%s\nGot:\n%s", expectedContent, testFile.Body)
		}
	})

	t.Run("error handling and reporting", func(t *testing.T) {
		// Test what happens during normal processing - this tests error reporting during link updates
		options := processor.RenameOptions{
			VaultRoot: tempDir,
			DryRun:    true, // Use dry-run to avoid actual file operations
			Verbose:   true,
		}

		renameProcessor := processor.NewRenameProcessor(options)
		defer renameProcessor.Cleanup()

		// Rename to a valid path (this should succeed in dry-run mode)
		newPath := filepath.Join(resourcesDir, "20250525145132-Renamed-Big-Kids.md")

		result, err := renameProcessor.ProcessRename(context.Background(), bigKidsPath, newPath, options)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Verify the operation was successful
		if result.FilesModified != 1 {
			t.Errorf("Expected 1 file to be modified, got %d", result.FilesModified)
		}

		if result.LinksUpdated != 1 {
			t.Errorf("Expected 1 link to be updated, got %d", result.LinksUpdated)
		}

		t.Logf("Successfully processed rename with %d files scanned, %d links updated", result.FilesScanned, result.LinksUpdated)
	})
}
