package processor

import (
	"context"
	"fmt"
	"testing"

	"github.com/eoinhurrell/mdnotes/internal/vault"
	"github.com/stretchr/testify/assert"
)

func TestNewExportBacklinksHandler(t *testing.T) {
	files := []*vault.VaultFile{
		{RelativePath: "test.md", Body: "# Test"},
	}
	
	handler := NewExportBacklinksHandler(files, true)
	
	assert.Equal(t, files, handler.allVaultFiles)
	assert.True(t, handler.verbose)
	assert.Equal(t, 10, handler.maxDepth)
}

func TestBacklinksHandler_ResolveLinkPath(t *testing.T) {
	allFiles := []*vault.VaultFile{
		{RelativePath: "notes/note1.md"},
		{RelativePath: "notes/note2.md"},
		{RelativePath: "projects/project1.md"},
		{RelativePath: "areas/area1.md"},
	}
	
	handler := NewExportBacklinksHandler(allFiles, false)
	
	tests := []struct {
		name               string
		target             string
		sourceRelativePath string
		expected           string
	}{
		{
			name:               "Wiki link same directory",
			target:             "note2",
			sourceRelativePath: "notes/note1.md",
			expected:           "notes/note2.md",
		},
		{
			name:               "Wiki link with extension",
			target:             "note2.md",
			sourceRelativePath: "notes/note1.md",
			expected:           "notes/note2.md",
		},
		{
			name:               "Wiki link cross directory",
			target:             "project1",
			sourceRelativePath: "notes/note1.md",
			expected:           "projects/project1.md",
		},
		{
			name:               "Relative path with ../",
			target:             "../projects/project1.md",
			sourceRelativePath: "notes/note1.md",
			expected:           "projects/project1.md",
		},
		{
			name:               "Absolute path",
			target:             "/areas/area1",
			sourceRelativePath: "notes/note1.md",
			expected:           "areas/area1.md",
		},
		{
			name:               "External URL",
			target:             "https://example.com",
			sourceRelativePath: "notes/note1.md",
			expected:           "", // External URLs return empty
		},
		{
			name:               "Link with fragment",
			target:             "note2#section",
			sourceRelativePath: "notes/note1.md",
			expected:           "notes/note2.md",
		},
		{
			name:               "Non-existent file",
			target:             "missing",
			sourceRelativePath: "notes/note1.md",
			expected:           "notes/missing.md", // Fallback to same directory
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.resolveLinkPath(tt.target, tt.sourceRelativePath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBacklinksHandler_FileExists(t *testing.T) {
	allFiles := []*vault.VaultFile{
		{RelativePath: "notes/existing.md"},
		{RelativePath: "projects/project.md"},
	}
	
	handler := NewExportBacklinksHandler(allFiles, false)
	
	// Test existing files
	assert.True(t, handler.fileExists("notes/existing.md"))
	assert.True(t, handler.fileExists("projects/project.md"))
	
	// Test non-existing files
	assert.False(t, handler.fileExists("notes/missing.md"))
	assert.False(t, handler.fileExists("nonexistent/path.md"))
}

func TestBacklinksHandler_FindBacklinksToFiles(t *testing.T) {
	// Create a vault with various linking patterns
	allFiles := []*vault.VaultFile{
		{
			RelativePath: "target.md",
			Body:         "# Target File\n\nThis is the target file.",
		},
		{
			RelativePath: "linker1.md",
			Body:         "# Linker 1\n\nThis links to [[target]] file.",
		},
		{
			RelativePath: "linker2.md",
			Body:         "# Linker 2\n\nThis has a [Target](target.md) link.",
		},
		{
			RelativePath: "no-links.md",
			Body:         "# No Links\n\nThis file has no links.",
		},
		{
			RelativePath: "external-links.md",
			Body:         "# External Links\n\nThis links to [Google](https://google.com).",
		},
		{
			RelativePath: "different-target.md",
			Body:         "# Different Target\n\nThis links to [[other-file]].",
		},
	}
	
	handler := NewExportBacklinksHandler(allFiles, false)
	
	// Find backlinks to target.md
	targetFiles := []*vault.VaultFile{
		{RelativePath: "target.md", Body: "# Target File"},
	}
	
	processedFiles := map[string]bool{
		"target.md": true, // Target file is already processed
	}
	
	backlinks := handler.findBacklinksToFiles(targetFiles, processedFiles)
	
	// Should find linker1.md and linker2.md
	assert.Len(t, backlinks, 2)
	
	backlinkPaths := make([]string, len(backlinks))
	for i, file := range backlinks {
		backlinkPaths[i] = file.RelativePath
	}
	
	assert.Contains(t, backlinkPaths, "linker1.md")
	assert.Contains(t, backlinkPaths, "linker2.md")
	assert.NotContains(t, backlinkPaths, "no-links.md")
	assert.NotContains(t, backlinkPaths, "external-links.md")
	assert.NotContains(t, backlinkPaths, "different-target.md")
}

func TestBacklinksHandler_DiscoverBacklinks(t *testing.T) {
	ctx := context.Background()
	
	// Create a more complex vault with multi-level backlinks
	allFiles := []*vault.VaultFile{
		{
			RelativePath: "original.md",
			Body:         "# Original File\n\nThis is the original file.",
		},
		{
			RelativePath: "level1-a.md",
			Body:         "# Level 1A\n\nLinks to [[original]] file.",
		},
		{
			RelativePath: "level1-b.md",
			Body:         "# Level 1B\n\nAlso links to [Original](original.md).",
		},
		{
			RelativePath: "level2.md",
			Body:         "# Level 2\n\nLinks to [[level1-a]] which links to original.",
		},
		{
			RelativePath: "isolated.md",
			Body:         "# Isolated\n\nThis file has no connections.",
		},
		{
			RelativePath: "circular-a.md",
			Body:         "# Circular A\n\nLinks to [[circular-b]].",
		},
		{
			RelativePath: "circular-b.md",
			Body:         "# Circular B\n\nLinks back to [[circular-a]].",
		},
	}
	
	handler := NewExportBacklinksHandler(allFiles, false)
	
	t.Run("Single level backlinks", func(t *testing.T) {
		exportedFiles := []*vault.VaultFile{
			{RelativePath: "original.md", Body: "# Original File"},
		}
		
		result := handler.DiscoverBacklinks(ctx, exportedFiles)
		
		// Should find level1-a.md, level1-b.md, and level2.md (recursive)
		assert.Equal(t, 3, result.TotalBacklinks)
		
		backlinkPaths := make([]string, len(result.BacklinkFiles))
		for i, file := range result.BacklinkFiles {
			backlinkPaths[i] = file.RelativePath
		}
		
		assert.Contains(t, backlinkPaths, "level1-a.md")
		assert.Contains(t, backlinkPaths, "level1-b.md")
		assert.Contains(t, backlinkPaths, "level2.md") // Recursive backlink
		
		// Should have processed original file + backlinks
		assert.True(t, result.ProcessedFiles["original.md"])
		assert.True(t, result.ProcessedFiles["level1-a.md"])
		assert.True(t, result.ProcessedFiles["level1-b.md"])
	})
	
	t.Run("Multi-level backlinks", func(t *testing.T) {
		exportedFiles := []*vault.VaultFile{
			{RelativePath: "level1-a.md", Body: "# Level 1A"},
		}
		
		result := handler.DiscoverBacklinks(ctx, exportedFiles)
		
		// Should find level2.md (which links to level1-a)
		assert.Equal(t, 1, result.TotalBacklinks)
		assert.Equal(t, "level2.md", result.BacklinkFiles[0].RelativePath)
	})
	
	t.Run("No backlinks", func(t *testing.T) {
		exportedFiles := []*vault.VaultFile{
			{RelativePath: "isolated.md", Body: "# Isolated"},
		}
		
		result := handler.DiscoverBacklinks(ctx, exportedFiles)
		
		// Should find no backlinks
		assert.Equal(t, 0, result.TotalBacklinks)
		assert.Empty(t, result.BacklinkFiles)
	})
	
	t.Run("Circular references", func(t *testing.T) {
		exportedFiles := []*vault.VaultFile{
			{RelativePath: "circular-a.md", Body: "# Circular A"},
		}
		
		result := handler.DiscoverBacklinks(ctx, exportedFiles)
		
		// Should find circular-b.md but not get stuck in infinite loop
		assert.Equal(t, 1, result.TotalBacklinks)
		assert.Equal(t, "circular-b.md", result.BacklinkFiles[0].RelativePath)
		
		// Should have processed both files to prevent cycles
		assert.True(t, result.ProcessedFiles["circular-a.md"])
		assert.True(t, result.ProcessedFiles["circular-b.md"])
	})
}

func TestBacklinksHandler_ContextCancellation(t *testing.T) {
	// Create a large vault for timeout testing
	allFiles := make([]*vault.VaultFile, 100)
	for i := 0; i < 100; i++ {
		allFiles[i] = &vault.VaultFile{
			RelativePath: fmt.Sprintf("file%d.md", i),
			Body:         fmt.Sprintf("# File %d\n\nContent for file %d", i, i),
		}
	}
	
	handler := NewExportBacklinksHandler(allFiles, false)
	
	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	
	exportedFiles := []*vault.VaultFile{
		{RelativePath: "file0.md", Body: "# File 0"},
	}
	
	result := handler.DiscoverBacklinks(ctx, exportedFiles)
	
	// Should return early due to context cancellation
	// Result should still be valid but may be incomplete
	assert.NotNil(t, result)
	assert.NotNil(t, result.BacklinkFiles)
	assert.NotNil(t, result.ProcessedFiles)
}

func TestBacklinksHandler_MaxDepthLimit(t *testing.T) {
	ctx := context.Background()
	
	// Create a chain of files that link to each other
	allFiles := make([]*vault.VaultFile, 15)
	for i := 0; i < 15; i++ {
		var body string
		if i < 14 {
			body = fmt.Sprintf("# File %d\n\nLinks to [[file%d]]", i, i+1)
		} else {
			body = fmt.Sprintf("# File %d\n\nLast file in chain", i)
		}
		
		allFiles[i] = &vault.VaultFile{
			RelativePath: fmt.Sprintf("file%d.md", i),
			Body:         body,
		}
	}
	
	handler := NewExportBacklinksHandler(allFiles, false)
	
	// Start with the last file in the chain
	exportedFiles := []*vault.VaultFile{
		{RelativePath: "file14.md", Body: "# File 14"},
	}
	
	result := handler.DiscoverBacklinks(ctx, exportedFiles)
	
	// Should stop at max depth (10) and not process all 14 potential backlinks
	assert.LessOrEqual(t, result.TotalBacklinks, 10, "Should respect max depth limit")
	assert.Greater(t, result.TotalBacklinks, 0, "Should find some backlinks")
}