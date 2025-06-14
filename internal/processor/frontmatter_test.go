package processor

import (
	"testing"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

func TestFrontmatterProcessor_Ensure(t *testing.T) {
	tests := []struct {
		name     string
		file     *vault.VaultFile
		field    string
		defValue interface{}
		want     interface{}
		modified bool
	}{
		{
			name: "add missing field",
			file: &vault.VaultFile{
				Frontmatter: map[string]interface{}{
					"title": "Test",
				},
			},
			field:    "tags",
			defValue: []string{},
			want:     []string{},
			modified: true,
		},
		{
			name: "preserve existing field",
			file: &vault.VaultFile{
				Frontmatter: map[string]interface{}{
					"tags": []string{"existing"},
				},
			},
			field:    "tags",
			defValue: []string{},
			want:     []string{"existing"},
			modified: false,
		},
		{
			name: "add field to empty frontmatter",
			file: &vault.VaultFile{
				Frontmatter: map[string]interface{}{},
			},
			field:    "created",
			defValue: "2023-01-01",
			want:     "2023-01-01",
			modified: true,
		},
		{
			name: "nil frontmatter",
			file: &vault.VaultFile{
				Frontmatter: nil,
			},
			field:    "title",
			defValue: "New Title",
			want:     "New Title",
			modified: true,
		},
		{
			name: "preserve nil value if it exists",
			file: &vault.VaultFile{
				Frontmatter: map[string]interface{}{
					"optional": nil,
				},
			},
			field:    "optional",
			defValue: "default",
			want:     nil,
			modified: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewFrontmatterProcessor()
			modified := p.Ensure(tt.file, tt.field, tt.defValue)

			if modified != tt.modified {
				t.Errorf("Ensure() modified = %v, want %v", modified, tt.modified)
			}

			got, exists := tt.file.GetField(tt.field)
			if !exists {
				t.Errorf("Field %s not found after Ensure()", tt.field)
			}

			// Handle slice comparison
			if gotSlice, ok := got.([]string); ok {
				if wantSlice, ok := tt.want.([]string); ok {
					if len(gotSlice) != len(wantSlice) {
						t.Errorf("Ensure() field value = %v, want %v", got, tt.want)
						return
					}
					for i, v := range gotSlice {
						if i >= len(wantSlice) || v != wantSlice[i] {
							t.Errorf("Ensure() field value = %v, want %v", got, tt.want)
							return
						}
					}
					return
				}
			}

			if got != tt.want {
				t.Errorf("Ensure() field value = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFrontmatterProcessor_EnsureWithTemplate(t *testing.T) {
	tests := []struct {
		name     string
		file     *vault.VaultFile
		field    string
		template string
		want     string
	}{
		{
			name: "simple template variable",
			file: &vault.VaultFile{
				Path:        "/vault/test-note.md",
				Frontmatter: map[string]interface{}{},
			},
			field:    "id",
			template: "{{filename}}",
			want:     "test-note",
		},
		{
			name: "multiple template variables",
			file: &vault.VaultFile{
				Path: "/vault/my-note.md",
				Frontmatter: map[string]interface{}{
					"title": "My Note",
				},
			},
			field:    "slug",
			template: "{{filename}}-{{title}}",
			want:     "my-note-My Note",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewFrontmatterProcessor()
			p.Ensure(tt.file, tt.field, tt.template)

			got, exists := tt.file.GetField(tt.field)
			if !exists {
				t.Errorf("Field %s not found after template processing", tt.field)
			}

			if got != tt.want {
				t.Errorf("Template result = %v, want %v", got, tt.want)
			}
		})
	}
}