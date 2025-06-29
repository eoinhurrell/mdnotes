package processor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

func TestExportLinkAnalyzer_AnalyzeFile(t *testing.T) {
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

	tests := []struct {
		name               string
		file               *vault.VaultFile
		content            string
		expectedInternal   int
		expectedExternal   int
		expectedAssets     int
		expectedURLs       int
		expectedCategories []LinkCategory
	}{
		{
			name: "internal wikilinks",
			file: &vault.VaultFile{RelativePath: "note1.md"},
			content: `# Test
[[note1]] - self reference
[[folder/note2]] - exported file
`,
			expectedInternal:   2,
			expectedExternal:   0,
			expectedAssets:     0,
			expectedURLs:       0,
			expectedCategories: []LinkCategory{InternalLink, InternalLink},
		},
		{
			name: "external wikilinks",
			file: &vault.VaultFile{RelativePath: "note1.md"},
			content: `# Test
[[folder/note3]] - not exported
[[nonexistent]] - doesn't exist
`,
			expectedInternal:   0,
			expectedExternal:   2,
			expectedAssets:     0,
			expectedURLs:       0,
			expectedCategories: []LinkCategory{ExternalLink, ExternalLink},
		},
		{
			name: "asset links",
			file: &vault.VaultFile{RelativePath: "note1.md"},
			content: `# Test
![[assets/image.png]]
![Alt text](assets/image.png)
`,
			expectedInternal:   0,
			expectedExternal:   0,
			expectedAssets:     2,
			expectedURLs:       0,
			expectedCategories: []LinkCategory{AssetLink, AssetLink},
		},
		{
			name: "URL links",
			file: &vault.VaultFile{RelativePath: "note1.md"},
			content: `# Test
[Google](https://google.com)
[Email](mailto:test@example.com)
`,
			expectedInternal:   0,
			expectedExternal:   0,
			expectedAssets:     0,
			expectedURLs:       2,
			expectedCategories: []LinkCategory{URLLink, URLLink},
		},
		{
			name: "mixed links",
			file: &vault.VaultFile{RelativePath: "folder/note2.md"},
			content: `# Test
[[note1]] - internal (in export)
[[note3]] - external (not in export)
![[../assets/image.png]] - asset
[Google](https://google.com) - URL
[Relative note](note1.md) - internal markdown link
`,
			expectedInternal:   2,
			expectedExternal:   1,
			expectedAssets:     1,
			expectedURLs:       1,
			expectedCategories: []LinkCategory{InternalLink, ExternalLink, AssetLink, URLLink, InternalLink},
		},
		{
			name: "markdown links with extensions",
			file: &vault.VaultFile{RelativePath: "note1.md"},
			content: `# Test
[Note 1](note1.md) - self reference
[Note 2](folder/note2.md) - exported file
[Note 3](folder/note3.md) - not exported
`,
			expectedInternal:   2,
			expectedExternal:   1,
			expectedAssets:     0,
			expectedURLs:       0,
			expectedCategories: []LinkCategory{InternalLink, InternalLink, ExternalLink},
		},
		{
			name: "relative paths from subfolder",
			file: &vault.VaultFile{RelativePath: "folder/note2.md"},
			content: `# Test
[[../note1]] - up one level (exported)
[[note3]] - same folder (not exported)
[Up level](../note1.md) - markdown link up
`,
			expectedInternal:   2,
			expectedExternal:   1,
			expectedAssets:     0,
			expectedURLs:       0,
			expectedCategories: []LinkCategory{InternalLink, ExternalLink, InternalLink},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the body content
			tt.file.Body = tt.content

			// Analyze the file
			analysis := analyzer.AnalyzeFile(tt.file)

			// Check counters
			assert.Equal(t, tt.expectedInternal, analysis.InternalCount, "Internal count mismatch")
			assert.Equal(t, tt.expectedExternal, analysis.ExternalCount, "External count mismatch")
			assert.Equal(t, tt.expectedAssets, analysis.AssetCount, "Asset count mismatch")
			assert.Equal(t, tt.expectedURLs, analysis.URLCount, "URL count mismatch")

			// Check individual link categories
			require.Len(t, analysis.Links, len(tt.expectedCategories), "Number of links mismatch")
			for i, expectedCategory := range tt.expectedCategories {
				assert.Equal(t, expectedCategory, analysis.Links[i].Category,
					"Link %d category mismatch: expected %v, got %v", i, expectedCategory, analysis.Links[i].Category)
			}
		})
	}
}

func TestExportLinkAnalyzer_ResolveTargetPath(t *testing.T) {
	analyzer := NewExportLinkAnalyzer([]*vault.VaultFile{}, []*vault.VaultFile{
		{RelativePath: "note1.md"},
		{RelativePath: "folder/note2.md"},
		{RelativePath: "deep/nested/note3.md"},
	})

	tests := []struct {
		name               string
		target             string
		sourceRelativePath string
		expected           string
	}{
		{
			name:               "wiki link same directory",
			target:             "note2",
			sourceRelativePath: "folder/note1.md",
			expected:           "folder/note2.md",
		},
		{
			name:               "wiki link with extension",
			target:             "note2.md",
			sourceRelativePath: "folder/note1.md",
			expected:           "folder/note2.md",
		},
		{
			name:               "relative path up one level",
			target:             "../note1",
			sourceRelativePath: "folder/note2.md",
			expected:           "note1.md",
		},
		{
			name:               "relative path with extension",
			target:             "../note1.md",
			sourceRelativePath: "folder/note2.md",
			expected:           "note1.md",
		},
		{
			name:               "absolute path from root",
			target:             "/deep/nested/note3",
			sourceRelativePath: "folder/note2.md",
			expected:           "deep/nested/note3.md",
		},
		{
			name:               "deep relative path",
			target:             "../../folder/note2",
			sourceRelativePath: "deep/nested/note3.md",
			expected:           "folder/note2.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.resolveTargetPath(tt.target, tt.sourceRelativePath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExportLinkAnalyzer_IsAssetFile(t *testing.T) {
	analyzer := NewExportLinkAnalyzer([]*vault.VaultFile{}, []*vault.VaultFile{})

	tests := []struct {
		path     string
		expected bool
	}{
		{"image.png", true},
		{"image.jpg", true},
		{"image.jpeg", true},
		{"document.pdf", true},
		{"spreadsheet.xlsx", true},
		{"note.md", false},
		{"README.txt", true},
		{"video.mp4", true},
		{"archive.zip", true},
		{"no-extension", false},
		{"folder/image.PNG", true}, // case insensitive
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := analyzer.isAssetFile(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLinkAnalysis_GetLinksByCategory(t *testing.T) {
	analysis := &LinkAnalysis{
		Links: []AnalyzedLink{
			{Category: InternalLink},
			{Category: ExternalLink},
			{Category: InternalLink},
			{Category: AssetLink},
			{Category: URLLink},
		},
	}

	internal := analysis.GetLinksByCategory(InternalLink)
	assert.Len(t, internal, 2)

	external := analysis.GetLinksByCategory(ExternalLink)
	assert.Len(t, external, 1)

	assets := analysis.GetLinksByCategory(AssetLink)
	assert.Len(t, assets, 1)

	urls := analysis.GetLinksByCategory(URLLink)
	assert.Len(t, urls, 1)
}

func TestLinkAnalysis_HasMethods(t *testing.T) {
	tests := []struct {
		name        string
		analysis    *LinkAnalysis
		hasExternal bool
		hasAssets   bool
	}{
		{
			name: "has external links",
			analysis: &LinkAnalysis{
				ExternalCount: 2,
				AssetCount:    0,
			},
			hasExternal: true,
			hasAssets:   false,
		},
		{
			name: "has assets",
			analysis: &LinkAnalysis{
				ExternalCount: 0,
				AssetCount:    1,
			},
			hasExternal: false,
			hasAssets:   true,
		},
		{
			name: "has both",
			analysis: &LinkAnalysis{
				ExternalCount: 1,
				AssetCount:    1,
			},
			hasExternal: true,
			hasAssets:   true,
		},
		{
			name: "has neither",
			analysis: &LinkAnalysis{
				ExternalCount: 0,
				AssetCount:    0,
			},
			hasExternal: false,
			hasAssets:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.hasExternal, tt.analysis.HasExternalLinks())
			assert.Equal(t, tt.hasAssets, tt.analysis.HasAssets())
		})
	}
}

func TestLinkAnalysis_Summary(t *testing.T) {
	tests := []struct {
		name     string
		analysis *LinkAnalysis
		expected string
	}{
		{
			name: "no links",
			analysis: &LinkAnalysis{
				Links: []AnalyzedLink{},
			},
			expected: "No links found",
		},
		{
			name: "only internal",
			analysis: &LinkAnalysis{
				Links:         make([]AnalyzedLink, 3),
				InternalCount: 3,
			},
			expected: "3 links (3 internal)",
		},
		{
			name: "mixed types",
			analysis: &LinkAnalysis{
				Links:         make([]AnalyzedLink, 7),
				InternalCount: 2,
				ExternalCount: 1,
				AssetCount:    3,
				URLCount:      1,
			},
			expected: "7 links (2 internal, 1 external, 3 assets, 1 URLs)",
		},
		{
			name: "only external and assets",
			analysis: &LinkAnalysis{
				Links:         make([]AnalyzedLink, 4),
				ExternalCount: 2,
				AssetCount:    2,
			},
			expected: "4 links (2 external, 2 assets)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.analysis.Summary()
			assert.Equal(t, tt.expected, result)
		})
	}
}
