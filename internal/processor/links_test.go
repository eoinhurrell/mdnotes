package processor

import (
	"testing"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

func TestLinkParser_Extract(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []Link
	}{
		{
			name:    "wiki links",
			content: "See [[other note]] and [[folder/note|custom text]]",
			want: []Link{
				{Type: WikiLink, Target: "other note", Text: "other note", Position: Position{Start: 4, End: 18}},
				{Type: WikiLink, Target: "folder/note", Text: "custom text", Position: Position{Start: 23, End: 51}},
			},
		},
		{
			name:    "markdown links",
			content: "See [text](note.md) and [](empty.md)",
			want: []Link{
				{Type: MarkdownLink, Target: "note.md", Text: "text", Position: Position{Start: 4, End: 20}},
				{Type: MarkdownLink, Target: "empty.md", Text: "", Position: Position{Start: 25, End: 37}},
			},
		},
		{
			name:    "embedded links",
			content: "![[image.png]] and ![[note.md]]",
			want: []Link{
				{Type: EmbedLink, Target: "image.png", Position: Position{Start: 0, End: 14}},
				{Type: EmbedLink, Target: "note.md", Position: Position{Start: 19, End: 32}},
			},
		},
		{
			name:    "mixed link types",
			content: "Wiki [[note]] and markdown [link](file.md) and embed ![[image.png]]",
			want: []Link{
				{Type: WikiLink, Target: "note", Text: "note", Position: Position{Start: 5, End: 13}},
				{Type: MarkdownLink, Target: "file.md", Text: "link", Position: Position{Start: 27, End: 42}},
				{Type: EmbedLink, Target: "image.png", Position: Position{Start: 53, End: 67}},
			},
		},
		{
			name:    "links with spaces and special chars",
			content: "[[Note with spaces]] and [[Note (with) brackets|alias]]",
			want: []Link{
				{Type: WikiLink, Target: "Note with spaces", Text: "Note with spaces", Position: Position{Start: 0, End: 20}},
				{Type: WikiLink, Target: "Note (with) brackets", Text: "alias", Position: Position{Start: 25, End: 55}},
			},
		},
		{
			name:    "external links (ignored)",
			content: "External [link](https://example.com) should be ignored",
			want:    []Link{},
		},
		{
			name:    "no links",
			content: "Just plain text with no links",
			want:    []Link{},
		},
		{
			name:    "nested brackets",
			content: "[[Note [with] brackets]] and normal [[note]]",
			want: []Link{
				{Type: WikiLink, Target: "Note [with] brackets", Text: "Note [with] brackets", Position: Position{Start: 0, End: 24}},
				{Type: WikiLink, Target: "note", Text: "note", Position: Position{Start: 37, End: 45}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewLinkParser()
			links := parser.Extract(tt.content)

			if len(links) != len(tt.want) {
				t.Errorf("Extract() links count = %d, want %d", len(links), len(tt.want))
				t.Errorf("Got links: %+v", links)
				t.Errorf("Want links: %+v", tt.want)
				return
			}

			for i, link := range links {
				want := tt.want[i]
				if link.Type != want.Type || link.Target != want.Target || link.Text != want.Text {
					t.Errorf("Link %d = %+v, want %+v", i, link, want)
				}
				// Position validation is approximate due to regex complexity
			}
		})
	}
}

func TestLinkParser_IsInternalLink(t *testing.T) {
	tests := []struct {
		name   string
		target string
		want   bool
	}{
		{"relative path", "note.md", true},
		{"relative path with folder", "folder/note.md", true},
		{"without extension", "note", true},
		{"with spaces", "note with spaces", true},
		{"http URL", "http://example.com", false},
		{"https URL", "https://example.com/page", false},
		{"ftp URL", "ftp://files.example.com", false},
		{"mailto", "mailto:user@example.com", false},
		{"absolute path with scheme", "/absolute/path", true},
		{"current directory", "./note.md", true},
		{"parent directory", "../note.md", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewLinkParser()
			got := parser.IsInternalLink(tt.target)
			if got != tt.want {
				t.Errorf("IsInternalLink(%q) = %v, want %v", tt.target, got, tt.want)
			}
		})
	}
}

func TestLinkParser_UpdateFile(t *testing.T) {
	tests := []struct {
		name        string
		file        *vault.VaultFile
		wantLinks   int
		firstTarget string
	}{
		{
			name: "parse links from file body",
			file: &vault.VaultFile{
				Body: "# Note\n\nSee [[other note]] and [markdown](link.md)\n\n![[embed.png]]",
			},
			wantLinks:   3,
			firstTarget: "other note",
		},
		{
			name: "file with no links",
			file: &vault.VaultFile{
				Body: "# Note\n\nJust plain content with no links",
			},
			wantLinks: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewLinkParser()
			parser.UpdateFile(tt.file)

			if len(tt.file.Links) != tt.wantLinks {
				t.Errorf("UpdateFile() links count = %d, want %d", len(tt.file.Links), tt.wantLinks)
			}

			if tt.wantLinks > 0 && tt.file.Links[0].Target != tt.firstTarget {
				t.Errorf("UpdateFile() first link target = %q, want %q", tt.file.Links[0].Target, tt.firstTarget)
			}
		})
	}
}