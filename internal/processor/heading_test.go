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
