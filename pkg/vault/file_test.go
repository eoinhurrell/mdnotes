package vault

import (
	"reflect"
	"testing"
)

func TestVaultFile_Parse(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    *VaultFile
		wantErr bool
	}{
		{
			name: "valid markdown with frontmatter",
			content: `---
title: Test Note
tags: [test, example]
created: "2023-01-01"
---

# Test Note

Content here.`,
			want: &VaultFile{
				Frontmatter: map[string]interface{}{
					"title":   "Test Note",
					"tags":    []interface{}{"test", "example"},
					"created": "2023-01-01",
				},
				Body: "# Test Note\n\nContent here.",
			},
		},
		{
			name:    "markdown without frontmatter",
			content: "# Just Content\n\nNo frontmatter here.",
			want: &VaultFile{
				Frontmatter: map[string]interface{}{},
				Body:        "# Just Content\n\nNo frontmatter here.",
			},
		},
		{
			name: "empty frontmatter",
			content: `---
---

# Content`,
			want: &VaultFile{
				Frontmatter: map[string]interface{}{},
				Body:        "# Content",
			},
		},
		{
			name: "frontmatter with complex types",
			content: `---
title: Complex Note
tags:
  - tag1
  - tag2
metadata:
  created: "2023-01-01"
  published: true
  priority: 5
---

Content here.`,
			want: &VaultFile{
				Frontmatter: map[string]interface{}{
					"title": "Complex Note",
					"tags":  []interface{}{"tag1", "tag2"},
					"metadata": map[string]interface{}{
						"created":   "2023-01-01",
						"published": true,
						"priority":  5,
					},
				},
				Body: "Content here.",
			},
		},
		{
			name: "invalid frontmatter yaml",
			content: `---
title: Test
invalid: [unclosed
---

Content`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vf := &VaultFile{}
			err := vf.Parse([]byte(tt.content))
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(vf.Frontmatter, tt.want.Frontmatter) {
				t.Errorf("Frontmatter = %v, want %v", vf.Frontmatter, tt.want.Frontmatter)
			}
			if vf.Body != tt.want.Body {
				t.Errorf("Body = %v, want %v", vf.Body, tt.want.Body)
			}
		})
	}
}

func TestVaultFile_Serialize(t *testing.T) {
	tests := []struct {
		name string
		file *VaultFile
		want string
	}{
		{
			name: "file with frontmatter",
			file: &VaultFile{
				Frontmatter: map[string]interface{}{
					"title": "Test Note",
					"tags":  []string{"test", "example"},
				},
				Body: "# Test Note\n\nContent here.",
			},
			want: `---
tags:
    - test
    - example
title: Test Note
---

# Test Note

Content here.`,
		},
		{
			name: "file without frontmatter",
			file: &VaultFile{
				Frontmatter: map[string]interface{}{},
				Body:        "# Just Content",
			},
			want: "# Just Content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.file.Serialize()
			if err != nil {
				t.Errorf("Serialize() error = %v", err)
				return
			}
			if string(got) != tt.want {
				t.Errorf("Serialize() = %v, want %v", string(got), tt.want)
			}
		})
	}
}
