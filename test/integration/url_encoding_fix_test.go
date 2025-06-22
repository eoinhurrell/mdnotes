package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/eoinhurrell/mdnotes/internal/processor"
	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// TestURLEncodingFix tests the specific bug reported by the user
func TestURLEncodingFix(t *testing.T) {
	// Create temporary vault structure
	tempDir, err := os.MkdirTemp("", "mdnotes_url_test")
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
	bloodsHidingPath := filepath.Join(resourcesDir, "20250527111132-Blood's Hiding.md")
	readingListPath := filepath.Join(projectsDir, "20241226230440-2025-to-be-read-list.md")

	bloodsHidingContent := `# Blood's Hiding

Book by Ken Baumann.
`

	readingListContent := `# 2025 To Be Read List

- [/] [Blood's Hiding](resources/books/20250527111132-Blood's%20Hiding.md), Ken Baumann
- [ ] Another book
`

	if err := os.WriteFile(bloodsHidingPath, []byte(bloodsHidingContent), 0644); err != nil {
		t.Fatalf("Failed to create Blood's Hiding file: %v", err)
	}

	if err := os.WriteFile(readingListPath, []byte(readingListContent), 0644); err != nil {
		t.Fatalf("Failed to create reading list file: %v", err)
	}

	t.Run("link detection works with URL encoding", func(t *testing.T) {
		readingListFile := &vault.VaultFile{
			Path:         readingListPath,
			RelativePath: "projects/20241226230440-2025-to-be-read-list.md",
			Body:         readingListContent,
		}
		
		linkParser := processor.NewLinkParser()
		linkParser.UpdateFile(readingListFile)
		
		if len(readingListFile.Links) != 1 {
			t.Errorf("Expected 1 link, got %d", len(readingListFile.Links))
		}
		
		if len(readingListFile.Links) > 0 {
			link := readingListFile.Links[0]
			expectedTarget := "resources/books/20250527111132-Blood's%20Hiding.md"
			if link.Target != expectedTarget {
				t.Errorf("Expected target %q, got %q", expectedTarget, link.Target)
			}
			if link.Text != "Blood's Hiding" {
				t.Errorf("Expected text %q, got %q", "Blood's Hiding", link.Text)
			}
		}
	})

	t.Run("rename processor finds and updates URL-encoded links", func(t *testing.T) {
		options := processor.RenameOptions{
			VaultRoot: tempDir,
			DryRun:    true,
			Verbose:   false,
		}

		renameProcessor := processor.NewRenameProcessor(options)
		defer renameProcessor.Cleanup()

		// Rename to something else
		newPath := filepath.Join(resourcesDir, "20250527111132-Renamed-Book.md")
		
		result, err := renameProcessor.ProcessRename(context.Background(), bloodsHidingPath, newPath, options)
		if err != nil {
			t.Fatalf("Failed to process rename: %v", err)
		}

		if result.FilesModified != 1 {
			t.Errorf("Expected 1 file to be modified, got %d", result.FilesModified)
		}

		if result.LinksUpdated != 1 {
			t.Errorf("Expected 1 link to be updated, got %d", result.LinksUpdated)
		}

		expectedModifiedFile := "projects/20241226230440-2025-to-be-read-list.md"
		if len(result.ModifiedFiles) != 1 || result.ModifiedFiles[0] != expectedModifiedFile {
			t.Errorf("Expected modified file %q, got %v", expectedModifiedFile, result.ModifiedFiles)
		}
	})

	t.Run("link updater handles URL encoding correctly", func(t *testing.T) {
		linkUpdater := processor.NewLinkUpdater()
		moves := []processor.FileMove{
			{From: "resources/books/20250527111132-Blood's Hiding.md", To: "resources/books/20250527111132-Renamed-Book.md"},
		}
		
		testFile := &vault.VaultFile{
			Path:         readingListPath,
			RelativePath: "projects/20241226230440-2025-to-be-read-list.md",
			Body:         readingListContent,
		}
		
		linkParser := processor.NewLinkParser()
		linkParser.UpdateFile(testFile)
		modified := linkUpdater.UpdateFile(testFile, moves)
		
		if !modified {
			t.Error("Expected file to be modified")
		}

		expectedContent := `# 2025 To Be Read List

- [/] [Blood's Hiding](resources/books/20250527111132-Renamed-Book.md), Ken Baumann
- [ ] Another book
`

		if testFile.Body != expectedContent {
			t.Errorf("Content not updated correctly.\nExpected:\n%s\nGot:\n%s", expectedContent, testFile.Body)
		}
	})

	t.Run("URL encoding preserved when needed", func(t *testing.T) {
		// Test that when renaming to a file with spaces, URL encoding is applied
		linkUpdater := processor.NewLinkUpdater()
		moves := []processor.FileMove{
			{From: "resources/books/20250527111132-Blood's Hiding.md", To: "resources/books/20250527111132-New File Name.md"},
		}
		
		testFile := &vault.VaultFile{
			Body: `- [Blood's Hiding](resources/books/20250527111132-Blood's%20Hiding.md)`,
		}
		
		linkParser := processor.NewLinkParser()
		linkParser.UpdateFile(testFile)
		modified := linkUpdater.UpdateFile(testFile, moves)
		
		if !modified {
			t.Error("Expected file to be modified")
		}

		expectedContent := `- [Blood's Hiding](resources/books/20250527111132-New%20File%20Name.md)`
		if testFile.Body != expectedContent {
			t.Errorf("Content not updated correctly.\nExpected:\n%s\nGot:\n%s", expectedContent, testFile.Body)
		}
	})
}