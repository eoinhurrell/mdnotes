package processor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// generateLargeVault creates a test vault with the specified number of files
func generateLargeVault(b *testing.B, fileCount int) *Vault {
	files := make([]*vault.VaultFile, fileCount)

	for i := 0; i < fileCount; i++ {
		files[i] = &vault.VaultFile{
			Path:         fmt.Sprintf("file_%d.md", i),
			RelativePath: fmt.Sprintf("file_%d.md", i),
			Modified:     time.Now(),
			Frontmatter: map[string]interface{}{
				"title": fmt.Sprintf("File %d", i),
				"id":    fmt.Sprintf("file-%d", i),
			},
			Body: fmt.Sprintf("# File %d\n\nThis is the content of file %d.\n\n[[link_%d]]", i, i, (i+1)%fileCount),
		}

		// Parse content to populate links and headings
		content := fmt.Sprintf("---\ntitle: File %d\nid: file-%d\n---\n%s", i, i, files[i].Body)
		files[i].Parse([]byte(content))
	}

	return &Vault{
		Files: files,
		Path:  "/test/vault",
	}
}

func BenchmarkFrontmatterEnsure(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("files-%d", size), func(b *testing.B) {
			vault := generateLargeVault(b, size)
			processor := NewFrontmatterProcessor()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for _, file := range vault.Files {
					processor.Ensure(file, "tags", []string{})
				}
			}
		})
	}
}

func BenchmarkFrontmatterValidate(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("files-%d", size), func(b *testing.B) {
			vault := generateLargeVault(b, size)
			validator := NewValidator(ValidationRules{
				Required: []string{"title", "id"},
				Types: map[string]string{
					"title": "string",
					"id":    "string",
				},
			})

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for _, file := range vault.Files {
					validator.Validate(file)
				}
			}
		})
	}
}

func BenchmarkTypeCasting(b *testing.B) {
	caster := NewTypeCaster()
	testValues := []interface{}{
		"2023-01-01",
		"42",
		"true",
		"tag1, tag2, tag3",
		"2023-01-01T10:30:00Z",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, value := range testValues {
			caster.AutoDetect(value)
		}
	}
}

func BenchmarkHeadingProcessing(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("files-%d", size), func(b *testing.B) {
			vault := generateLargeVault(b, size)
			processor := NewHeadingProcessor()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for _, file := range vault.Files {
					processor.Analyze(file)
				}
			}
		})
	}
}

func BenchmarkLinkParsing(b *testing.B) {
	parser := NewLinkParser()
	testContent := `
# Test Note

This note contains various link types:

- Wiki links: [[other note]] and [[folder/note|custom text]]
- Markdown links: [text](note.md) and [another](../path/note.md)
- Embedded links: ![[image.png]] and ![[diagram.svg]]
- URLs: [external](https://example.com)

More content with [[many]] [[different]] [[wiki]] [[links]] and
[multiple](one.md) [markdown](two.md) [links](three.md).

![[embed1.png]] ![[embed2.jpg]] ![[embed3.pdf]]
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.Extract(testContent)
	}
}

func BenchmarkBatchProcessing(b *testing.B) {
	sizes := []int{100, 1000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("files-%d", size), func(b *testing.B) {
			vault := generateLargeVault(b, size)
			processor := NewBatchProcessor()

			config := BatchConfig{
				Operations: []Operation{
					{
						Name:    "Ensure tags",
						Command: "frontmatter.ensure",
						Parameters: map[string]interface{}{
							"field":   "tags",
							"default": []string{},
						},
					},
					{
						Name:    "Validate fields",
						Command: "frontmatter.validate",
						Parameters: map[string]interface{}{
							"required": []string{"title"},
						},
					},
				},
				StopOnError:  false,
				CreateBackup: false,
				DryRun:       true,
				MaxWorkers:   1, // Sequential for consistent benchmarking
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				processor.Execute(context.Background(), vault, config)
			}
		})
	}
}

func BenchmarkParallelProcessing(b *testing.B) {
	vault := generateLargeVault(b, 1000)

	workerCounts := []int{1, 2, 4, 8}

	for _, workers := range workerCounts {
		b.Run(fmt.Sprintf("workers-%d", workers), func(b *testing.B) {
			processor := NewBatchProcessor()

			config := BatchConfig{
				Operations: []Operation{
					{
						Name:    "Ensure tags",
						Command: "frontmatter.ensure",
						Parameters: map[string]interface{}{
							"field":   "tags",
							"default": []string{},
						},
					},
				},
				MaxWorkers:   workers,
				StopOnError:  false,
				CreateBackup: false,
				DryRun:       true,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				processor.Execute(context.Background(), vault, config)
			}
		})
	}
}

func BenchmarkVaultScanning(b *testing.B) {
	// This benchmark would need a real filesystem, so we'll skip the actual file I/O
	// and just benchmark the ignore pattern matching
	// This benchmark focuses on ignore pattern matching performance
	ignorePatterns := []string{
		".obsidian/*",
		"*.tmp",
		"*.bak",
		".DS_Store",
		"node_modules/*",
		".git/*",
	}

	testPaths := []string{
		"normal-file.md",
		".obsidian/config.json",
		"temp-file.tmp",
		"backup.bak",
		".DS_Store",
		"node_modules/package.json",
		".git/config",
		"folder/subfolder/note.md",
		"another/path/document.md",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, path := range testPaths {
			// Benchmark pattern matching logic
			shouldIgnore := false
			for _, pattern := range ignorePatterns {
				if pattern == path {
					shouldIgnore = true
					break
				}
			}
			_ = shouldIgnore
		}
	}
}

// Memory allocation benchmarks
func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("VaultFile-Parse", func(b *testing.B) {
		content := []byte(`---
title: Test Note
tags: [tag1, tag2, tag3]
created: 2023-01-01
id: test-123
---

# Test Note

This is a test note with some content.

## Section 1

Content here with [[wiki links]] and [markdown links](other.md).

## Section 2

More content with ![[embedded.png]] files.
`)

		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			vf := &vault.VaultFile{}
			vf.Parse(content)
		}
	})

	b.Run("LinkParser-Extract", func(b *testing.B) {
		parser := NewLinkParser()
		content := "This has [[wiki]] and [markdown](link.md) and ![[embed.png]] links repeated multiple times. " +
			"[[another]] [more](links.md) ![[image.jpg]] patterns for testing memory allocation."

		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			parser.Extract(content)
		}
	})
}

// CPU profiling helper (not a benchmark, but useful for profiling)
func BenchmarkCPUIntensive(b *testing.B) {
	vault := generateLargeVault(b, 5000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate CPU-intensive operations
		processor := NewFrontmatterProcessor()
		validator := NewValidator(ValidationRules{
			Required: []string{"title", "id"},
		})
		linkParser := NewLinkParser()

		for _, file := range vault.Files {
			processor.Ensure(file, "tags", []string{})
			validator.Validate(file)
			linkParser.Extract(file.Body)
		}
	}
}
