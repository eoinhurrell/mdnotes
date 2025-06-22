package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/eoinhurrell/mdnotes/internal/processor"
	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// TestVaultRelativeLinks tests that markdown links are properly handled as vault-relative paths
func TestVaultRelativeLinks(t *testing.T) {
	// Create temporary vault structure
	tempDir, err := os.MkdirTemp("", "mdnotes_vault_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create directory structure
	resourcesDir := filepath.Join(tempDir, "resources")
	docsDir := filepath.Join(tempDir, "docs", "guides")
	
	if err := os.MkdirAll(resourcesDir, 0755); err != nil {
		t.Fatalf("Failed to create resources dir: %v", err)
	}
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatalf("Failed to create docs dir: %v", err)
	}

	// Create test files
	testFiles := map[string]string{
		"index.md": `# Index

This file contains various link types:
- Wiki link to resource: [[resources/test]]
- Markdown link to resource: [Test Resource](resources/test.md)
- Wiki link to guide: [[docs/guides/setup]]
- Markdown link to guide: [Setup Guide](docs/guides/setup.md)
`,
		"resources/test.md": `# Test Resource

This is a test resource file.
`,
		"docs/guides/setup.md": `# Setup Guide

This is a setup guide.
`,
		"other.md": `# Other File

References:
- [Link to resource](resources/test.md)
- [[docs/guides/setup|Setup]]
`,
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tempDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", path, err)
		}
	}

	t.Run("links check finds all links correctly", func(t *testing.T) {
		// Scan files
		scanner := vault.NewScanner()
		files, err := scanner.Walk(tempDir)
		if err != nil {
			t.Fatalf("Failed to scan vault: %v", err)
		}

		// Check that all expected files are found
		expectedFiles := []string{"index.md", "resources/test.md", "docs/guides/setup.md", "other.md"}
		if len(files) != len(expectedFiles) {
			t.Errorf("Expected %d files, got %d", len(expectedFiles), len(files))
		}

		// Parse links in all files
		linkParser := processor.NewLinkParser()
		for _, file := range files {
			linkParser.UpdateFile(file)
		}

		// Verify specific files have the expected links
		var indexFile *vault.VaultFile
		for _, file := range files {
			if file.RelativePath == "index.md" {
				indexFile = file
				break
			}
		}

		if indexFile == nil {
			t.Fatal("index.md not found")
		}

		// Index file should have 4 links
		if len(indexFile.Links) != 4 {
			t.Errorf("Expected 4 links in index.md, got %d", len(indexFile.Links))
			for i, link := range indexFile.Links {
				t.Logf("Link %d: %s (%d)", i, link.Target, link.Type)
			}
		}
	})

	t.Run("rename updates all vault-relative links correctly", func(t *testing.T) {
		// Test renaming resources/test.md to resources/renamed-test.md
		sourceFile := filepath.Join(tempDir, "resources", "test.md")
		targetFile := filepath.Join(tempDir, "resources", "renamed-test.md")

		// Create rename processor
		options := processor.RenameOptions{
			VaultRoot: tempDir,
			DryRun:    true, // Don't actually rename for this test
			Verbose:   true,
		}

		renameProcessor := processor.NewRenameProcessor(options)
		defer renameProcessor.Cleanup()

		// Perform rename operation
		result, err := renameProcessor.ProcessRename(context.Background(), sourceFile, targetFile, options)
		if err != nil {
			t.Fatalf("Failed to process rename: %v", err)
		}

		// Verify that links were found and would be updated
		if result.FilesModified < 2 {
			t.Errorf("Expected at least 2 files to be modified, got %d", result.FilesModified)
		}

		if result.LinksUpdated < 3 {
			t.Errorf("Expected at least 3 links to be updated, got %d", result.LinksUpdated)
		}

		t.Logf("Rename would update %d links in %d files", result.LinksUpdated, result.FilesModified)
	})

	t.Run("link updater handles vault-relative paths", func(t *testing.T) {
		// Create a test file with various link types
		testContent := `# Test File

Links to test:
- [[resources/test]]
- [Resource](resources/test.md)
- [[docs/guides/setup]]
- [Guide](docs/guides/setup.md)
`

		file := &vault.VaultFile{
			Body:         testContent,
			RelativePath: "test-file.md",
		}

		// Parse links
		linkParser := processor.NewLinkParser()
		linkParser.UpdateFile(file)

		// Create file moves
		moves := []processor.FileMove{
			{From: "resources/test.md", To: "resources/renamed-test.md"},
			{From: "docs/guides/setup.md", To: "documentation/setup.md"},
		}

		// Update links
		linkUpdater := processor.NewLinkUpdater()
		modified := linkUpdater.UpdateFile(file, moves)

		if !modified {
			t.Error("Expected file to be modified")
		}

		expectedContent := `# Test File

Links to test:
- [[resources/renamed-test]]
- [Resource](resources/renamed-test.md)
- [[documentation/setup]]
- [Guide](documentation/setup.md)
`

		if file.Body != expectedContent {
			t.Errorf("Content not updated correctly.\nExpected:\n%s\nGot:\n%s", expectedContent, file.Body)
		}
	})
}