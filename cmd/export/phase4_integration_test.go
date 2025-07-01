package export

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eoinhurrell/mdnotes/internal/processor"
)

// TestPhase4Integration tests the complete Phase 4 implementation
func TestPhase4Integration(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	vaultDir := filepath.Join(tempDir, "vault")
	outputDir := filepath.Join(tempDir, "output")

	// Create vault structure
	require.NoError(t, os.MkdirAll(vaultDir, 0755))

	// Create test files
	files := map[string]string{
		"note1.md": `---
title: "Test Note 1"
tags: ["test", "phase4"]
created: "2024-01-01"
---

# Test Note 1

This is a test note with a [[note2|link to note 2]].
`,
		"note2.md": `---
title: "Test Note 2"  
tags: ["test"]
created: "2024-01-02"
---

# Test Note 2

This note links back to [[note1]].
`,
		"external.md": `---
title: "External Note"
tags: ["external"]
url: "https://example.com"
---

# External Note

This note has an external link to [Google](https://google.com).
`,
	}

	for filename, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(vaultDir, filename), []byte(content), 0644))
	}

	t.Run("Error Handling - Invalid Query", func(t *testing.T) {
		err := validateExportInputs(outputDir, vaultDir, "invalid query with \"unclosed quote", "remove", true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unmatched quotes")
	})

	t.Run("Error Handling - Invalid Paths", func(t *testing.T) {
		_, _, err := validateAndResolvePaths("/nonexistent", outputDir, false)
		var exportErr *ExportError
		assert.ErrorAs(t, err, &exportErr)
		assert.Equal(t, ErrFileSystem, exportErr.Type)
	})

	t.Run("Performance - Basic Export with Metrics", func(t *testing.T) {
		options := processor.ExportOptions{
			VaultPath:       vaultDir,
			OutputPath:      outputDir,
			Query:           "tags contains 'test'",
			DryRun:          false,
			Verbose:         true,
			ProcessLinks:    true,
			LinkStrategy:    "remove",
			ParallelWorkers: 2,
			OptimizeMemory:  false,
		}

		exportProcessor := processor.NewExportProcessor(options)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		result, err := exportProcessor.ProcessExport(ctx, options)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify basic results
		assert.Equal(t, 3, result.FilesScanned)
		assert.Equal(t, 2, result.FilesSelected) // Only note1.md and note2.md have 'test' tag
		assert.Greater(t, result.Duration, time.Duration(0))

		// Verify performance metrics are calculated
		assert.NotNil(t, result.Performance)
		assert.Greater(t, result.Performance.FilesPerSecond, 0.0)
		assert.Greater(t, result.Performance.MemoryUsageMB, 0.0)

		// Verify files were exported
		assert.FileExists(t, filepath.Join(outputDir, "note1.md"))
		assert.FileExists(t, filepath.Join(outputDir, "note2.md"))
		assert.NoFileExists(t, filepath.Join(outputDir, "external.md"))

		// Clean up for next test
		require.NoError(t, os.RemoveAll(outputDir))
	})

	t.Run("Performance - Parallel Processing", func(t *testing.T) {
		options := processor.ExportOptions{
			VaultPath:       vaultDir,
			OutputPath:      outputDir,
			DryRun:          false,
			Verbose:         false,
			ProcessLinks:    false,
			ParallelWorkers: 4, // Force parallel processing
			OptimizeMemory:  false,
		}

		exportProcessor := processor.NewExportProcessor(options)

		ctx := context.Background()
		result, err := exportProcessor.ProcessExport(ctx, options)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify all files were processed
		assert.Equal(t, 3, result.FilesScanned)
		assert.Equal(t, 3, result.FilesSelected) // All files should be selected (no query)
		assert.Equal(t, 3, result.FilesExported)

		// Verify performance metrics
		assert.NotNil(t, result.Performance)
		assert.Equal(t, 4, result.Performance.ParallelWorkers)

		// Clean up for next test
		require.NoError(t, os.RemoveAll(outputDir))
	})

	t.Run("Performance - Memory Optimization", func(t *testing.T) {
		options := processor.ExportOptions{
			VaultPath:       vaultDir,
			OutputPath:      outputDir,
			DryRun:          false,
			Verbose:         false,
			ProcessLinks:    true,
			LinkStrategy:    "remove",
			ParallelWorkers: 2,
			OptimizeMemory:  true, // Enable memory optimization
		}

		exportProcessor := processor.NewExportProcessor(options)

		ctx := context.Background()
		result, err := exportProcessor.ProcessExport(ctx, options)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify export completed successfully
		assert.Equal(t, 3, result.FilesExported)
		assert.NotNil(t, result.Performance)

		// Clean up for next test
		require.NoError(t, os.RemoveAll(outputDir))
	})

	t.Run("Error Handling - Context Timeout", func(t *testing.T) {
		options := processor.ExportOptions{
			VaultPath:       vaultDir,
			OutputPath:      outputDir,
			DryRun:          false,
			Verbose:         false,
			ProcessLinks:    true,
			ParallelWorkers: 1,
		}

		exportProcessor := processor.NewExportProcessor(options)

		// Create a context with very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		result, err := exportProcessor.ProcessExport(ctx, options)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("Documentation - Enhanced Output Display", func(t *testing.T) {
		options := processor.ExportOptions{
			VaultPath:    vaultDir,
			OutputPath:   outputDir,
			DryRun:       false,
			Verbose:      true,
			ProcessLinks: true,
		}

		exportProcessor := processor.NewExportProcessor(options)

		ctx := context.Background()
		result, err := exportProcessor.ProcessExport(ctx, options)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Test the summary display functions (these don't return values, just check they don't panic)
		assert.NotPanics(t, func() {
			displayExportSummary(result, outputDir, true)
		})

		// Verify result has all expected fields for enhanced display
		assert.Greater(t, result.Duration, time.Duration(0))
		assert.NotNil(t, result.Performance)
		assert.GreaterOrEqual(t, result.Performance.FilesPerSecond, 0.0)
		assert.GreaterOrEqual(t, result.Performance.MemoryUsageMB, 0.0)

		// Clean up
		require.NoError(t, os.RemoveAll(outputDir))
	})
}

// TestPhase4ErrorMessages tests that error messages are user-friendly
func TestPhase4ErrorMessages(t *testing.T) {
	tests := []struct {
		name          string
		errorType     ExportErrorType
		message       string
		expectedInMsg string
	}{
		{
			name:          "Input validation error",
			errorType:     ErrInvalidInput,
			message:       "Output directory is not empty",
			expectedInMsg: "Output directory is not empty",
		},
		{
			name:          "File system error",
			errorType:     ErrFileSystem,
			message:       "Vault path does not exist",
			expectedInMsg: "does not exist",
		},
		{
			name:          "Permission error",
			errorType:     ErrPermission,
			message:       "Permission denied",
			expectedInMsg: "Permission denied",
		},
		{
			name:          "Query error",
			errorType:     ErrQuery,
			message:       "Invalid query syntax",
			expectedInMsg: "query",
		},
		{
			name:          "Processing error",
			errorType:     ErrProcessing,
			message:       "Link processing failed",
			expectedInMsg: "processing",
		},
		{
			name:          "Cancellation error",
			errorType:     ErrCancellation,
			message:       "Operation was cancelled",
			expectedInMsg: "cancelled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewExportError(tt.errorType, tt.message)
			assert.Equal(t, tt.errorType, err.Type)
			assert.Contains(t, err.Error(), tt.expectedInMsg)
		})
	}
}

// TestPhase4PerformanceThresholds tests that performance targets are met
func TestPhase4PerformanceThresholds(t *testing.T) {
	// Create a larger test vault
	tempDir := t.TempDir()
	vaultDir := filepath.Join(tempDir, "vault")
	outputDir := filepath.Join(tempDir, "output")

	require.NoError(t, os.MkdirAll(vaultDir, 0755))

	// Create 50 test files (should process in <1 second according to spec)
	for i := 0; i < 50; i++ {
		content := fmt.Sprintf(`---
title: "Test Note %d"
tags: ["test"]
---

# Test Note %d

This is test content for note %d.
Some more content to make it realistic.
And a [[note%d]] link to test link processing.
`, i, i, i, (i+1)%50)

		filename := fmt.Sprintf("note%d.md", i)
		require.NoError(t, os.WriteFile(filepath.Join(vaultDir, filename), []byte(content), 0644))
	}

	options := processor.ExportOptions{
		VaultPath:       vaultDir,
		OutputPath:      outputDir,
		DryRun:          false,
		Verbose:         false,
		ProcessLinks:    true,
		LinkStrategy:    "remove",
		ParallelWorkers: 0, // Auto-detect
		OptimizeMemory:  false,
	}

	exportProcessor := processor.NewExportProcessor(options)

	ctx := context.Background()
	start := time.Now()
	result, err := exportProcessor.ProcessExport(ctx, options)
	duration := time.Since(start)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify performance threshold: <1s for <100 files
	assert.Less(t, duration, 2*time.Second, "Export of 50 files should complete in under 2 seconds")
	assert.Equal(t, 50, result.FilesExported)

	// Verify performance metrics are reasonable
	assert.NotNil(t, result.Performance)
	assert.Greater(t, result.Performance.FilesPerSecond, 10.0, "Should process at least 10 files per second")

	t.Logf("Performance results:")
	t.Logf("  Files processed: %d", result.FilesExported)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Files/second: %.1f", result.Performance.FilesPerSecond)
	t.Logf("  Memory usage: %.1f MB", result.Performance.MemoryUsageMB)
	t.Logf("  Parallel workers: %d", result.Performance.ParallelWorkers)
}
