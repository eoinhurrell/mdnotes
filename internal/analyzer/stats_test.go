package analyzer

import (
	"testing"
	"time"

	"github.com/eoinhurrell/mdnotes/internal/vault"
	"github.com/stretchr/testify/assert"
)

func TestAnalyzer_GenerateStats(t *testing.T) {
	vault := createTestVault(t)
	analyzer := NewAnalyzer()

	stats := analyzer.GenerateStats(vault.Files)

	assert.Equal(t, 4, stats.TotalFiles)
	assert.Equal(t, 3, stats.FilesWithFrontmatter)
	assert.Equal(t, 1, stats.FilesWithoutFrontmatter)
	assert.Contains(t, stats.TagDistribution, "project")
	assert.Contains(t, stats.TagDistribution, "important")
	assert.Equal(t, 2, stats.TagDistribution["project"])
	assert.Equal(t, 1, stats.TagDistribution["important"])
	assert.Greater(t, stats.TotalLinks, 0)
	assert.Greater(t, stats.TotalHeadings, 0)
	assert.NotZero(t, stats.AverageFileSize)
	assert.NotZero(t, stats.TotalSize)
}

func TestAnalyzer_FindDuplicates(t *testing.T) {
	analyzer := NewAnalyzer()

	files := []*vault.VaultFile{
		{
			Path: "a.md",
			Frontmatter: map[string]interface{}{
				"title": "Same Title",
				"id":    "unique1",
			},
		},
		{
			Path: "b.md",
			Frontmatter: map[string]interface{}{
				"title": "Same Title",
				"id":    "unique2",
			},
		},
		{
			Path: "c.md",
			Frontmatter: map[string]interface{}{
				"title": "Different Title",
				"id":    "unique3",
			},
		},
		{
			Path: "d.md",
			Frontmatter: map[string]interface{}{
				"id": "unique1", // Duplicate ID
			},
		},
	}

	// Test title duplicates
	titleDuplicates := analyzer.FindDuplicates(files, "title")
	assert.Len(t, titleDuplicates, 1)
	assert.Equal(t, "Same Title", titleDuplicates[0].Value)
	assert.Len(t, titleDuplicates[0].Files, 2)
	assert.Contains(t, titleDuplicates[0].Files, "a.md")
	assert.Contains(t, titleDuplicates[0].Files, "b.md")

	// Test ID duplicates
	idDuplicates := analyzer.FindDuplicates(files, "id")
	assert.Len(t, idDuplicates, 1)
	assert.Equal(t, "unique1", idDuplicates[0].Value)
	assert.Len(t, idDuplicates[0].Files, 2)
	assert.Contains(t, idDuplicates[0].Files, "a.md")
	assert.Contains(t, idDuplicates[0].Files, "d.md")

	// Test field that doesn't exist
	nonExistentDuplicates := analyzer.FindDuplicates(files, "nonexistent")
	assert.Len(t, nonExistentDuplicates, 0)
}

func TestAnalyzer_FindContentDuplicates(t *testing.T) {
	analyzer := NewAnalyzer()

	files := []*vault.VaultFile{
		{
			Path: "original.md",
			Body: "# Title\n\nThis is some content",
		},
		{
			Path: "duplicate.md",
			Body: "# Title\n\nThis is some content",
		},
		{
			Path: "similar.md",
			Body: "# Title\n\nThis is some content with extra text",
		},
		{
			Path: "different.md",
			Body: "# Different\n\nCompletely different content",
		},
	}

	// Test exact content duplicates
	exactDuplicates := analyzer.FindContentDuplicates(files, ExactMatch)
	assert.Len(t, exactDuplicates, 1)
	assert.Len(t, exactDuplicates[0].Files, 2)
	assert.Contains(t, exactDuplicates[0].Files, "original.md")
	assert.Contains(t, exactDuplicates[0].Files, "duplicate.md")

	// Test similarity-based duplicates
	similarDuplicates := analyzer.FindContentDuplicates(files, SimilarityMatch)
	assert.Greater(t, len(similarDuplicates), 0)
}

func TestAnalyzer_AnalyzeField(t *testing.T) {
	analyzer := NewAnalyzer()

	files := []*vault.VaultFile{
		{
			Path: "file1.md",
			Frontmatter: map[string]interface{}{
				"priority": 1,
				"tags":     []interface{}{"work", "urgent"},
				"created":  "2023-01-01",
			},
		},
		{
			Path: "file2.md",
			Frontmatter: map[string]interface{}{
				"priority": 2,
				"tags":     []interface{}{"personal"},
				"created":  "2023-01-02",
			},
		},
		{
			Path: "file3.md",
			Frontmatter: map[string]interface{}{
				"priority": 1,
				"created":  "invalid-date",
			},
		},
		{
			Path:        "file4.md",
			Frontmatter: map[string]interface{}{}, // Missing fields
		},
	}

	analysis := analyzer.AnalyzeField(files, "priority")
	assert.Equal(t, "priority", analysis.FieldName)
	assert.Equal(t, 3, analysis.TotalFiles)   // Files that have the field
	assert.Equal(t, 1, analysis.MissingCount) // Files that don't have the field
	assert.Equal(t, 2, analysis.UniqueValues)
	assert.Equal(t, map[interface{}]int{1: 2, 2: 1}, analysis.ValueDistribution)
	assert.Equal(t, "number", analysis.PredominantType)

	// Test tags field (array)
	tagsAnalysis := analyzer.AnalyzeField(files, "tags")
	assert.Equal(t, 2, tagsAnalysis.TotalFiles)
	assert.Equal(t, 2, tagsAnalysis.MissingCount)
	assert.Equal(t, 2, tagsAnalysis.UniqueValues) // Two different tag arrays
	assert.Equal(t, "array", tagsAnalysis.PredominantType)

	// Test created field (mixed types)
	createdAnalysis := analyzer.AnalyzeField(files, "created")
	assert.Equal(t, 3, createdAnalysis.TotalFiles)
	assert.Equal(t, 1, createdAnalysis.MissingCount)
	assert.Equal(t, 3, createdAnalysis.UniqueValues) // Three different created values
}

func TestAnalyzer_FindOrphanedFiles(t *testing.T) {
	analyzer := NewAnalyzer()

	files := []*vault.VaultFile{
		{
			Path: "linked.md",
			Body: "# Linked File\n\nThis file is referenced",
		},
		{
			Path: "linker.md",
			Body: "# File with Links\n\nSee [[linked]] for more info",
		},
		{
			Path: "orphaned.md",
			Body: "# Orphaned File\n\nNo one links to this file",
		},
		{
			Path: "self-referencing.md",
			Body: "# Self Reference\n\nSee [[self-referencing]] for recursion",
		},
	}

	// Parse links in files
	for _, file := range files {
		// Simulate link parsing
		if file.Path == "linker.md" {
			file.Links = []vault.Link{
				{Type: vault.WikiLink, Target: "linked", Text: "linked"},
			}
		}
		if file.Path == "self-referencing.md" {
			file.Links = []vault.Link{
				{Type: vault.WikiLink, Target: "self-referencing", Text: "self-referencing"},
			}
		}
	}

	orphaned := analyzer.FindOrphanedFiles(files)

	// Convert to paths for easier testing
	orphanedPaths := make([]string, len(orphaned))
	for i, f := range orphaned {
		orphanedPaths[i] = f.Path
	}

	// Only linked.md should NOT be orphaned (because linker.md links to it)
	// All other files should be orphaned
	assert.Contains(t, orphanedPaths, "orphaned.md")
	assert.Contains(t, orphanedPaths, "self-referencing.md") // Self-references don't count
	assert.Contains(t, orphanedPaths, "linker.md")           // Nothing links to this
	assert.NotContains(t, orphanedPaths, "linked.md")        // This is linked by linker.md
}

func TestAnalyzer_GetHealthScore(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		name     string
		files    []*vault.VaultFile
		expected HealthLevel
	}{
		{
			name: "excellent health",
			files: []*vault.VaultFile{
				{
					Path: "perfect.md",
					Frontmatter: map[string]interface{}{
						"title": "Perfect File",
						"tags":  []interface{}{"complete"},
					},
					Body: "# Perfect File\n\nWell structured content",
				},
			},
			expected: Excellent,
		},
		{
			name: "poor health with missing frontmatter",
			files: []*vault.VaultFile{
				{
					Path:        "bad1.md",
					Frontmatter: map[string]interface{}{},
					Body:        "No frontmatter",
				},
				{
					Path:        "bad2.md",
					Frontmatter: map[string]interface{}{},
					Body:        "Also no frontmatter",
				},
			},
			expected: Poor,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := analyzer.GenerateStats(tt.files)
			score := analyzer.GetHealthScore(stats)
			assert.Equal(t, tt.expected, score.Level)
		})
	}
}

// Helper function to create test vault
func createTestVault(t *testing.T) *TestVault {
	files := []*vault.VaultFile{
		{
			Path: "project1.md",
			Frontmatter: map[string]interface{}{
				"title": "Project One",
				"tags":  []interface{}{"project", "important"},
				"id":    "proj-001",
			},
			Body:    "# Project One\n\n## Overview\n\nContent here\n\nSee [[project2]] for related work",
			Content: []byte("---\ntitle: Project One\ntags: [project, important]\nid: proj-001\n---\n\n# Project One\n\n## Overview\n\nContent here"),
			Links: []vault.Link{
				{Type: vault.WikiLink, Target: "project2", Text: "project2"},
			},
			Headings: []vault.Heading{
				{Level: 1, Text: "Project One", Line: 1},
				{Level: 2, Text: "Overview", Line: 3},
			},
		},
		{
			Path: "project2.md",
			Frontmatter: map[string]interface{}{
				"title": "Project Two",
				"tags":  []interface{}{"project"},
				"id":    "proj-002",
			},
			Body:    "# Project Two\n\n## Details\n\nMore content",
			Content: []byte("---\ntitle: Project Two\ntags: [project]\nid: proj-002\n---\n\n# Project Two\n\n## Details\n\nMore content"),
			Links:   []vault.Link{},
			Headings: []vault.Heading{
				{Level: 1, Text: "Project Two", Line: 1},
				{Level: 2, Text: "Details", Line: 3},
			},
		},
		{
			Path: "notes.md",
			Frontmatter: map[string]interface{}{
				"title": "Random Notes",
			},
			Body:    "# Random Notes\n\nSome notes without much structure",
			Content: []byte("---\ntitle: Random Notes\n---\n\n# Random Notes\n\nSome notes without much structure"),
			Links:   []vault.Link{},
			Headings: []vault.Heading{
				{Level: 1, Text: "Random Notes", Line: 1},
			},
		},
		{
			Path:        "no-frontmatter.md",
			Frontmatter: map[string]interface{}{},
			Body:        "# No Frontmatter\n\nThis file has no frontmatter",
			Content:     []byte("# No Frontmatter\n\nThis file has no frontmatter"),
			Links:       []vault.Link{},
			Headings: []vault.Heading{
				{Level: 1, Text: "No Frontmatter", Line: 1},
			},
		},
	}

	// Set file sizes and modification times
	for _, file := range files {
		file.Modified = time.Now()
	}

	return &TestVault{
		Files: files,
		Path:  "/test/vault",
	}
}

type TestVault struct {
	Files []*vault.VaultFile
	Path  string
}
