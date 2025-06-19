package processor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

func TestFrontmatterSync_SyncField(t *testing.T) {
	// Create a temporary file for testing
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-note.md")
	content := []byte("# Test Note\n\nContent")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	// Get file info for modification time
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatal(err)
	}

	now := info.ModTime()

	tests := []struct {
		name   string
		field  string
		source string
		file   *vault.VaultFile
		want   interface{}
	}{
		{
			name:   "sync from file modification time",
			field:  "modified",
			source: "file-mtime",
			file: &vault.VaultFile{
				Path:        testFile,
				Modified:    now,
				Frontmatter: map[string]interface{}{},
			},
			want: now.Format("2006-01-02"),
		},
		{
			name:   "sync from filename",
			field:  "id",
			source: "filename",
			file: &vault.VaultFile{
				Path:        "/vault/20230101-test-note.md",
				Frontmatter: map[string]interface{}{},
			},
			want: "20230101-test-note",
		},
		{
			name:   "sync from filename with pattern extraction",
			field:  "date",
			source: "filename:pattern:^(\\d{8})",
			file: &vault.VaultFile{
				Path:        "/vault/20230101-test-note.md",
				Frontmatter: map[string]interface{}{},
			},
			want: "20230101",
		},
		{
			name:   "sync from relative path",
			field:  "category",
			source: "path:dir",
			file: &vault.VaultFile{
				Path:         "/vault/projects/work/note.md",
				RelativePath: "projects/work/note.md",
				Frontmatter:  map[string]interface{}{},
			},
			want: "work",
		},
		{
			name:   "don't overwrite existing field",
			field:  "title",
			source: "filename",
			file: &vault.VaultFile{
				Path: "/vault/test-note.md",
				Frontmatter: map[string]interface{}{
					"title": "Existing Title",
				},
			},
			want: "Existing Title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sync := NewFrontmatterSync()
			modified := sync.SyncField(tt.file, tt.field, tt.source)

			got, exists := tt.file.GetField(tt.field)
			if !exists {
				t.Errorf("Field %s not found after sync", tt.field)
				return
			}

			if got != tt.want {
				t.Errorf("SyncField() result = %v, want %v", got, tt.want)
			}

			// Check if modification was reported correctly
			expectedModified := tt.name != "don't overwrite existing field"
			if modified != expectedModified {
				t.Errorf("SyncField() modified = %v, want %v", modified, expectedModified)
			}
		})
	}
}

func TestFrontmatterSync_ParseSource(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		wantType   string
		wantConfig string
	}{
		{
			name:       "simple source",
			source:     "file-mtime",
			wantType:   "file-mtime",
			wantConfig: "",
		},
		{
			name:       "source with config",
			source:     "filename:pattern:^(\\d{8})",
			wantType:   "filename",
			wantConfig: "pattern:^(\\d{8})",
		},
		{
			name:       "path source with config",
			source:     "path:dir",
			wantType:   "path",
			wantConfig: "dir",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sync := NewFrontmatterSync()
			gotType, gotConfig := sync.parseSource(tt.source)

			if gotType != tt.wantType {
				t.Errorf("parseSource() type = %v, want %v", gotType, tt.wantType)
			}
			if gotConfig != tt.wantConfig {
				t.Errorf("parseSource() config = %v, want %v", gotConfig, tt.wantConfig)
			}
		})
	}
}

func TestFrontmatterSync_ExtractFromFilename(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		pattern string
		want    string
	}{
		{
			name:    "extract date from filename",
			path:    "/vault/20230101-test-note.md",
			pattern: "^(\\d{8})",
			want:    "20230101",
		},
		{
			name:    "extract uuid from filename",
			path:    "/vault/abc-123-def-456-note.md",
			pattern: "([a-f0-9-]{36})",
			want:    "",
		},
		{
			name:    "no match",
			path:    "/vault/simple-note.md",
			pattern: "^(\\d{8})",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sync := NewFrontmatterSync()
			got := sync.extractFromFilename(tt.path, tt.pattern)
			if got != tt.want {
				t.Errorf("extractFromFilename() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFrontmatterSync_GetDirectoryFromPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "get immediate parent directory",
			path: "projects/work/note.md",
			want: "work",
		},
		{
			name: "file in root",
			path: "note.md",
			want: "",
		},
		{
			name: "nested path",
			path: "a/b/c/note.md",
			want: "c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sync := NewFrontmatterSync()
			got := sync.getDirectoryFromPath(tt.path)
			if got != tt.want {
				t.Errorf("getDirectoryFromPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
