package processor

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

func TestRenameProcessor_LinkMatchesMove(t *testing.T) {
	processor := NewRenameProcessor(RenameOptions{})

	tests := []struct {
		name   string
		link   vault.Link
		move   FileMove
		expect bool
	}{
		// Wiki link tests
		{
			name:   "wiki link matches exact path",
			link:   vault.Link{Type: vault.WikiLink, Target: "resources/test"},
			move:   FileMove{From: "resources/test.md", To: "resources/renamed.md"},
			expect: true,
		},
		{
			name:   "wiki link matches basename",
			link:   vault.Link{Type: vault.WikiLink, Target: "test"},
			move:   FileMove{From: "resources/test.md", To: "resources/renamed.md"},
			expect: true,
		},
		{
			name:   "wiki link with extension matches",
			link:   vault.Link{Type: vault.WikiLink, Target: "resources/test.md"},
			move:   FileMove{From: "resources/test.md", To: "resources/renamed.md"},
			expect: true,
		},
		// Markdown link tests - key fixes
		{
			name:   "markdown link matches vault-relative path",
			link:   vault.Link{Type: vault.MarkdownLink, Target: "resources/test.md"},
			move:   FileMove{From: "resources/test.md", To: "resources/renamed.md"},
			expect: true,
		},
		{
			name:   "markdown link with subdirectory",
			link:   vault.Link{Type: vault.MarkdownLink, Target: "docs/guides/setup.md"},
			move:   FileMove{From: "docs/guides/setup.md", To: "documentation/setup.md"},
			expect: true,
		},
		{
			name:   "markdown link without extension",
			link:   vault.Link{Type: vault.MarkdownLink, Target: "resources/test"},
			move:   FileMove{From: "resources/test.md", To: "resources/renamed.md"},
			expect: true,
		},
		// Embed link tests
		{
			name:   "embed link matches path",
			link:   vault.Link{Type: vault.EmbedLink, Target: "images/photo.png"},
			move:   FileMove{From: "images/photo.png", To: "assets/photo.png"},
			expect: true,
		},
		// Fragment handling
		{
			name:   "link with fragment matches",
			link:   vault.Link{Type: vault.MarkdownLink, Target: "resources/test.md", Fragment: "section"},
			move:   FileMove{From: "resources/test.md", To: "resources/renamed.md"},
			expect: true,
		},
		// Non-matching cases
		{
			name:   "markdown link doesn't match different path",
			link:   vault.Link{Type: vault.MarkdownLink, Target: "other/file.md"},
			move:   FileMove{From: "resources/test.md", To: "resources/renamed.md"},
			expect: false,
		},
		{
			name:   "markdown link doesn't match basename only",
			link:   vault.Link{Type: vault.MarkdownLink, Target: "test.md"},
			move:   FileMove{From: "resources/test.md", To: "resources/renamed.md"},
			expect: false,
		},
		// URL encoding tests
		{
			name:   "markdown link with URL encoding matches",
			link:   vault.Link{Type: vault.MarkdownLink, Target: "resources/books/20250527111132-Blood's%20Hiding.md"},
			move:   FileMove{From: "resources/books/20250527111132-Blood's Hiding.md", To: "resources/books/20250527111132-renamed.md"},
			expect: true,
		},
		{
			name:   "markdown link with spaces matches URL encoded",
			link:   vault.Link{Type: vault.MarkdownLink, Target: "resources/books/file%20with%20spaces.md"},
			move:   FileMove{From: "resources/books/file with spaces.md", To: "resources/books/renamed.md"},
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.linkMatchesMove(tt.link, tt.move)
			if result != tt.expect {
				t.Errorf("linkMatchesMove() = %v, expect %v for link %q -> move %q",
					result, tt.expect, tt.link.Target, tt.move.From)
			}
		})
	}
}

func TestRenameProcessor_Performance(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "mdnotes_rename_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files with various link patterns
	testFiles := []struct {
		name    string
		content string
	}{
		{
			name: "source.md",
			content: `---
title: Source File
created: 2024-01-01
---

# Source File

This is the source file that will be renamed.
`,
		},
		{
			name: "file1.md",
			content: `---
title: File 1
---

# File 1

This file references [[source]] in multiple ways:
- [[source|Source File]]
- [[source#header]]
- [Link to source](source.md)
`,
		},
		{
			name: "file2.md",
			content: `---
title: File 2
---

# File 2

No references to the source file here.
Just some random content.
`,
		},
		{
			name: "file3.md",
			content: `---
title: File 3
---

# File 3

Another file with references:
- [[source]]
- ![[source]]
`,
		},
	}

	// Create test files
	for _, tf := range testFiles {
		path := filepath.Join(tempDir, tf.name)
		if err := os.WriteFile(path, []byte(tf.content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", tf.name, err)
		}
	}

	// Test the rename operation
	sourcePath := filepath.Join(tempDir, "source.md")
	targetPath := filepath.Join(tempDir, "renamed-source.md")

	options := RenameOptions{
		VaultRoot:      tempDir,
		IgnorePatterns: []string{},
		DryRun:         true, // Use dry run for testing
		Verbose:        false,
		Workers:        2,
	}

	processor := NewRenameProcessor(options)

	startTime := time.Now()
	result, err := processor.ProcessRename(context.Background(), sourcePath, targetPath, options)
	duration := time.Since(startTime)

	if err != nil {
		t.Fatalf("Rename processing failed: %v", err)
	}

	// Verify results
	if result.FilesScanned != 4 { // All 4 test files should be scanned
		t.Errorf("Expected 4 files scanned, got %d", result.FilesScanned)
	}

	if result.FilesModified != 2 { // file1.md and file3.md should be modified
		t.Errorf("Expected 2 files modified, got %d", result.FilesModified)
	}

	if result.LinksUpdated != 6 { // Total links that would be updated: 4 in file1.md + 2 in file3.md
		t.Errorf("Expected 6 links updated, got %d", result.LinksUpdated)
	}

	t.Logf("Performance results:")
	t.Logf("  Duration: %v", duration)
	t.Logf("  Files scanned: %d", result.FilesScanned)
	t.Logf("  Files modified: %d", result.FilesModified)
	t.Logf("  Links updated: %d", result.LinksUpdated)
	t.Logf("  Processing rate: %.2f files/ms", float64(result.FilesScanned)/float64(duration.Milliseconds()))
}

func TestGenerateNameFromTemplate(t *testing.T) {
	// Create a temporary test file
	tempDir, err := ioutil.TempDir("", "mdnotes_template_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.md")
	content := `---
title: Test File
created: 2024-01-15
---

# Test File
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "Basic template",
			template: "{{filename|slugify}}.md",
			expected: "test.md",
		},
		{
			name:     "Date template",
			template: "{{created|date:2006-01-02}}-{{filename}}.md",
			expected: "2024-01-15-test.md",
		},
		{
			name:     "Complex template",
			template: "{{created|date:20060102}}-{{filename|slugify}}.md",
			expected: "20240115-test.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateNameFromTemplate(testFile, tt.template)
			if err != nil {
				t.Fatalf("Template generation failed: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func BenchmarkRenameProcessor(b *testing.B) {
	// Create a larger test vault for benchmarking
	tempDir, err := ioutil.TempDir("", "mdnotes_bench")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create source file
	sourcePath := filepath.Join(tempDir, "source.md")
	sourceContent := `---
title: Source File
created: 2024-01-01
---

# Source File
`
	if err := os.WriteFile(sourcePath, []byte(sourceContent), 0644); err != nil {
		b.Fatalf("Failed to create source file: %v", err)
	}

	// Create many test files with varying link patterns
	numFiles := 100
	for i := 0; i < numFiles; i++ {
		fileName := fmt.Sprintf("file%d.md", i)
		var content string

		// Every 3rd file has links to source
		if i%3 == 0 {
			content = fmt.Sprintf(`---
title: File %d
---

# File %d

This file references [[source]] and [link](source.md).
More content here to make files more realistic.
`, i, i)
		} else {
			content = fmt.Sprintf(`---
title: File %d
---

# File %d

This file has no links to source.
Just some content to make it realistic.
Some more text to pad the file size.
`, i, i)
		}

		path := filepath.Join(tempDir, fileName)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			b.Fatalf("Failed to create test file %s: %v", fileName, err)
		}
	}

	targetPath := filepath.Join(tempDir, "renamed-source.md")
	options := RenameOptions{
		VaultRoot:      tempDir,
		IgnorePatterns: []string{},
		DryRun:         true,
		Verbose:        false,
		Workers:        4,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		processor := NewRenameProcessor(options)
		_, err := processor.ProcessRename(context.Background(), sourcePath, targetPath, options)
		if err != nil {
			b.Fatalf("Benchmark iteration %d failed: %v", i, err)
		}
	}
}

func BenchmarkLinkMatching(b *testing.B) {
	processor := &RenameProcessor{}

	// Test link
	link := vault.Link{
		Type:   vault.WikiLink,
		Target: "source",
		Text:   "Source",
	}

	// Test move
	move := FileMove{
		From: "source.md",
		To:   "renamed-source.md",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		processor.linkMatchesMove(link, move)
	}
}
