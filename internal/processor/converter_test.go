package processor

import (
	"testing"
)

func TestLinkConverter_Convert(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		from     LinkFormat
		to       LinkFormat
		want     string
	}{
		{
			name:    "wiki to markdown",
			content: "See [[note]] and [[folder/note|custom text]]",
			from:    WikiFormat,
			to:      MarkdownFormat,
			want:    "See [note](note.md) and [custom text](folder/note.md)",
		},
		{
			name:    "markdown to wiki",
			content: "See [text](note.md) and [](empty.md)",
			from:    MarkdownFormat,
			to:      WikiFormat,
			want:    "See [[note|text]] and [[empty]]",
		},
		{
			name:    "wiki to markdown with relative paths",
			content: "See [[../parent/note]] and [[./current/note]]",
			from:    WikiFormat,
			to:      MarkdownFormat,
			want:    "See [../parent/note](../parent/note.md) and [./current/note](./current/note.md)",
		},
		{
			name:    "preserve external links",
			content: "See [external](https://example.com) and [[internal]]",
			from:    WikiFormat,
			to:      MarkdownFormat,
			want:    "See [external](https://example.com) and [internal](internal.md)",
		},
		{
			name:    "complex mixed conversion",
			content: "Wiki [[note|alias]] and markdown [text](file.md) with embed ![[image.png]]",
			from:    WikiFormat,
			to:      MarkdownFormat,
			want:    "Wiki [alias](note.md) and markdown [text](file.md) with embed ![[image.png]]",
		},
		{
			name:    "no changes needed",
			content: "Already in [markdown](format.md)",
			from:    WikiFormat,
			to:      MarkdownFormat,
			want:    "Already in [markdown](format.md)",
		},
		{
			name:    "wiki with special characters",
			content: "[[Note with spaces and (parentheses)|Display Text]]",
			from:    WikiFormat,
			to:      MarkdownFormat,
			want:    "[Display Text](Note%20with%20spaces%20and%20(parentheses).md)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter := NewLinkConverter()
			got := converter.Convert(tt.content, tt.from, tt.to)
			if got != tt.want {
				t.Errorf("Convert() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLinkConverter_FormatLink(t *testing.T) {
	tests := []struct {
		name   string
		link   Link
		format LinkFormat
		want   string
	}{
		{
			name:   "wiki to markdown basic",
			link:   Link{Type: WikiLink, Target: "note", Text: "note"},
			format: MarkdownFormat,
			want:   "[note](note.md)",
		},
		{
			name:   "wiki to markdown with alias",
			link:   Link{Type: WikiLink, Target: "note", Text: "custom"},
			format: MarkdownFormat,
			want:   "[custom](note.md)",
		},
		{
			name:   "wiki to markdown with path",
			link:   Link{Type: WikiLink, Target: "folder/note", Text: "folder/note"},
			format: MarkdownFormat,
			want:   "[folder/note](folder/note.md)",
		},
		{
			name:   "markdown to wiki basic",
			link:   Link{Type: MarkdownLink, Target: "note.md", Text: "text"},
			format: WikiFormat,
			want:   "[[note|text]]",
		},
		{
			name:   "markdown to wiki empty text",
			link:   Link{Type: MarkdownLink, Target: "note.md", Text: ""},
			format: WikiFormat,
			want:   "[[note]]",
		},
		{
			name:   "markdown to wiki same text as target",
			link:   Link{Type: MarkdownLink, Target: "note.md", Text: "note"},
			format: WikiFormat,
			want:   "[[note]]",
		},
		{
			name:   "preserve embed format",
			link:   Link{Type: EmbedLink, Target: "image.png"},
			format: MarkdownFormat,
			want:   "![[image.png]]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter := NewLinkConverter()
			got := converter.formatLink(tt.link, tt.format)
			if got != tt.want {
				t.Errorf("formatLink() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLinkConverter_NormalizePath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"remove .md extension", "note.md", "note"},
		{"preserve path", "folder/note.md", "folder/note"},
		{"no extension", "note", "note"},
		{"multiple dots", "file.name.md", "file.name"},
		{"different extension", "image.png", "image.png"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter := NewLinkConverter()
			got := converter.normalizePath(tt.path)
			if got != tt.want {
				t.Errorf("normalizePath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestLinkConverter_EscapePath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"spaces", "note with spaces", "note%20with%20spaces"},
		{"parentheses", "note (with) parens", "note%20(with)%20parens"},
		{"normal path", "normal-note", "normal-note"},
		{"mixed", "complex path (test)", "complex%20path%20(test)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter := NewLinkConverter()
			got := converter.escapePath(tt.path)
			if got != tt.want {
				t.Errorf("escapePath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}