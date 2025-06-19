package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func createTestVault(t *testing.T, dir string) {
	// Create test files
	files := map[string]string{
		"note1.md":              "# Note 1\n\nContent",
		"note2.md":              "---\ntitle: Note 2\n---\n\n# Note 2",
		"subdir/note3.md":       "# Note 3\n\nIn subdirectory",
		"file.txt":              "Not a markdown file",
		".obsidian/app.json":    `{"theme": "dark"}`,
		"temp.tmp":              "Temporary file",
		"templates/template.md": "# Template\n\nTemplate content",
	}

	for path, content := range files {
		fullPath := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestScanner_Walk(t *testing.T) {
	tmpDir := t.TempDir()
	createTestVault(t, tmpDir)

	scanner := NewScanner()
	files, err := scanner.Walk(tmpDir)

	if err != nil {
		t.Fatalf("Walk() error = %v", err)
	}

	if len(files) != 4 { // note1.md, note2.md, subdir/note3.md, templates/template.md
		t.Errorf("Expected 4 markdown files, got %d", len(files))
	}

	// Check that all files have .md extension
	for _, file := range files {
		if !strings.HasSuffix(file.Path, ".md") {
			t.Errorf("Non-markdown file found: %s", file.Path)
		}
	}

	// Check that relative paths are set correctly
	for _, file := range files {
		if file.RelativePath == "" {
			t.Errorf("RelativePath not set for file: %s", file.Path)
		}
	}
}

func TestScanner_WithIgnorePatterns(t *testing.T) {
	tmpDir := t.TempDir()
	createTestVault(t, tmpDir)

	scanner := NewScanner(
		WithIgnorePatterns([]string{".obsidian/*", "*.tmp", "templates/*"}),
	)

	files, err := scanner.Walk(tmpDir)
	if err != nil {
		t.Fatalf("Walk() error = %v", err)
	}

	// Should only find note1.md, note2.md, subdir/note3.md (templates/* excluded)
	if len(files) != 3 {
		t.Errorf("Expected 3 files after ignoring patterns, got %d", len(files))
	}

	// Verify no ignored files are included
	for _, file := range files {
		if strings.HasPrefix(file.RelativePath, ".obsidian") {
			t.Errorf("Ignored .obsidian file found: %s", file.RelativePath)
		}
		if strings.HasPrefix(file.RelativePath, "templates") {
			t.Errorf("Ignored templates file found: %s", file.RelativePath)
		}
		if filepath.Ext(file.Path) == ".tmp" {
			t.Errorf("Ignored .tmp file found: %s", file.Path)
		}
	}
}

func TestScanner_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	scanner := NewScanner()
	files, err := scanner.Walk(tmpDir)

	if err != nil {
		t.Fatalf("Walk() error = %v", err)
	}

	if len(files) != 0 {
		t.Errorf("Expected 0 files in empty directory, got %d", len(files))
	}
}

func TestScanner_NonexistentDirectory(t *testing.T) {
	scanner := NewScanner()
	_, err := scanner.Walk("/nonexistent/directory")

	if err == nil {
		t.Error("Expected error for nonexistent directory, got nil")
	}
}
