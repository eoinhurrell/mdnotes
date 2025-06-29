package processor

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

func TestExportLinkRewriter_RewriteFileContent_RemoveStrategy(t *testing.T) {
	// Create test files
	exportedFiles := []*vault.VaultFile{
		{RelativePath: "note1.md"},
		{RelativePath: "folder/note2.md"},
	}

	allVaultFiles := []*vault.VaultFile{
		{RelativePath: "note1.md"},
		{RelativePath: "folder/note2.md"},
		{RelativePath: "folder/note3.md"},
		{RelativePath: "assets/image.png"},
	}

	analyzer := NewExportLinkAnalyzer(exportedFiles, allVaultFiles)
	rewriter := NewExportLinkRewriter(analyzer, RemoveStrategy)

	tests := []struct {
		name                      string
		file                      *vault.VaultFile
		content                   string
		expectedContent           string
		expectedExternalRemoved   int
		expectedExternalConverted int
		expectedInternalUpdated   int
	}{
		{
			name: "remove external wikilinks",
			file: &vault.VaultFile{RelativePath: "note1.md"},
			content: `# Test
This links to [[nonexistent note]] and [[another missing|Custom Display]].
Also links to [[note1]] which is internal.`,
			expectedContent: `# Test
This links to nonexistent note and Custom Display.
Also links to [[note1]] which is internal.`,
			expectedExternalRemoved: 2,
		},
		{
			name: "remove external markdown links",
			file: &vault.VaultFile{RelativePath: "note1.md"},
			content: `# Test
This links to [Missing Note](missing.md) and [Custom](folder/missing.md).
Also links to [Note 1](note1.md) which is internal.`,
			expectedContent: `# Test
This links to Missing Note and Custom.
Also links to [Note 1](note1.md) which is internal.`,
			expectedExternalRemoved: 2,
		},
		{
			name: "mixed external and internal links",
			file: &vault.VaultFile{RelativePath: "folder/note2.md"},
			content: `# Test
External: [[missing]] and [Gone](gone.md)
Internal: [[note1]] and [Note 1](../note1.md)
URL: [Google](https://google.com)`,
			expectedContent: `# Test
External: missing and Gone
Internal: [[note1]] and [Note 1](../note1.md)
URL: [Google](https://google.com)`,
			expectedExternalRemoved: 2,
		},
		{
			name: "no external links",
			file: &vault.VaultFile{RelativePath: "note1.md"},
			content: `# Test
Only internal: [[note1]] and [Note 2](folder/note2.md)
And URLs: [Google](https://google.com)`,
			expectedContent: `# Test
Only internal: [[note1]] and [Note 2](folder/note2.md)
And URLs: [Google](https://google.com)`,
			expectedExternalRemoved: 0,
		},
		{
			name: "target equals display text",
			file: &vault.VaultFile{RelativePath: "note1.md"},
			content: `# Test
Link with same target and text: [[missing|missing]]`,
			expectedContent: `# Test
Link with same target and text: missing`,
			expectedExternalRemoved: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.file.Body = tt.content

			result := rewriter.RewriteFileContent(tt.file)

			assert.Equal(t, tt.expectedContent, result.RewrittenContent, "Content mismatch")
			assert.Equal(t, tt.expectedExternalRemoved, result.ExternalLinksRemoved, "External removed count mismatch")
			assert.Equal(t, tt.expectedExternalConverted, result.ExternalLinksConverted, "External converted count mismatch")
			assert.Equal(t, tt.expectedInternalUpdated, result.InternalLinksUpdated, "Internal updated count mismatch")
		})
	}
}

func TestExportLinkRewriter_RewriteFileContent_URLStrategy(t *testing.T) {
	// Create test files
	exportedFiles := []*vault.VaultFile{
		{RelativePath: "note1.md"},
	}

	allVaultFiles := []*vault.VaultFile{
		{RelativePath: "note1.md"},
		{RelativePath: "folder/note3.md"},
	}

	analyzer := NewExportLinkAnalyzer(exportedFiles, allVaultFiles)
	rewriter := NewExportLinkRewriter(analyzer, URLStrategy)

	tests := []struct {
		name                      string
		file                      *vault.VaultFile
		content                   string
		frontmatter               map[string]interface{}
		expectedContent           string
		expectedExternalRemoved   int
		expectedExternalConverted int
	}{
		{
			name: "convert with URL from frontmatter",
			file: &vault.VaultFile{
				RelativePath: "note1.md",
				Frontmatter: map[string]interface{}{
					"url": "https://example.com/missing-note",
				},
			},
			content: `# Test
External link: [[missing note]]
Another external: [Gone](gone.md)`,
			expectedContent: `# Test
External link: [missing note](https://example.com/missing-note)
Another external: [Gone](https://example.com/missing-note)`,
			expectedExternalConverted: 2,
		},
		{
			name: "fallback to remove when no URL",
			file: &vault.VaultFile{
				RelativePath: "note1.md",
				Frontmatter: map[string]interface{}{
					"title": "Test Note",
				},
			},
			content: `# Test
External link: [[missing note]]`,
			expectedContent: `# Test
External link: missing note`,
			expectedExternalRemoved: 1,
		},
		{
			name: "use different URL fields",
			file: &vault.VaultFile{
				RelativePath: "note1.md",
				Frontmatter: map[string]interface{}{
					"source": "https://source.com/page",
				},
			},
			content: `# Test
External: [[missing]]`,
			expectedContent: `# Test
External: [missing](https://source.com/page)`,
			expectedExternalConverted: 1,
		},
		{
			name: "preserve display text in URL conversion",
			file: &vault.VaultFile{
				RelativePath: "note1.md",
				Frontmatter: map[string]interface{}{
					"url": "https://example.com",
				},
			},
			content: `# Test
External: [[missing|Custom Title]]`,
			expectedContent: `# Test
External: [Custom Title](https://example.com)`,
			expectedExternalConverted: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.file.Body = tt.content
			if tt.frontmatter != nil {
				tt.file.Frontmatter = tt.frontmatter
			}

			result := rewriter.RewriteFileContent(tt.file)

			assert.Equal(t, tt.expectedContent, result.RewrittenContent, "Content mismatch")
			assert.Equal(t, tt.expectedExternalRemoved, result.ExternalLinksRemoved, "External removed count mismatch")
			assert.Equal(t, tt.expectedExternalConverted, result.ExternalLinksConverted, "External converted count mismatch")
		})
	}
}

func TestExportLinkRewriter_FindURLInFrontmatter(t *testing.T) {
	analyzer := NewExportLinkAnalyzer([]*vault.VaultFile{}, []*vault.VaultFile{})
	rewriter := NewExportLinkRewriter(analyzer, URLStrategy)

	tests := []struct {
		name        string
		frontmatter map[string]interface{}
		target      string
		expected    string
	}{
		{
			name: "url field",
			frontmatter: map[string]interface{}{
				"url": "https://example.com",
			},
			target:   "missing",
			expected: "https://example.com",
		},
		{
			name: "source field",
			frontmatter: map[string]interface{}{
				"source": "https://source.com/page",
			},
			target:   "missing",
			expected: "https://source.com/page",
		},
		{
			name: "link field",
			frontmatter: map[string]interface{}{
				"link": "https://link.com",
			},
			target:   "missing",
			expected: "https://link.com",
		},
		{
			name: "website field",
			frontmatter: map[string]interface{}{
				"website": "https://website.com",
			},
			target:   "missing",
			expected: "https://website.com",
		},
		{
			name: "no URL field",
			frontmatter: map[string]interface{}{
				"title": "Test",
			},
			target:   "missing",
			expected: "",
		},
		{
			name: "non-string URL",
			frontmatter: map[string]interface{}{
				"url": 123,
			},
			target:   "missing",
			expected: "",
		},
		{
			name: "non-http URL",
			frontmatter: map[string]interface{}{
				"url": "file:///local/path",
			},
			target:   "missing",
			expected: "",
		},
		{
			name:        "nil frontmatter",
			frontmatter: nil,
			target:      "missing",
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := &vault.VaultFile{
				Frontmatter: tt.frontmatter,
			}

			result := rewriter.findURLInFrontmatter(tt.target, file)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExportLinkRewriter_ExtractLinkText(t *testing.T) {
	analyzer := NewExportLinkAnalyzer([]*vault.VaultFile{}, []*vault.VaultFile{})
	rewriter := NewExportLinkRewriter(analyzer, RemoveStrategy)

	tests := []struct {
		name     string
		content  string
		link     vault.Link
		expected string
	}{
		{
			name:    "wiki link",
			content: "This has [[link text]] in it",
			link: vault.Link{
				Position: vault.Position{Start: 9, End: 22},
			},
			expected: "[[link text]]",
		},
		{
			name:    "markdown link",
			content: "This has [text](url) in it",
			link: vault.Link{
				Position: vault.Position{Start: 9, End: 20},
			},
			expected: "[text](url)",
		},
		{
			name:    "invalid position",
			content: "Short text",
			link: vault.Link{
				Position: vault.Position{Start: 20, End: 30},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rewriter.extractLinkText(tt.content, tt.link)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetRewriteStrategies(t *testing.T) {
	strategies := GetRewriteStrategies()
	expected := []LinkRewriteStrategy{RemoveStrategy, URLStrategy}
	assert.Equal(t, expected, strategies)
}

func TestIsValidStrategy(t *testing.T) {
	tests := []struct {
		strategy string
		expected bool
	}{
		{"remove", true},
		{"url", true},
		{"invalid", false},
		{"", false},
		{"REMOVE", false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.strategy, func(t *testing.T) {
			result := IsValidStrategy(tt.strategy)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLinkRewriteResult(t *testing.T) {
	// Test that LinkRewriteResult structure works as expected
	result := &LinkRewriteResult{
		OriginalContent:        "original",
		RewrittenContent:       "rewritten",
		ExternalLinksRemoved:   1,
		ExternalLinksConverted: 2,
		InternalLinksUpdated:   3,
		ChangedLinks: []LinkChange{
			{
				OriginalText: "[[old]]",
				NewText:      "old",
				LinkType:     vault.WikiLink,
				Category:     ExternalLink,
				WasConverted: false,
			},
		},
	}

	assert.Equal(t, "original", result.OriginalContent)
	assert.Equal(t, "rewritten", result.RewrittenContent)
	assert.Equal(t, 1, result.ExternalLinksRemoved)
	assert.Equal(t, 2, result.ExternalLinksConverted)
	assert.Equal(t, 3, result.InternalLinksUpdated)
	assert.Len(t, result.ChangedLinks, 1)
	assert.Equal(t, "[[old]]", result.ChangedLinks[0].OriginalText)
	assert.Equal(t, "old", result.ChangedLinks[0].NewText)
}
