package template

import (
	"testing"
	"time"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

func TestTemplateEngine_Process(t *testing.T) {
	// Fixed time for consistent testing
	fixedTime := time.Date(2023, 1, 15, 10, 30, 0, 0, time.UTC)

	engine := NewEngine()
	engine.SetCurrentTime(fixedTime)

	file := &vault.VaultFile{
		Path:         "/vault/test-note.md",
		RelativePath: "projects/test-note.md",
		Frontmatter: map[string]interface{}{
			"title": "My Test Note",
			"tags":  []string{"project", "test"},
		},
		Modified: fixedTime,
	}

	tests := []struct {
		name     string
		template string
		want     string
	}{
		{
			name:     "current_date",
			template: "{{current_date}}",
			want:     "2023-01-15",
		},
		{
			name:     "current_datetime",
			template: "{{current_datetime}}",
			want:     "2023-01-15T10:30:00Z",
		},
		{
			name:     "filename",
			template: "{{filename}}",
			want:     "test-note",
		},
		{
			name:     "filename with filter",
			template: "{{filename|upper}}",
			want:     "TEST-NOTE",
		},
		{
			name:     "filename with slug filter",
			template: "{{filename|slug}}",
			want:     "test-note",
		},
		{
			name:     "title field",
			template: "{{title}}",
			want:     "My Test Note",
		},
		{
			name:     "title with slug filter",
			template: "{{title|slug}}",
			want:     "my-test-note",
		},
		{
			name:     "uuid generation",
			template: "{{uuid}}",
			want:     "valid-uuid", // Will validate format separately
		},
		{
			name:     "file_mtime",
			template: "{{file_mtime}}",
			want:     "2023-01-15",
		},
		{
			name:     "file_mtime with format",
			template: "{{file_mtime|date:2006-01-02}}",
			want:     "2023-01-15",
		},
		{
			name:     "relative_path",
			template: "{{relative_path}}",
			want:     "projects/test-note.md",
		},
		{
			name:     "parent_dir",
			template: "{{parent_dir}}",
			want:     "projects",
		},
		{
			name:     "complex template",
			template: "{{current_date}}-{{filename|slug}}-{{title|slug}}",
			want:     "2023-01-15-test-note-my-test-note",
		},
		{
			name:     "missing field",
			template: "{{nonexistent}}",
			want:     "",
		},
		{
			name:     "template with literal text",
			template: "Created on {{current_date}} - {{title}}",
			want:     "Created on 2023-01-15 - My Test Note",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := engine.Process(tt.template, file)

			// Special handling for UUID validation
			if tt.name == "uuid generation" {
				if len(got) != 36 || !isValidUUID(got) {
					t.Errorf("Process() UUID = %v, want valid UUID format", got)
				}
				return
			}

			if got != tt.want {
				t.Errorf("Process() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTemplateEngine_Filters(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name   string
		input  string
		filter string
		want   string
	}{
		{
			name:   "upper filter",
			input:  "hello world",
			filter: "upper",
			want:   "HELLO WORLD",
		},
		{
			name:   "lower filter",
			input:  "HELLO WORLD",
			filter: "lower",
			want:   "hello world",
		},
		{
			name:   "slug filter",
			input:  "Hello World! @#$",
			filter: "slug",
			want:   "hello-world",
		},
		{
			name:   "slug filter with special chars",
			input:  "My Awesome Note (v2)",
			filter: "slug",
			want:   "my-awesome-note-v2",
		},
		{
			name:   "date filter",
			input:  "2023-01-15T10:30:00Z",
			filter: "date:2006-01-02",
			want:   "2023-01-15",
		},
		{
			name:   "date filter with different format",
			input:  "2023-01-15T10:30:00Z",
			filter: "date:Jan 2, 2006",
			want:   "Jan 15, 2023",
		},
		{
			name:   "unknown filter",
			input:  "test",
			filter: "unknown",
			want:   "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := engine.applyFilter(tt.input, tt.filter)
			if got != tt.want {
				t.Errorf("applyFilter(%v, %v) = %v, want %v", tt.input, tt.filter, got, tt.want)
			}
		})
	}
}

// isValidUUID checks if a string is a valid UUID format
func isValidUUID(uuid string) bool {
	if len(uuid) != 36 {
		return false
	}
	// Basic format check: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	if uuid[8] != '-' || uuid[13] != '-' || uuid[18] != '-' || uuid[23] != '-' {
		return false
	}
	return true
}