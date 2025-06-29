package processor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/eoinhurrell/mdnotes/internal/vault"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParallelFileProcessor_Creation(t *testing.T) {
	progress := NewExportProgressReporter(false, false)

	tests := []struct {
		name        string
		workerCount int
		expectedMin int
		expectedMax int
	}{
		{
			name:        "Auto-detect workers",
			workerCount: 0,
			expectedMin: 1,
			expectedMax: 8, // Capped at 8
		},
		{
			name:        "Explicit worker count",
			workerCount: 4,
			expectedMin: 4,
			expectedMax: 4,
		},
		{
			name:        "Single worker",
			workerCount: 1,
			expectedMin: 1,
			expectedMax: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := NewParallelFileProcessor(tt.workerCount, false, progress)

			assert.GreaterOrEqual(t, processor.workerCount, tt.expectedMin)
			assert.LessOrEqual(t, processor.workerCount, tt.expectedMax)
			assert.Equal(t, false, processor.optimizeMemory)
			assert.Equal(t, progress, processor.progress)
		})
	}
}

func TestParallelFileProcessor_SmallFileSet(t *testing.T) {
	progress := NewExportProgressReporter(false, false)
	processor := NewParallelFileProcessor(4, false, progress)

	// Create a small set of files (should use sequential processing)
	files := []*vault.VaultFile{
		{RelativePath: "file1.md", Body: "# File 1"},
		{RelativePath: "file2.md", Body: "# File 2"},
	}

	filenameMap := map[string]string{
		"file1.md": "file1.md",
		"file2.md": "file2.md",
	}

	options := ExportOptions{
		OutputPath: "/tmp/test",
	}

	// Mock processor function
	processCount := 0
	fileProcessor := func(file *vault.VaultFile, outputPath string, opts ExportOptions) (*FileProcessingResult, error) {
		processCount++
		return &FileProcessingResult{
			File:    file,
			Success: true,
		}, nil
	}

	ctx := context.Background()
	result, err := processor.ProcessFilesInParallel(ctx, files, filenameMap, options, fileProcessor)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, processCount)
}

func TestParallelFileProcessor_LargeFileSet(t *testing.T) {
	progress := NewExportProgressReporter(false, false)
	processor := NewParallelFileProcessor(2, false, progress) // Use 2 workers for predictable testing

	// Create a larger set of files (should use parallel processing)
	files := make([]*vault.VaultFile, 20)
	filenameMap := make(map[string]string)

	for i := 0; i < 20; i++ {
		filename := fmt.Sprintf("file%d.md", i)
		files[i] = &vault.VaultFile{
			RelativePath: filename,
			Body:         fmt.Sprintf("# File %d", i),
		}
		filenameMap[filename] = filename
	}

	options := ExportOptions{
		OutputPath: "/tmp/test",
	}

	// Mock processor function that simulates work
	processCount := 0
	fileProcessor := func(file *vault.VaultFile, outputPath string, opts ExportOptions) (*FileProcessingResult, error) {
		processCount++
		// Simulate some processing time
		time.Sleep(1 * time.Millisecond)
		return &FileProcessingResult{
			File:    file,
			Success: true,
		}, nil
	}

	ctx := context.Background()
	start := time.Now()
	result, err := processor.ProcessFilesInParallel(ctx, files, filenameMap, options, fileProcessor)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 20, processCount)

	// Parallel processing should be faster than sequential
	// With 2 workers and 1ms per file, should be roughly 10-15ms instead of 20ms
	assert.Less(t, duration, 18*time.Millisecond, "Parallel processing should be faster")
}

func TestParallelFileProcessor_ErrorHandling(t *testing.T) {
	progress := NewExportProgressReporter(false, false)
	processor := NewParallelFileProcessor(2, false, progress)

	files := []*vault.VaultFile{
		{RelativePath: "good.md", Body: "# Good File"},
		{RelativePath: "bad.md", Body: "# Bad File"},
		{RelativePath: "another.md", Body: "# Another File"},
	}

	filenameMap := map[string]string{
		"good.md":    "good.md",
		"bad.md":     "bad.md",
		"another.md": "another.md",
	}

	options := ExportOptions{
		OutputPath: "/tmp/test",
	}

	// Mock processor function that fails on "bad.md"
	fileProcessor := func(file *vault.VaultFile, outputPath string, opts ExportOptions) (*FileProcessingResult, error) {
		if file.RelativePath == "bad.md" {
			return nil, fmt.Errorf("simulated error processing %s", file.RelativePath)
		}
		return &FileProcessingResult{
			File:    file,
			Success: true,
		}, nil
	}

	ctx := context.Background()
	result, err := processor.ProcessFilesInParallel(ctx, files, filenameMap, options, fileProcessor)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "bad.md")
}

func TestParallelFileProcessor_ContextCancellation(t *testing.T) {
	progress := NewExportProgressReporter(false, false)
	processor := NewParallelFileProcessor(2, false, progress)

	// Create many files to ensure cancellation happens during processing
	files := make([]*vault.VaultFile, 100)
	filenameMap := make(map[string]string)

	for i := 0; i < 100; i++ {
		filename := fmt.Sprintf("file%d.md", i)
		files[i] = &vault.VaultFile{
			RelativePath: filename,
			Body:         fmt.Sprintf("# File %d", i),
		}
		filenameMap[filename] = filename
	}

	options := ExportOptions{
		OutputPath: "/tmp/test",
	}

	// Mock processor function that takes time
	fileProcessor := func(file *vault.VaultFile, outputPath string, opts ExportOptions) (*FileProcessingResult, error) {
		time.Sleep(10 * time.Millisecond) // Simulate work
		return &FileProcessingResult{
			File:    file,
			Success: true,
		}, nil
	}

	// Create context that will be cancelled quickly
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	result, err := processor.ProcessFilesInParallel(ctx, files, filenameMap, options, fileProcessor)

	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	assert.Nil(t, result)
}

func TestMemoryOptimizedFileProcessor(t *testing.T) {
	progress := NewExportProgressReporter(false, false)
	processor := NewMemoryOptimizedFileProcessor(progress)

	assert.Equal(t, 50, processor.batchSize)
	assert.Equal(t, progress, processor.progress)
}

func TestMemoryOptimizedFileProcessor_BatchProcessing(t *testing.T) {
	progress := NewExportProgressReporter(false, false)
	processor := NewMemoryOptimizedFileProcessor(progress)
	processor.batchSize = 3 // Small batch size for testing

	// Create files that exceed batch size
	files := make([]*vault.VaultFile, 7)
	filenameMap := make(map[string]string)

	for i := 0; i < 7; i++ {
		filename := fmt.Sprintf("file%d.md", i)
		files[i] = &vault.VaultFile{
			RelativePath: filename,
			Body:         fmt.Sprintf("# File %d", i),
		}
		filenameMap[filename] = filename
	}

	options := ExportOptions{
		OutputPath: "/tmp/test",
	}

	batchCount := 0
	totalProcessed := 0

	// Mock batch processor function
	batchProcessor := func(batch []*vault.VaultFile, fnMap map[string]string, opts ExportOptions) (*LinkProcessingResult, error) {
		batchCount++
		totalProcessed += len(batch)

		// Verify batch sizes
		if batchCount <= 2 {
			assert.Equal(t, 3, len(batch)) // First two batches should be full
		} else {
			assert.Equal(t, 1, len(batch)) // Last batch should have remainder
		}

		return &LinkProcessingResult{
			FilesWithLinksProcessed: len(batch),
		}, nil
	}

	ctx := context.Background()
	result, err := processor.ProcessFilesInBatches(ctx, files, filenameMap, options, batchProcessor)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 3, batchCount) // Should process 3 batches (3, 3, 1)
	assert.Equal(t, 7, totalProcessed)
	assert.Equal(t, 7, result.FilesWithLinksProcessed) // Sum of all batch results
}

func TestCalculatePerformanceMetrics(t *testing.T) {
	tests := []struct {
		name                string
		fileCount           int
		processingDuration  float64
		workerCount         int
		expectedFilesPerSec float64
	}{
		{
			name:                "Fast processing",
			fileCount:           100,
			processingDuration:  1.0,
			workerCount:         4,
			expectedFilesPerSec: 100.0,
		},
		{
			name:                "Slow processing",
			fileCount:           50,
			processingDuration:  10.0,
			workerCount:         2,
			expectedFilesPerSec: 5.0,
		},
		{
			name:                "Single file",
			fileCount:           1,
			processingDuration:  0.5,
			workerCount:         1,
			expectedFilesPerSec: 2.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := CalculatePerformanceMetrics(
				tt.fileCount,
				tt.processingDuration,
				tt.workerCount,
			)

			assert.Equal(t, tt.expectedFilesPerSec, metrics.FilesPerSecond)
			assert.Equal(t, tt.processingDuration, metrics.ProcessingTimeSec)
			assert.Equal(t, tt.workerCount, metrics.ParallelWorkers)
			assert.Greater(t, metrics.MemoryUsageMB, 0.0) // Should have some memory usage
		})
	}
}

func BenchmarkParallelFileProcessor(b *testing.B) {
	progress := NewExportProgressReporter(true, false) // Quiet mode for benchmarking

	// Test different worker counts
	workerCounts := []int{1, 2, 4, 8}
	fileCounts := []int{10, 50, 100}

	for _, workers := range workerCounts {
		for _, fileCount := range fileCounts {
			b.Run(fmt.Sprintf("Workers%d_Files%d", workers, fileCount), func(b *testing.B) {
				processor := NewParallelFileProcessor(workers, false, progress)

				// Create test files
				files := make([]*vault.VaultFile, fileCount)
				filenameMap := make(map[string]string)

				for i := 0; i < fileCount; i++ {
					filename := fmt.Sprintf("file%d.md", i)
					files[i] = &vault.VaultFile{
						RelativePath: filename,
						Body:         fmt.Sprintf("# File %d\nSome content here", i),
					}
					filenameMap[filename] = filename
				}

				options := ExportOptions{
					OutputPath: "/tmp/bench",
				}

				// Mock processor function
				fileProcessor := func(file *vault.VaultFile, outputPath string, opts ExportOptions) (*FileProcessingResult, error) {
					// Simulate some work
					_ = len(file.Body)
					return &FileProcessingResult{
						File:    file,
						Success: true,
					}, nil
				}

				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					ctx := context.Background()
					_, err := processor.ProcessFilesInParallel(ctx, files, filenameMap, options, fileProcessor)
					if err != nil {
						b.Fatalf("Benchmark failed: %v", err)
					}
				}
			})
		}
	}
}

func BenchmarkMemoryOptimizedProcessor(b *testing.B) {
	progress := NewExportProgressReporter(true, false) // Quiet mode for benchmarking
	processor := NewMemoryOptimizedFileProcessor(progress)

	fileCounts := []int{100, 500, 1000}

	for _, fileCount := range fileCounts {
		b.Run(fmt.Sprintf("Files%d", fileCount), func(b *testing.B) {
			// Create test files
			files := make([]*vault.VaultFile, fileCount)
			filenameMap := make(map[string]string)

			for i := 0; i < fileCount; i++ {
				filename := fmt.Sprintf("file%d.md", i)
				files[i] = &vault.VaultFile{
					RelativePath: filename,
					Body:         fmt.Sprintf("# File %d\nSome content here", i),
				}
				filenameMap[filename] = filename
			}

			options := ExportOptions{
				OutputPath: "/tmp/bench",
			}

			// Mock batch processor function
			batchProcessor := func(batch []*vault.VaultFile, fnMap map[string]string, opts ExportOptions) (*LinkProcessingResult, error) {
				return &LinkProcessingResult{
					FilesWithLinksProcessed: len(batch),
				}, nil
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				ctx := context.Background()
				_, err := processor.ProcessFilesInBatches(ctx, files, filenameMap, options, batchProcessor)
				if err != nil {
					b.Fatalf("Benchmark failed: %v", err)
				}
			}
		})
	}
}
