package processor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

func TestOrganizer_GenerateFilename(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		file    *vault.VaultFile
		want    string
	}{
		{
			name:    "simple field replacement",
			pattern: "{{id}}.md",
			file: &vault.VaultFile{
				Frontmatter: map[string]interface{}{
					"id": "12345",
				},
			},
			want: "12345.md",
		},
		{
			name:    "date formatting",
			pattern: "{{created|date:2006-01-02}}-{{title|slug}}.md",
			file: &vault.VaultFile{
				Frontmatter: map[string]interface{}{
					"created": "2023-01-15",
					"title":   "My Test Note!",
				},
			},
			want: "2023-01-15-my-test-note.md",
		},
		{
			name:    "filename without extension",
			pattern: "{{filename}}-{{priority}}",
			file: &vault.VaultFile{
				Path: "/vault/original-note.md",
				Frontmatter: map[string]interface{}{
					"priority": 5,
				},
			},
			want: "original-note-5",
		},
		{
			name:    "complex pattern with fallbacks",
			pattern: "{{date|date:2006-01-02}}-{{title|slug}}-{{id}}",
			file: &vault.VaultFile{
				Frontmatter: map[string]interface{}{
					"date":  "2023-01-15",
					"title": "Complex Note (Version 2)",
					"id":    "abc123",
				},
			},
			want: "2023-01-15-complex-note-version-2-abc123",
		},
		{
			name:    "missing fields result in empty parts",
			pattern: "{{missing}}-{{title}}.md",
			file: &vault.VaultFile{
				Frontmatter: map[string]interface{}{
					"title": "Test",
				},
			},
			want: "-Test.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			organizer := NewOrganizer()
			got := organizer.GenerateFilename(tt.pattern, tt.file)
			if got != tt.want {
				t.Errorf("GenerateFilename() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOrganizer_GenerateDirectoryPath(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		file    *vault.VaultFile
		want    string
	}{
		{
			name:    "organize by field",
			pattern: "{{category}}",
			file: &vault.VaultFile{
				Frontmatter: map[string]interface{}{
					"category": "projects",
				},
			},
			want: "projects",
		},
		{
			name:    "nested organization",
			pattern: "{{category}}/{{subcategory}}",
			file: &vault.VaultFile{
				Frontmatter: map[string]interface{}{
					"category":    "work",
					"subcategory": "meetings",
				},
			},
			want: "work/meetings",
		},
		{
			name:    "date-based organization",
			pattern: "{{created|date:2006}}/{{created|date:01-January}}",
			file: &vault.VaultFile{
				Frontmatter: map[string]interface{}{
					"created": "2023-01-15",
				},
			},
			want: "2023/01-January",
		},
		{
			name:    "mixed organization",
			pattern: "{{type}}/{{created|date:2006}}",
			file: &vault.VaultFile{
				Frontmatter: map[string]interface{}{
					"type":    "notes",
					"created": "2023-01-15",
				},
			},
			want: "notes/2023",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			organizer := NewOrganizer()
			got := organizer.GenerateDirectoryPath(tt.pattern, tt.file)
			if got != tt.want {
				t.Errorf("GenerateDirectoryPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOrganizer_RenameFile(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := t.TempDir()

	// Create source file
	sourceFile := filepath.Join(tmpDir, "original.md")
	content := []byte("---\ntitle: Test Note\nid: \"123\"\n---\n\n# Test Note\n\nContent")
	if err := os.WriteFile(sourceFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	// Create VaultFile and parse content
	file := &vault.VaultFile{
		Path:         sourceFile,
		RelativePath: "original.md",
	}
	if err := file.Parse(content); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		pattern     string
		onConflict  ConflictResolution
		wantName    string
		shouldError bool
	}{
		{
			name:       "simple rename",
			pattern:    "{{id}}-{{title|slug}}.md",
			onConflict: ConflictSkip,
			wantName:   "123-test-note.md",
		},
		{
			name:       "rename with numbering on conflict",
			pattern:    "renamed.md",
			onConflict: ConflictNumber,
			wantName:   "renamed.md", // First time should be clean
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			organizer := NewOrganizer()

			newPath, err := organizer.RenameFile(file, tt.pattern, tmpDir, tt.onConflict)

			if tt.shouldError && err == nil {
				t.Error("Expected error but got none")
			} else if !tt.shouldError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.shouldError {
				expectedPath := filepath.Join(tmpDir, tt.wantName)
				if newPath != expectedPath {
					t.Errorf("RenameFile() newPath = %q, want %q", newPath, expectedPath)
				}

				// Check file exists at new location
				if _, err := os.Stat(newPath); os.IsNotExist(err) {
					t.Errorf("File does not exist at new path: %s", newPath)
				}

				// Check old file doesn't exist (unless it's the same)
				if newPath != sourceFile {
					if _, err := os.Stat(sourceFile); !os.IsNotExist(err) {
						t.Errorf("Old file still exists at: %s", sourceFile)
					}
				}
			}

			// Update sourceFile for next test
			if !tt.shouldError && newPath != sourceFile {
				sourceFile = newPath
				file.Path = newPath
			}
		})
	}
}

func TestOrganizer_HandleConflict(t *testing.T) {
	tmpDir := t.TempDir()

	// Create existing file
	existingFile := filepath.Join(tmpDir, "existing.md")
	if err := os.WriteFile(existingFile, []byte("existing content"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		targetPath string
		resolution ConflictResolution
		want       string
	}{
		{
			name:       "no conflict",
			targetPath: filepath.Join(tmpDir, "new-file.md"),
			resolution: ConflictSkip,
			want:       filepath.Join(tmpDir, "new-file.md"),
		},
		{
			name:       "conflict with numbering",
			targetPath: existingFile,
			resolution: ConflictNumber,
			want:       filepath.Join(tmpDir, "existing-1.md"),
		},
		{
			name:       "conflict with skip",
			targetPath: existingFile,
			resolution: ConflictSkip,
			want:       "", // Should return empty string for skip
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			organizer := NewOrganizer()
			got := organizer.handleConflict(tt.targetPath, tt.resolution)
			if got != tt.want {
				t.Errorf("handleConflict() = %q, want %q", got, tt.want)
			}
		})
	}
}
