package processor

import (
	"testing"

	"github.com/eoinhurrell/mdnotes/internal/vault"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLinkParser_WikiLinks(t *testing.T) {
	parser := NewLinkParser()

	tests := []struct {
		name     string
		content  string
		expected []vault.Link
	}{
		{
			name:    "basic wiki link",
			content: "See [[note]] for details",
			expected: []vault.Link{
				{
					Type:     vault.WikiLink,
					Target:   "note",
					Text:     "note",
					Fragment: "",
					Alias:    "",
					RawText:  "[[note]]",
				},
			},
		},
		{
			name:    "wiki link with extension",
			content: "See [[note.md]] for details",
			expected: []vault.Link{
				{
					Type:     vault.WikiLink,
					Target:   "note.md",
					Text:     "note.md",
					Fragment: "",
					Alias:    "",
					RawText:  "[[note.md]]",
				},
			},
		},
		{
			name:    "wiki link with heading fragment",
			content: "See [[note#Section 1]] for details",
			expected: []vault.Link{
				{
					Type:     vault.WikiLink,
					Target:   "note",
					Text:     "note#Section 1",
					Fragment: "Section 1",
					Alias:    "",
					RawText:  "[[note#Section 1]]",
				},
			},
		},
		{
			name:    "wiki link with block fragment",
			content: "See [[note#^abc123]] for details",
			expected: []vault.Link{
				{
					Type:     vault.WikiLink,
					Target:   "note",
					Text:     "note#^abc123",
					Fragment: "^abc123",
					Alias:    "",
					RawText:  "[[note#^abc123]]",
				},
			},
		},
		{
			name:    "wiki link with alias",
			content: "See [[note|Custom Title]] for details",
			expected: []vault.Link{
				{
					Type:     vault.WikiLink,
					Target:   "note",
					Text:     "Custom Title",
					Fragment: "",
					Alias:    "Custom Title",
					RawText:  "[[note|Custom Title]]",
				},
			},
		},
		{
			name:    "wiki link with fragment and alias",
			content: "See [[note#heading|Custom]] for details",
			expected: []vault.Link{
				{
					Type:     vault.WikiLink,
					Target:   "note",
					Text:     "Custom",
					Fragment: "heading",
					Alias:    "Custom",
					RawText:  "[[note#heading|Custom]]",
				},
			},
		},
		{
			name:    "wiki link with path",
			content: "See [[folder/subfolder/note]] for details",
			expected: []vault.Link{
				{
					Type:     vault.WikiLink,
					Target:   "folder/subfolder/note",
					Text:     "folder/subfolder/note",
					Fragment: "",
					Alias:    "",
					RawText:  "[[folder/subfolder/note]]",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			links := parser.Extract(tt.content)
			require.Len(t, links, len(tt.expected))
			
			for i, expected := range tt.expected {
				actual := links[i]
				assert.Equal(t, expected.Type, actual.Type)
				assert.Equal(t, expected.Target, actual.Target)
				assert.Equal(t, expected.Text, actual.Text)
				assert.Equal(t, expected.Fragment, actual.Fragment)
				assert.Equal(t, expected.Alias, actual.Alias)
				assert.Equal(t, expected.RawText, actual.RawText)
			}
		})
	}
}

func TestLinkParser_MarkdownLinks(t *testing.T) {
	parser := NewLinkParser()

	tests := []struct {
		name     string
		content  string
		expected []vault.Link
	}{
		{
			name:    "basic markdown link",
			content: "See [note](note.md) for details",
			expected: []vault.Link{
				{
					Type:     vault.MarkdownLink,
					Target:   "note.md",
					Text:     "note",
					Fragment: "",
					Encoding: "none",
					RawText:  "[note](note.md)",
				},
			},
		},
		{
			name:    "markdown link with fragment",
			content: "See [Section](note.md#heading) for details",
			expected: []vault.Link{
				{
					Type:     vault.MarkdownLink,
					Target:   "note.md",
					Text:     "Section",
					Fragment: "heading",
					Encoding: "none",
					RawText:  "[Section](note.md#heading)",
				},
			},
		},
		{
			name:    "markdown link with URL encoding",
			content: "See [Note](note%20with%20spaces.md) for details",
			expected: []vault.Link{
				{
					Type:     vault.MarkdownLink,
					Target:   "note with spaces.md",
					Text:     "Note",
					Fragment: "",
					Encoding: "url",
					RawText:  "[Note](note%20with%20spaces.md)",
				},
			},
		},
		{
			name:    "markdown link with angle brackets",
			content: "See [Note](<note with spaces.md>) for details",
			expected: []vault.Link{
				{
					Type:     vault.MarkdownLink,
					Target:   "note with spaces.md",
					Text:     "Note",
					Fragment: "",
					Encoding: "angle",
					RawText:  "[Note](<note with spaces.md>)",
				},
			},
		},
		{
			name:    "markdown link with encoded fragment",
			content: "See [Note](note%20file.md#section%201) for details",
			expected: []vault.Link{
				{
					Type:     vault.MarkdownLink,
					Target:   "note file.md",
					Text:     "Note", 
					Fragment: "section 1",
					Encoding: "url",
					RawText:  "[Note](note%20file.md#section%201)",
				},
			},
		},
		{
			name:    "markdown link with parentheses in target",
			content: "See [Batman](resources/books/20250103151541-Batman%20Sword%20of%20Azrael%20(1992-).md) for details",
			expected: []vault.Link{
				{
					Type:     vault.MarkdownLink,
					Target:   "resources/books/20250103151541-Batman Sword of Azrael (1992-).md",
					Text:     "Batman",
					Fragment: "",
					Encoding: "url",
					RawText:  "[Batman](resources/books/20250103151541-Batman%20Sword%20of%20Azrael%20(1992-).md)",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			links := parser.Extract(tt.content)
			require.Len(t, links, len(tt.expected))
			
			for i, expected := range tt.expected {
				actual := links[i]
				assert.Equal(t, expected.Type, actual.Type)
				assert.Equal(t, expected.Target, actual.Target)
				assert.Equal(t, expected.Text, actual.Text)
				assert.Equal(t, expected.Fragment, actual.Fragment)
				assert.Equal(t, expected.Encoding, actual.Encoding)
				assert.Equal(t, expected.RawText, actual.RawText)
			}
		})
	}
}

func TestLinkParser_EmbedLinks(t *testing.T) {
	parser := NewLinkParser()

	tests := []struct {
		name     string
		content  string
		expected []vault.Link
	}{
		{
			name:    "basic embed",
			content: "![[image.png]]",
			expected: []vault.Link{
				{
					Type:     vault.EmbedLink,
					Target:   "image.png",
					Text:     "image.png",
					Fragment: "",
					RawText:  "![[image.png]]",
				},
			},
		},
		{
			name:    "embed with heading",
			content: "![[note#heading]]",
			expected: []vault.Link{
				{
					Type:     vault.EmbedLink,
					Target:   "note",
					Text:     "note#heading",
					Fragment: "heading",
					RawText:  "![[note#heading]]",
				},
			},
		},
		{
			name:    "embed with block reference",
			content: "![[note#^block123]]",
			expected: []vault.Link{
				{
					Type:     vault.EmbedLink,
					Target:   "note",
					Text:     "note#^block123",
					Fragment: "^block123",
					RawText:  "![[note#^block123]]",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			links := parser.Extract(tt.content)
			require.Len(t, links, len(tt.expected))
			
			for i, expected := range tt.expected {
				actual := links[i]
				assert.Equal(t, expected.Type, actual.Type)
				assert.Equal(t, expected.Target, actual.Target)
				assert.Equal(t, expected.Text, actual.Text)
				assert.Equal(t, expected.Fragment, actual.Fragment)
				assert.Equal(t, expected.RawText, actual.RawText)
			}
		})
	}
}

func TestLinkParser_ComplexContent(t *testing.T) {
	parser := NewLinkParser()
	
	content := `# Complex Document

This document has multiple link types:

1. Basic wiki links: [[note1]] and [[note2|Custom Alias]]
2. Wiki links with fragments: [[note3#heading]] and [[note4#^block123|Block Ref]]
3. Markdown links: [Regular Link](note5.md) and [Fragment Link](note6.md#section)
4. Encoded links: [Spaced](note%20with%20spaces.md) and [Angled](<another spaced note.md>)
5. Embeds: ![[image.png]] and ![[note7#section]]

External links should be ignored: [Google](https://google.com)`

	links := parser.Extract(content)
	
	// Should find 10 internal links (ignoring external URL)
	assert.Len(t, links, 10)
	
	// Check specific links
	wikiLinks := make([]vault.Link, 0)
	markdownLinks := make([]vault.Link, 0)
	embedLinks := make([]vault.Link, 0)
	
	for _, link := range links {
		switch link.Type {
		case vault.WikiLink:
			wikiLinks = append(wikiLinks, link)
		case vault.MarkdownLink:
			markdownLinks = append(markdownLinks, link)
		case vault.EmbedLink:
			embedLinks = append(embedLinks, link)
		}
	}
	
	assert.Len(t, wikiLinks, 4)    // note1, note2, note3, note4
	assert.Len(t, markdownLinks, 4) // note5, note6, spaced note, angled note  
	assert.Len(t, embedLinks, 2)    // image.png, note7
	
	// Verify fragment parsing
	fragmentLinks := make([]vault.Link, 0)
	for _, link := range links {
		if link.HasFragment() {
			fragmentLinks = append(fragmentLinks, link)
		}
	}
	assert.Len(t, fragmentLinks, 4) // note3#heading, note4#^block123, note6#section, note7#section
}

func TestLinkParser_IsInternalLink(t *testing.T) {
	parser := NewLinkParser()
	
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
			got := parser.IsInternalLink(tt.target)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLinkParser_UpdateFile(t *testing.T) {
	parser := NewLinkParser()
	
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
			parser.UpdateFile(tt.file)

			assert.Len(t, tt.file.Links, tt.wantLinks)
			if tt.wantLinks > 0 {
				assert.Equal(t, tt.firstTarget, tt.file.Links[0].Target)
			}
		})
	}
}

func TestLink_ShouldUpdate(t *testing.T) {
	tests := []struct {
		name     string
		link     vault.Link
		oldPath  string
		newPath  string
		expected bool
	}{
		{
			name: "exact path match",
			link: vault.Link{
				Type:   vault.WikiLink,
				Target: "folder/note",
			},
			oldPath:  "folder/note.md",
			newPath:  "new-folder/note.md",
			expected: true,
		},
		{
			name: "basename match for wiki link",
			link: vault.Link{
				Type:   vault.WikiLink,
				Target: "note",
			},
			oldPath:  "folder/note.md",
			newPath:  "new-folder/note.md",
			expected: true,
		},
		{
			name: "no match for different file",
			link: vault.Link{
				Type:   vault.WikiLink,
				Target: "other-note",
			},
			oldPath:  "folder/note.md",
			newPath:  "new-folder/note.md",
			expected: false,
		},
		{
			name: "markdown link exact match",
			link: vault.Link{
				Type:   vault.MarkdownLink,
				Target: "folder/note.md",
			},
			oldPath:  "folder/note.md",
			newPath:  "new-folder/note.md",
			expected: true,
		},
		{
			name: "markdown link no basename match",
			link: vault.Link{
				Type:   vault.MarkdownLink,
				Target: "note.md",
			},
			oldPath:  "folder/note.md",
			newPath:  "new-folder/note.md",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.link.ShouldUpdate(tt.oldPath, tt.newPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLink_GenerateUpdatedLink(t *testing.T) {
	tests := []struct {
		name     string
		link     vault.Link
		newPath  string
		expected string
	}{
		{
			name: "simple wiki link",
			link: vault.Link{
				Type:   vault.WikiLink,
				Target: "old-note",
				Text:   "old-note",
			},
			newPath:  "new-note.md",
			expected: "[[new-note]]",
		},
		{
			name: "wiki link with alias",
			link: vault.Link{
				Type:   vault.WikiLink,
				Target: "old-note",
				Text:   "Custom Title",
				Alias:  "Custom Title",
			},
			newPath:  "new-note.md",
			expected: "[[new-note|Custom Title]]",
		},
		{
			name: "wiki link with fragment",
			link: vault.Link{
				Type:     vault.WikiLink,
				Target:   "old-note",
				Text:     "old-note#heading",
				Fragment: "heading",
			},
			newPath:  "new-note.md",
			expected: "[[new-note#heading]]",
		},
		{
			name: "wiki link with fragment and alias",
			link: vault.Link{
				Type:     vault.WikiLink,
				Target:   "old-note",
				Text:     "Custom",
				Fragment: "heading",
				Alias:    "Custom",
			},
			newPath:  "new-note.md",
			expected: "[[new-note#heading|Custom]]",
		},
		{
			name: "markdown link",
			link: vault.Link{
				Type:     vault.MarkdownLink,
				Target:   "old-note.md",
				Text:     "Link Text",
				Encoding: "none",
			},
			newPath:  "new-note.md",
			expected: "[Link Text](new-note.md)",
		},
		{
			name: "markdown link with fragment",
			link: vault.Link{
				Type:     vault.MarkdownLink,
				Target:   "old-note.md",
				Text:     "Link Text",
				Fragment: "section",
				Encoding: "none",
			},
			newPath:  "new-note.md",
			expected: "[Link Text](new-note.md#section)",
		},
		{
			name: "markdown link with URL encoding needed",
			link: vault.Link{
				Type:     vault.MarkdownLink,
				Target:   "old note.md",
				Text:     "Link Text",
				Encoding: "url",
			},
			newPath:  "new note.md",
			expected: "[Link Text](new%20note.md)",
		},
		{
			name: "embed link",
			link: vault.Link{
				Type:   vault.EmbedLink,
				Target: "old-image.png",
			},
			newPath:  "new-image.png",
			expected: "![[new-image.png]]",
		},
		{
			name: "embed link with fragment",
			link: vault.Link{
				Type:     vault.EmbedLink,
				Target:   "old-note",
				Fragment: "section",
			},
			newPath:  "new-note.md",
			expected: "![[new-note#section]]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.link.GenerateUpdatedLink(tt.newPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLink_FragmentMethods(t *testing.T) {
	tests := []struct {
		name              string
		link              vault.Link
		hasFragment       bool
		isHeadingFragment bool
		isBlockFragment   bool
		fullTarget        string
	}{
		{
			name: "no fragment",
			link: vault.Link{
				Target:   "note",
				Fragment: "",
			},
			hasFragment:       false,
			isHeadingFragment: false,
			isBlockFragment:   false,
			fullTarget:        "note",
		},
		{
			name: "heading fragment",
			link: vault.Link{
				Target:   "note",
				Fragment: "heading",
			},
			hasFragment:       true,
			isHeadingFragment: true,
			isBlockFragment:   false,
			fullTarget:        "note#heading",
		},
		{
			name: "block fragment",
			link: vault.Link{
				Target:   "note",
				Fragment: "^block123",
			},
			hasFragment:       true,
			isHeadingFragment: false,
			isBlockFragment:   true,
			fullTarget:        "note#^block123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.hasFragment, tt.link.HasFragment())
			assert.Equal(t, tt.isHeadingFragment, tt.link.IsHeadingFragment())
			assert.Equal(t, tt.isBlockFragment, tt.link.IsBlockFragment())
			assert.Equal(t, tt.fullTarget, tt.link.FullTarget())
		})
	}
}