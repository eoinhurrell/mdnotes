package processor

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// ParallelProcessor wraps existing processors to provide parallel execution
type ParallelProcessor struct {
	wrapped    Processor
	maxWorkers int
}

// NewParallelProcessor creates a processor that executes the wrapped processor in parallel
func NewParallelProcessor(wrapped Processor, maxWorkers int) *ParallelProcessor {
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
	}

	return &ParallelProcessor{
		wrapped:    wrapped,
		maxWorkers: maxWorkers,
	}
}

// Process executes the wrapped processor on all files in parallel
func (p *ParallelProcessor) Process(ctx context.Context, vault *Vault, params map[string]interface{}) error {
	// Check if this operation can be parallelized safely
	if !p.isParallelizable(params) {
		return p.wrapped.Process(ctx, vault, params)
	}

	// Create worker pool
	jobs := make(chan *Vault, len(vault.Files))
	errors := make(chan error, len(vault.Files))

	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < p.maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for singleFileVault := range jobs {
				select {
				case <-ctx.Done():
					errors <- ctx.Err()
					return
				default:
				}

				if err := p.wrapped.Process(ctx, singleFileVault, params); err != nil {
					errors <- fmt.Errorf("processing files: %w", err)
					return
				}
			}
		}()
	}

	// Send jobs (each job is a single-file vault)
	go func() {
		defer close(jobs)
		for i := range vault.Files {
			singleFileVault := &Vault{Files: vault.Files[i : i+1], Path: vault.Path}
			select {
			case jobs <- singleFileVault:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for completion
	go func() {
		wg.Wait()
		close(errors)
	}()

	// Collect errors
	for err := range errors {
		if err != nil {
			return err
		}
	}

	return nil
}

// Name returns the name of the wrapped processor with parallel indicator
func (p *ParallelProcessor) Name() string {
	return fmt.Sprintf("%s (parallel)", p.wrapped.Name())
}

// isParallelizable determines if an operation can be safely parallelized
func (p *ParallelProcessor) isParallelizable(params map[string]interface{}) bool {
	// Dry run operations are always safe to parallelize
	if dryRun, ok := params["dry_run"].(bool); ok && dryRun {
		return true
	}

	// Most frontmatter operations are safe since they modify individual files
	// Operations that modify file relationships (like link updates) may not be safe
	return true
}

// BatchProcessorV2 provides parallel processing support
type BatchProcessorV2 struct {
	*BatchProcessor
	enableParallel bool
	maxWorkers     int
}

// NewBatchProcessorV2 creates a batch processor with parallel processing
func NewBatchProcessorV2(maxWorkers int, enableParallel bool) *BatchProcessorV2 {
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
	}

	return &BatchProcessorV2{
		BatchProcessor: NewBatchProcessor(),
		enableParallel: enableParallel,
		maxWorkers:     maxWorkers,
	}
}

// Execute executes a batch configuration with optional parallel processing
func (bp *BatchProcessorV2) Execute(ctx context.Context, vault *Vault, config BatchConfig) ([]OperationResult, error) {
	// Use parallel processing if enabled and the config allows it
	if bp.enableParallel && config.Parallel && config.MaxWorkers > 1 {
		return bp.executeParallel(ctx, vault, config)
	}

	// Fall back to sequential processing
	return bp.BatchProcessor.Execute(ctx, vault, config)
}

// executeParallel executes operations with parallel file processing
func (bp *BatchProcessorV2) executeParallel(ctx context.Context, vault *Vault, config BatchConfig) ([]OperationResult, error) {
	var results []OperationResult

	// Create backup if requested
	if config.CreateBackup && !config.DryRun {
		backup, err := bp.createBackup(vault)
		if err != nil {
			return nil, fmt.Errorf("creating backup: %w", err)
		}
		bp.lastBackup = backup
	}

	// Execute operations with parallel processing where possible
	for i, operation := range config.Operations {
		select {
		case <-ctx.Done():
			if bp.lastBackup != nil && !config.DryRun {
				bp.rollback(vault, bp.lastBackup)
			}
			return results, ctx.Err()
		default:
		}

		result := bp.executeOperationParallel(ctx, vault, operation, config)
		results = append(results, result)

		if !result.Success && config.StopOnError {
			if bp.lastBackup != nil && !config.DryRun {
				bp.rollback(vault, bp.lastBackup)
			}
			return results, fmt.Errorf("operation %d (%s) failed: %w", i, operation.Name, result.Error)
		}
	}

	return results, nil
}

// executeOperationParallel executes a single operation with parallel processing
func (bp *BatchProcessorV2) executeOperationParallel(ctx context.Context, vault *Vault, op Operation, config BatchConfig) OperationResult {
	start := time.Now()

	result := OperationResult{
		Operation: op.Command,
	}

	// Find processor
	processor, exists := bp.processors[op.Command]
	if !exists {
		result.Success = false
		result.Error = fmt.Errorf("unknown command: %s", op.Command)
		result.Duration = time.Since(start)
		return result
	}

	// Wrap processor for parallel execution if beneficial
	workers := config.MaxWorkers
	if workers > len(vault.Files) {
		workers = len(vault.Files)
	}

	if workers > 1 && bp.shouldParallelize(op, len(vault.Files)) {
		processor = NewParallelProcessor(processor, workers)
	}

	// Prepare parameters
	params := make(map[string]interface{})
	for k, v := range op.Parameters {
		params[k] = v
	}
	if config.DryRun {
		params["dry_run"] = true
	}

	// Execute with retry logic
	retryCount := op.RetryCount
	if retryCount == 0 {
		retryCount = 1
	}

	var lastErr error
	for attempt := 0; attempt < retryCount; attempt++ {
		if attempt > 0 {
			delay := 1 * time.Second
			if op.RetryDelay != "" {
				if d, err := time.ParseDuration(op.RetryDelay); err == nil {
					delay = d
				}
			}

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				result.Success = false
				result.Error = ctx.Err()
				result.Duration = time.Since(start)
				return result
			}
		}

		err := processor.Process(ctx, vault, params)
		if err == nil {
			result.Success = true
			result.FilesAffected = len(vault.Files)
			if config.DryRun {
				result.Message = fmt.Sprintf("would execute %s on %d files", op.Name, len(vault.Files))
			} else {
				result.Message = fmt.Sprintf("executed %s on %d files", op.Name, len(vault.Files))
			}
			break
		}

		lastErr = err
		if attempt == retryCount-1 {
			result.Success = false
			result.Error = lastErr
		}
	}

	result.Duration = time.Since(start)
	return result
}

// shouldParallelize determines if an operation should be parallelized
func (bp *BatchProcessorV2) shouldParallelize(op Operation, fileCount int) bool {
	// Don't parallelize for small numbers of files
	if fileCount < 10 {
		return false
	}

	// Operations that are CPU-intensive benefit from parallelization
	cpuIntensiveOps := map[string]bool{
		"frontmatter.ensure":   true,
		"frontmatter.cast":     true,
		"frontmatter.validate": true,
		"headings.fix":         true,
		"headings.analyze":     true,
		"links.check":          true,
	}

	// Operations that modify relationships might not be safe to parallelize
	relationshipOps := map[string]bool{
		"links.convert": false, // Link conversion might affect cross-references
	}

	if safe, exists := relationshipOps[op.Command]; exists {
		return safe
	}

	return cpuIntensiveOps[op.Command]
}

// ChunkedProcessor processes files in chunks for better memory management
type ChunkedProcessor struct {
	wrapped   Processor
	chunkSize int
}

// NewChunkedProcessor creates a processor that processes files in chunks
func NewChunkedProcessor(wrapped Processor, chunkSize int) *ChunkedProcessor {
	if chunkSize <= 0 {
		chunkSize = 100 // Default chunk size
	}

	return &ChunkedProcessor{
		wrapped:   wrapped,
		chunkSize: chunkSize,
	}
}

// Process executes the wrapped processor on files in chunks
func (p *ChunkedProcessor) Process(ctx context.Context, vault *Vault, params map[string]interface{}) error {
	for i := 0; i < len(vault.Files); i += p.chunkSize {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		end := i + p.chunkSize
		if end > len(vault.Files) {
			end = len(vault.Files)
		}

		chunk := &Vault{
			Files: vault.Files[i:end],
			Path:  vault.Path,
		}

		if err := p.wrapped.Process(ctx, chunk, params); err != nil {
			return fmt.Errorf("processing chunk %d-%d: %w", i, end-1, err)
		}
	}

	return nil
}

// Name returns the name of the wrapped processor with chunk indicator
func (p *ChunkedProcessor) Name() string {
	return fmt.Sprintf("%s (chunked)", p.wrapped.Name())
}
