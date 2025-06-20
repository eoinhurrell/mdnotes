package processor

import (
	"testing"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

func TestHeadingProcessor_Analyze(t *testing.T) {
	tests := []struct {
		name    string
		content string
		file    *vault.VaultFile
		want    HeadingAnalysis
	}{
		{
			name: "multiple H1s",
			content: `# First Title
Some content
# Second Title`,
			file: &vault.VaultFile{
				Frontmatter: map[string]interface{}{},
			},
			want: HeadingAnalysis{
				Issues: []HeadingIssue{
					{Type: "multiple_h1", Line: 3},
				},
			},
		},
		{
			name: "H1 doesn't match title",
			content: `---
title: Expected Title
---
# Different Title`,
			file: &vault.VaultFile{
				Frontmatter: map[string]interface{}{
					"title": "Expected Title",
				},
			},
			want: HeadingAnalysis{
				Issues: []HeadingIssue{
					{Type: "h1_title_mismatch", Expected: "Expected Title", Actual: "Different Title"},
				},
			},
		},
		{
			name: "skipped heading levels",
			content: `# Title
### Skipped H2
##### Skipped H3 and H4`,
			file: &vault.VaultFile{
				Frontmatter: map[string]interface{}{},
			},
			want: HeadingAnalysis{
				Issues: []HeadingIssue{
					{Type: "skipped_level", Line: 2, Expected: "H2", Actual: "H3"},
					{Type: "skipped_level", Line: 3, Expected: "H4", Actual: "H5"},
				},
			},
		},
		{
			name: "no H1 with title in frontmatter",
			content: `---
title: My Title
---
## Starting with H2`,
			file: &vault.VaultFile{
				Frontmatter: map[string]interface{}{
					"title": "My Title",
				},
			},
			want: HeadingAnalysis{
				Issues: []HeadingIssue{
					{Type: "missing_h1", Expected: "My Title"},
				},
			},
		},
		{
			name: "valid heading structure",
			content: `---
title: My Title
---
# My Title
## Section 1
### Subsection`,
			file: &vault.VaultFile{
				Frontmatter: map[string]interface{}{
					"title": "My Title",
				},
			},
			want: HeadingAnalysis{
				Issues: []HeadingIssue{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the content
			tt.file.Parse([]byte(tt.content))

			processor := NewHeadingProcessor()
			analysis := processor.Analyze(tt.file)

			if len(analysis.Issues) != len(tt.want.Issues) {
				t.Errorf("Analyze() issues count = %d, want %d", len(analysis.Issues), len(tt.want.Issues))
				t.Errorf("Got issues: %+v", analysis.Issues)
				t.Errorf("Want issues: %+v", tt.want.Issues)
				return
			}

			for i, issue := range analysis.Issues {
				if i >= len(tt.want.Issues) {
					t.Errorf("Unexpected issue: %+v", issue)
					continue
				}
				want := tt.want.Issues[i]
				if issue.Type != want.Type || issue.Line != want.Line || issue.Expected != want.Expected || issue.Actual != want.Actual {
					t.Errorf("Issue %d = %+v, want %+v", i, issue, want)
				}
			}
		})
	}
}

func TestHeadingProcessor_Fix(t *testing.T) {
	tests := []struct {
		name  string
		file  *vault.VaultFile
		rules HeadingRules
		want  string
	}{
		{
			name: "ensure H1 from title",
			file: &vault.VaultFile{
				Frontmatter: map[string]interface{}{
					"title": "My Note",
				},
				Body: "Some content without heading",
			},
			rules: HeadingRules{
				EnsureH1Title: true,
			},
			want: "# My Note\n\nSome content without heading",
		},
		{
			name: "replace wrong H1 with title",
			file: &vault.VaultFile{
				Frontmatter: map[string]interface{}{
					"title": "Correct Title",
				},
				Body: "# Wrong Title\n\nSome content",
			},
			rules: HeadingRules{
				EnsureH1Title: true,
			},
			want: "# Correct Title\n\nSome content",
		},
		{
			name: "convert multiple H1s",
			file: &vault.VaultFile{
				Frontmatter: map[string]interface{}{},
				Body:        "# First Title\nContent\n# Second Title\nMore content",
			},
			rules: HeadingRules{
				SingleH1: true,
			},
			want: "# First Title\nContent\n## Second Title\nMore content",
		},
		{
			name: "fix heading sequence",
			file: &vault.VaultFile{
				Frontmatter: map[string]interface{}{},
				Body:        "# Title\n### Skipped H2\n##### Skipped H3 and H4",
			},
			rules: HeadingRules{
				FixSequence: true,
			},
			want: "# Title\n## Skipped H2\n### Skipped H3 and H4",
		},
		{
			name: "no changes needed",
			file: &vault.VaultFile{
				Frontmatter: map[string]interface{}{
					"title": "My Title",
				},
				Body: "# My Title\n\n## Section",
			},
			rules: HeadingRules{
				EnsureH1Title: true,
				SingleH1:      true,
			},
			want: "# My Title\n\n## Section",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := NewHeadingProcessor()
			err := processor.Fix(tt.file, tt.rules)
			if err != nil {
				t.Errorf("Fix() error = %v", err)
				return
			}

			if tt.file.Body != tt.want {
				t.Errorf("Fix() body = %q, want %q", tt.file.Body, tt.want)
			}
		})
	}
}

func TestHeadingProcessor_ExtractHeadings(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []Heading
	}{
		{
			name: "various heading levels",
			content: `# H1 Title
## H2 Section
### H3 Subsection
#### H4 Detail`,
			want: []Heading{
				{Level: 1, Text: "H1 Title", Line: 1},
				{Level: 2, Text: "H2 Section", Line: 2},
				{Level: 3, Text: "H3 Subsection", Line: 3},
				{Level: 4, Text: "H4 Detail", Line: 4},
			},
		},
		{
			name: "headings with content",
			content: `Some intro text

# Main Title

Content paragraph

## Section One

More content here

### Subsection

Final content`,
			want: []Heading{
				{Level: 1, Text: "Main Title", Line: 3},
				{Level: 2, Text: "Section One", Line: 7},
				{Level: 3, Text: "Subsection", Line: 11},
			},
		},
		{
			name:    "no headings",
			content: "Just plain text\nNo headings here",
			want:    []Heading{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := NewHeadingProcessor()
			headings := processor.ExtractHeadings(tt.content)

			if len(headings) != len(tt.want) {
				t.Errorf("ExtractHeadings() count = %d, want %d", len(headings), len(tt.want))
				return
			}

			for i, heading := range headings {
				want := tt.want[i]
				if heading.Level != want.Level || heading.Text != want.Text || heading.Line != want.Line {
					t.Errorf("Heading %d = %+v, want %+v", i, heading, want)
				}
			}
		})
	}
}

func TestHeadingProcessor_Clean(t *testing.T) {
	tests := []struct {
		name      string
		file      *vault.VaultFile
		rules     CleanRules
		wantBody  string
		wantStats CleanStats
	}{
		{
			name: "replace square brackets in headings",
			file: &vault.VaultFile{
				Body: `# [X] Git Project
## [2019-11-28 04:56] Daily Log
### [URGENT] Important Task`,
			},
			rules: CleanRules{
				SquareBrackets: true,
				LinkHeaders:    false,
			},
			wantBody: `# <X> Git Project
## <2019-11-28 04:56> Daily Log
### <URGENT> Important Task`,
			wantStats: CleanStats{
				SquareBracketsFixed:  3,
				LinkHeadersConverted: 0,
			},
		},
		{
			name: "convert wiki link headers to list items",
			file: &vault.VaultFile{
				Body: `# [[Important Note]]
## Project [[Project A]] Notes
### Normal heading
#### [[Another Link]] in heading`,
			},
			rules: CleanRules{
				SquareBrackets: false,
				LinkHeaders:    true,
			},
			wantBody: `- [[Important Note]]
- Project [[Project A]] Notes
### Normal heading
- [[Another Link]] in heading`,
			wantStats: CleanStats{
				SquareBracketsFixed:  0,
				LinkHeadersConverted: 3,
			},
		},
		{
			name: "convert markdown link headers to list items",
			file: &vault.VaultFile{
				Body: `# [GitHub](https://github.com)
## Check out [this link](example.com)
### Normal heading
#### [Text](url) and more text`,
			},
			rules: CleanRules{
				SquareBrackets: false,
				LinkHeaders:    true,
			},
			wantBody: `- [GitHub](https://github.com)
- Check out [this link](example.com)
### Normal heading
- [Text](url) and more text`,
			wantStats: CleanStats{
				SquareBracketsFixed:  0,
				LinkHeadersConverted: 3,
			},
		},
		{
			name: "mixed cleaning - both rules enabled",
			file: &vault.VaultFile{
				Body: `# [TODO] [[Project Setup]]
## [DONE] Regular heading
### [[Link]] with [brackets]
#### Normal heading`,
			},
			rules: CleanRules{
				SquareBrackets: true,
				LinkHeaders:    true,
			},
			wantBody: `- [TODO] [[Project Setup]]
## <DONE> Regular heading
- [[Link]] with [brackets]
#### Normal heading`,
			wantStats: CleanStats{
				SquareBracketsFixed:  1, // Only the [DONE] in the regular heading gets fixed
				LinkHeadersConverted: 2, // Two headings with links get converted
			},
		},
		{
			name: "preserve indentation in converted headers",
			file: &vault.VaultFile{
				Body: `  # [[Indented Link Header]]
    ## Another [[Indented]] header
Normal text`,
			},
			rules: CleanRules{
				SquareBrackets: false,
				LinkHeaders:    true,
			},
			wantBody: `  - [[Indented Link Header]]
    - Another [[Indented]] header
Normal text`,
			wantStats: CleanStats{
				SquareBracketsFixed:  0,
				LinkHeadersConverted: 2,
			},
		},
		{
			name: "no changes needed",
			file: &vault.VaultFile{
				Body: `# Normal Heading
## Another Normal Heading
Regular content here`,
			},
			rules: CleanRules{
				SquareBrackets: true,
				LinkHeaders:    true,
			},
			wantBody: `# Normal Heading
## Another Normal Heading
Regular content here`,
			wantStats: CleanStats{
				SquareBracketsFixed:  0,
				LinkHeadersConverted: 0,
			},
		},
		{
			name: "both rules disabled",
			file: &vault.VaultFile{
				Body: `# [TODO] [[Important Link]]
## [DONE] Another task`,
			},
			rules: CleanRules{
				SquareBrackets: false,
				LinkHeaders:    false,
			},
			wantBody: `# [TODO] [[Important Link]]
## [DONE] Another task`,
			wantStats: CleanStats{
				SquareBracketsFixed:  0,
				LinkHeadersConverted: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := NewHeadingProcessor()
			stats, err := processor.Clean(tt.file, tt.rules)
			if err != nil {
				t.Errorf("Clean() error = %v", err)
				return
			}

			if tt.file.Body != tt.wantBody {
				t.Errorf("Clean() body = %q, want %q", tt.file.Body, tt.wantBody)
			}

			if stats.SquareBracketsFixed != tt.wantStats.SquareBracketsFixed {
				t.Errorf("Clean() SquareBracketsFixed = %d, want %d", stats.SquareBracketsFixed, tt.wantStats.SquareBracketsFixed)
			}

			if stats.LinkHeadersConverted != tt.wantStats.LinkHeadersConverted {
				t.Errorf("Clean() LinkHeadersConverted = %d, want %d", stats.LinkHeadersConverted, tt.wantStats.LinkHeadersConverted)
			}
		})
	}
}

func TestHeadingProcessor_replaceSquareBrackets(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantBody  string
		wantCount int
	}{
		{
			name:      "single square bracket replacement",
			body:      "# [X] Task heading",
			wantBody:  "# <X> Task heading",
			wantCount: 1,
		},
		{
			name:      "multiple square brackets in one heading",
			body:      "# [TODO] [URGENT] Important task",
			wantBody:  "# <TODO> <URGENT> Important task",
			wantCount: 2,
		},
		{
			name:      "date stamp in heading",
			body:      "## [2019-11-28 04:56] Meeting notes",
			wantBody:  "## <2019-11-28 04:56> Meeting notes",
			wantCount: 1,
		},
		{
			name:      "no headings with square brackets",
			body:      "# Normal heading\n## Another heading",
			wantBody:  "# Normal heading\n## Another heading",
			wantCount: 0,
		},
		{
			name:      "square brackets in non-heading text should be ignored",
			body:      "# Normal heading\nSome text with [brackets] here\n## Another heading",
			wantBody:  "# Normal heading\nSome text with [brackets] here\n## Another heading",
			wantCount: 0,
		},
		{
			name:      "wiki links should be preserved",
			body:      "# [[Important Link]]",
			wantBody:  "# [[Important Link]]",
			wantCount: 0,
		},
		{
			name:      "markdown links should be preserved",
			body:      "# [Text](url)",
			wantBody:  "# [Text](url)",
			wantCount: 0,
		},
		{
			name:      "mixed content with wiki links and square brackets",
			body:      "# [TODO] [[Important Link]] [URGENT]",
			wantBody:  "# <TODO> [[Important Link]] <URGENT>",
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := NewHeadingProcessor()
			gotBody, gotCount := processor.replaceSquareBrackets(tt.body)

			if gotBody != tt.wantBody {
				t.Errorf("replaceSquareBrackets() body = %q, want %q", gotBody, tt.wantBody)
			}

			if gotCount != tt.wantCount {
				t.Errorf("replaceSquareBrackets() count = %d, want %d", gotCount, tt.wantCount)
			}
		})
	}
}

func TestHeadingProcessor_convertLinkHeaders(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantBody  string
		wantCount int
	}{
		{
			name:      "wiki link header conversion",
			body:      "# [[Important Note]]",
			wantBody:  "- [[Important Note]]",
			wantCount: 1,
		},
		{
			name:      "markdown link header conversion",
			body:      "## [GitHub](https://github.com)",
			wantBody:  "- [GitHub](https://github.com)",
			wantCount: 1,
		},
		{
			name:      "mixed content with links",
			body:      "### Project [[Link]] and [text](url)",
			wantBody:  "- Project [[Link]] and [text](url)",
			wantCount: 1,
		},
		{
			name:      "preserve indentation",
			body:      "  ## [[Indented Header]]",
			wantBody:  "  - [[Indented Header]]",
			wantCount: 1,
		},
		{
			name:      "no link headers",
			body:      "# Normal heading\n## Another heading",
			wantBody:  "# Normal heading\n## Another heading",
			wantCount: 0,
		},
		{
			name:      "links in non-heading text should be ignored",
			body:      "# Normal heading\nSome text with [[link]] here\n## Another heading",
			wantBody:  "# Normal heading\nSome text with [[link]] here\n## Another heading",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := NewHeadingProcessor()
			gotBody, gotCount := processor.convertLinkHeaders(tt.body)

			if gotBody != tt.wantBody {
				t.Errorf("convertLinkHeaders() body = %q, want %q", gotBody, tt.wantBody)
			}

			if gotCount != tt.wantCount {
				t.Errorf("convertLinkHeaders() count = %d, want %d", gotCount, tt.wantCount)
			}
		})
	}
}
