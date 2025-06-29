package processor

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// ParallelFileProcessor handles parallel processing of files during export
type ParallelFileProcessor struct {
	workerCount    int
	optimizeMemory bool
	progress       *ExportProgressReporter
}

// NewParallelFileProcessor creates a new parallel file processor
func NewParallelFileProcessor(workerCount int, optimizeMemory bool, progress *ExportProgressReporter) *ParallelFileProcessor {
	// Auto-detect worker count if not specified
	if workerCount <= 0 {
		workerCount = runtime.NumCPU()
		// For export operations, limit to reasonable number of workers
		if workerCount > 8 {
			workerCount = 8
		}
	}

	return &ParallelFileProcessor{
		workerCount:    workerCount,
		optimizeMemory: optimizeMemory,
		progress:       progress,
	}
}

// FileProcessingJob represents a file processing job
type FileProcessingJob struct {
	File     *vault.VaultFile
	Index    int
	FilePath string
}

// FileProcessingResult represents the result of processing a file
type FileProcessingResult struct {
	Index   int
	File    *vault.VaultFile
	Success bool
	Error   error
	// Link processing statistics for this file
	ExternalLinksRemoved   int
	ExternalLinksConverted int
	InternalLinksUpdated   int
	LinksProcessed         int
}

// ProcessFilesInParallel processes files using parallel workers
func (pfp *ParallelFileProcessor) ProcessFilesInParallel(
	ctx context.Context,
	files []*vault.VaultFile,
	filenameMap map[string]string,
	options ExportOptions,
	processor func(*vault.VaultFile, string, ExportOptions) (*FileProcessingResult, error),
) (*LinkProcessingResult, error) {

	if len(files) == 0 {
		return &LinkProcessingResult{}, nil
	}

	// For small numbers of files, don't use parallel processing
	if len(files) < pfp.workerCount*2 {
		return pfp.processFilesSequentially(ctx, files, filenameMap, options, processor)
	}

	// Create job channel and result channel
	jobs := make(chan FileProcessingJob, len(files))
	results := make(chan FileProcessingResult, len(files))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < pfp.workerCount; i++ {
		wg.Add(1)
		go pfp.worker(ctx, &wg, jobs, results, filenameMap, options, processor)
	}

	// Send jobs
	go func() {
		defer close(jobs)
		for i, file := range files {
			select {
			case jobs <- FileProcessingJob{
				File:     file,
				Index:    i,
				FilePath: filenameMap[file.RelativePath],
			}:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Close results channel when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	return pfp.collectResults(ctx, results, len(files))
}

// worker is a worker goroutine that processes files
func (pfp *ParallelFileProcessor) worker(
	ctx context.Context,
	wg *sync.WaitGroup,
	jobs <-chan FileProcessingJob,
	results chan<- FileProcessingResult,
	filenameMap map[string]string,
	options ExportOptions,
	processor func(*vault.VaultFile, string, ExportOptions) (*FileProcessingResult, error),
) {
	defer wg.Done()

	for {
		select {
		case job, ok := <-jobs:
			if !ok {
				return
			}

			// Process the file
			result, err := processor(job.File, job.FilePath, options)
			if err != nil {
				results <- FileProcessingResult{
					Index:   job.Index,
					File:    job.File,
					Success: false,
					Error:   err,
				}
			} else {
				results <- FileProcessingResult{
					Index:                  job.Index,
					File:                   job.File,
					Success:                true,
					ExternalLinksRemoved:   result.ExternalLinksRemoved,
					ExternalLinksConverted: result.ExternalLinksConverted,
					InternalLinksUpdated:   result.InternalLinksUpdated,
					LinksProcessed:         result.LinksProcessed,
				}
			}

		case <-ctx.Done():
			return
		}
	}
}

// collectResults collects and aggregates results from parallel processing
func (pfp *ParallelFileProcessor) collectResults(
	ctx context.Context,
	results <-chan FileProcessingResult,
	expectedCount int,
) (*LinkProcessingResult, error) {

	linkResult := &LinkProcessingResult{}
	processedCount := 0

	for {
		select {
		case result, ok := <-results:
			if !ok {
				// Channel closed, we're done
				return linkResult, nil
			}

			processedCount++

			if !result.Success {
				return nil, fmt.Errorf("processing file %s: %w",
					result.File.RelativePath, result.Error)
			}

			// Aggregate statistics
			linkResult.ExternalLinksRemoved += result.ExternalLinksRemoved
			linkResult.ExternalLinksConverted += result.ExternalLinksConverted
			linkResult.InternalLinksUpdated += result.InternalLinksUpdated
			if result.LinksProcessed > 0 {
				linkResult.FilesWithLinksProcessed++
			}

			// Update progress
			pfp.progress.UpdatePhase(processedCount,
				fmt.Sprintf("Processed: %s", result.File.RelativePath))

			// Check if we've processed all files
			if processedCount >= expectedCount {
				return linkResult, nil
			}

		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// processFilesSequentially processes files sequentially (fallback for small file counts)
func (pfp *ParallelFileProcessor) processFilesSequentially(
	ctx context.Context,
	files []*vault.VaultFile,
	filenameMap map[string]string,
	options ExportOptions,
	processor func(*vault.VaultFile, string, ExportOptions) (*FileProcessingResult, error),
) (*LinkProcessingResult, error) {

	linkResult := &LinkProcessingResult{}

	for i, file := range files {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		filePath := filenameMap[file.RelativePath]
		result, err := processor(file, filePath, options)
		if err != nil {
			return nil, fmt.Errorf("processing file %s: %w", file.RelativePath, err)
		}

		// Aggregate statistics
		linkResult.ExternalLinksRemoved += result.ExternalLinksRemoved
		linkResult.ExternalLinksConverted += result.ExternalLinksConverted
		linkResult.InternalLinksUpdated += result.InternalLinksUpdated
		if result.LinksProcessed > 0 {
			linkResult.FilesWithLinksProcessed++
		}

		// Update progress
		pfp.progress.UpdatePhase(i+1, fmt.Sprintf("Processed: %s", file.RelativePath))
	}

	return linkResult, nil
}

// MemoryOptimizedFileProcessor provides memory-efficient file processing
type MemoryOptimizedFileProcessor struct {
	batchSize int
	progress  *ExportProgressReporter
}

// NewMemoryOptimizedFileProcessor creates a new memory-optimized processor
func NewMemoryOptimizedFileProcessor(progress *ExportProgressReporter) *MemoryOptimizedFileProcessor {
	return &MemoryOptimizedFileProcessor{
		batchSize: 50, // Process files in batches of 50
		progress:  progress,
	}
}

// ProcessFilesInBatches processes files in memory-efficient batches
func (mofp *MemoryOptimizedFileProcessor) ProcessFilesInBatches(
	ctx context.Context,
	files []*vault.VaultFile,
	filenameMap map[string]string,
	options ExportOptions,
	processor func([]*vault.VaultFile, map[string]string, ExportOptions) (*LinkProcessingResult, error),
) (*LinkProcessingResult, error) {

	totalResult := &LinkProcessingResult{}
	processedCount := 0

	for i := 0; i < len(files); i += mofp.batchSize {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Calculate batch boundaries
		end := i + mofp.batchSize
		if end > len(files) {
			end = len(files)
		}

		// Process batch
		batch := files[i:end]
		batchResult, err := processor(batch, filenameMap, options)
		if err != nil {
			return nil, fmt.Errorf("processing batch %d-%d: %w", i, end-1, err)
		}

		// Aggregate results
		totalResult.ExternalLinksRemoved += batchResult.ExternalLinksRemoved
		totalResult.ExternalLinksConverted += batchResult.ExternalLinksConverted
		totalResult.InternalLinksUpdated += batchResult.InternalLinksUpdated
		totalResult.FilesWithLinksProcessed += batchResult.FilesWithLinksProcessed

		processedCount += len(batch)
		mofp.progress.UpdatePhase(processedCount,
			fmt.Sprintf("Processed batch %d-%d", i+1, end))
	}

	return totalResult, nil
}

// PerformanceMetrics tracks performance metrics during export
type PerformanceMetrics struct {
	FilesPerSecond    float64
	MemoryUsageMB     float64
	ProcessingTimeSec float64
	ParallelWorkers   int
}

// CalculatePerformanceMetrics calculates performance metrics for the export
func CalculatePerformanceMetrics(
	fileCount int,
	processingDuration float64,
	workerCount int,
) PerformanceMetrics {

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return PerformanceMetrics{
		FilesPerSecond:    float64(fileCount) / processingDuration,
		MemoryUsageMB:     float64(memStats.Alloc) / 1024 / 1024,
		ProcessingTimeSec: processingDuration,
		ParallelWorkers:   workerCount,
	}
}
