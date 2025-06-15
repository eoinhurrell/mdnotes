package processor

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// MemoryOptimizedProcessor provides memory-efficient processing for large vaults
type MemoryOptimizedProcessor struct {
	wrapped         Processor
	maxMemoryMB     int64
	chunkSize       int
	enableGC        bool
	gcInterval      time.Duration
	lastGC          time.Time
}

// MemoryOptimizedConfig configures memory optimization settings
type MemoryOptimizedConfig struct {
	MaxMemoryMB int64         // Maximum memory usage in MB (0 = no limit)
	ChunkSize   int           // Number of files to process per chunk
	EnableGC    bool          // Enable automatic garbage collection
	GCInterval  time.Duration // Minimum interval between GC calls
}

// NewMemoryOptimizedProcessor creates a memory-optimized processor
func NewMemoryOptimizedProcessor(wrapped Processor, config MemoryOptimizedConfig) *MemoryOptimizedProcessor {
	if config.ChunkSize <= 0 {
		config.ChunkSize = 50 // Conservative default for memory efficiency
	}
	if config.GCInterval <= 0 {
		config.GCInterval = 5 * time.Second
	}
	
	return &MemoryOptimizedProcessor{
		wrapped:     wrapped,
		maxMemoryMB: config.MaxMemoryMB,
		chunkSize:   config.ChunkSize,
		enableGC:    config.EnableGC,
		gcInterval:  config.GCInterval,
		lastGC:      time.Now(),
	}
}

// Process executes the wrapped processor with memory optimizations
func (p *MemoryOptimizedProcessor) Process(ctx context.Context, vault *Vault, params map[string]interface{}) error {
	// Monitor memory usage before starting
	initialMem := p.getMemoryUsage()
	
	// Process files in chunks to limit memory usage
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
		
		// Create chunk
		chunk := &Vault{
			Files: vault.Files[i:end],
			Path:  vault.Path,
		}
		
		// Check memory before processing chunk
		if p.maxMemoryMB > 0 {
			currentMem := p.getMemoryUsage()
			if currentMem > p.maxMemoryMB {
				p.forceGC()
				
				// Check again after GC
				currentMem = p.getMemoryUsage()
				if currentMem > p.maxMemoryMB {
					return fmt.Errorf("memory usage (%d MB) exceeds limit (%d MB)", currentMem, p.maxMemoryMB)
				}
			}
		}
		
		// Process chunk
		if err := p.wrapped.Process(ctx, chunk, params); err != nil {
			return fmt.Errorf("processing chunk %d-%d: %w", i, end-1, err)
		}
		
		// Optional garbage collection after chunk
		if p.enableGC && time.Since(p.lastGC) >= p.gcInterval {
			p.triggerGC()
		}
		
		// Clear references to processed files to help GC
		// Note: This assumes the processor doesn't need to maintain references
		for j := i; j < end; j++ {
			if j < len(vault.Files) {
				p.clearFileReferences(vault.Files[j])
			}
		}
	}
	
	finalMem := p.getMemoryUsage()
	
	// Log memory usage if verbose
	if verbose, ok := params["verbose"].(bool); ok && verbose {
		fmt.Printf("Memory usage: initial=%dMB, final=%dMB, peak=%dMB\n", 
			initialMem, finalMem, p.getPeakMemoryUsage())
	}
	
	return nil
}

// Name returns the name of the wrapped processor with memory optimization indicator
func (p *MemoryOptimizedProcessor) Name() string {
	return fmt.Sprintf("%s (memory-optimized)", p.wrapped.Name())
}

// getMemoryUsage returns current memory usage in MB
func (p *MemoryOptimizedProcessor) getMemoryUsage() int64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return int64(m.Alloc) / 1024 / 1024
}

// getPeakMemoryUsage returns peak memory usage in MB
func (p *MemoryOptimizedProcessor) getPeakMemoryUsage() int64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return int64(m.Sys) / 1024 / 1024
}

// triggerGC triggers garbage collection if enough time has passed
func (p *MemoryOptimizedProcessor) triggerGC() {
	if time.Since(p.lastGC) >= p.gcInterval {
		runtime.GC()
		p.lastGC = time.Now()
	}
}

// forceGC forces immediate garbage collection
func (p *MemoryOptimizedProcessor) forceGC() {
	runtime.GC()
	debug.FreeOSMemory()
	p.lastGC = time.Now()
}

// clearFileReferences clears memory-heavy references from a file
func (p *MemoryOptimizedProcessor) clearFileReferences(file *vault.VaultFile) {
	// Clear large content if not needed for subsequent operations
	// This is aggressive optimization - use with caution
	if len(file.Content) > 0 {
		file.Content = nil
	}
}

// StreamingProcessor processes files one at a time without keeping them all in memory
type StreamingProcessor struct {
	wrapped    Processor
	bufferSize int
}

// NewStreamingProcessor creates a processor that streams files from disk
func NewStreamingProcessor(wrapped Processor, bufferSize int) *StreamingProcessor {
	if bufferSize <= 0 {
		bufferSize = 1 // Process one file at a time
	}
	
	return &StreamingProcessor{
		wrapped:    wrapped,
		bufferSize: bufferSize,
	}
}

// Process executes the wrapped processor in streaming mode
func (p *StreamingProcessor) Process(ctx context.Context, vault *Vault, params map[string]interface{}) error {
	// For streaming, we process files one at a time or in small batches
	for i := 0; i < len(vault.Files); i += p.bufferSize {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		end := i + p.bufferSize
		if end > len(vault.Files) {
			end = len(vault.Files)
		}
		
		// Create a small batch
		batch := &Vault{
			Files: vault.Files[i:end],
			Path:  vault.Path,
		}
		
		if err := p.wrapped.Process(ctx, batch, params); err != nil {
			return fmt.Errorf("processing batch %d-%d: %w", i, end-1, err)
		}
	}
	
	return nil
}

// Name returns the name of the wrapped processor with streaming indicator
func (p *StreamingProcessor) Name() string {
	return fmt.Sprintf("%s (streaming)", p.wrapped.Name())
}

// ResourceMonitor monitors system resources during processing
type ResourceMonitor struct {
	maxCPUPercent   float64
	maxMemoryMB     int64
	checkInterval   time.Duration
	stopOnOverage   bool
}

// NewResourceMonitor creates a resource monitor
func NewResourceMonitor(maxCPUPercent float64, maxMemoryMB int64, checkInterval time.Duration, stopOnOverage bool) *ResourceMonitor {
	if checkInterval <= 0 {
		checkInterval = 1 * time.Second
	}
	
	return &ResourceMonitor{
		maxCPUPercent: maxCPUPercent,
		maxMemoryMB:   maxMemoryMB,
		checkInterval: checkInterval,
		stopOnOverage: stopOnOverage,
	}
}

// MonitoredProcessor wraps a processor with resource monitoring
type MonitoredProcessor struct {
	wrapped Processor
	monitor *ResourceMonitor
}

// NewMonitoredProcessor creates a processor with resource monitoring
func NewMonitoredProcessor(wrapped Processor, monitor *ResourceMonitor) *MonitoredProcessor {
	return &MonitoredProcessor{
		wrapped: wrapped,
		monitor: monitor,
	}
}

// Process executes the wrapped processor with resource monitoring
func (p *MonitoredProcessor) Process(ctx context.Context, vault *Vault, params map[string]interface{}) error {
	// Create a context with cancellation for resource monitoring
	monitorCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	
	// Start resource monitoring
	resourceErrors := make(chan error, 1)
	go p.monitorResources(monitorCtx, resourceErrors)
	
	// Execute the wrapped processor
	processingDone := make(chan error, 1)
	go func() {
		processingDone <- p.wrapped.Process(ctx, vault, params)
	}()
	
	// Wait for completion or resource limit violation
	select {
	case err := <-processingDone:
		return err
	case err := <-resourceErrors:
		cancel() // Stop processing
		return fmt.Errorf("resource limit exceeded: %w", err)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Name returns the name of the wrapped processor with monitoring indicator
func (p *MonitoredProcessor) Name() string {
	return fmt.Sprintf("%s (monitored)", p.wrapped.Name())
}

// monitorResources monitors system resources in a separate goroutine
func (p *MonitoredProcessor) monitorResources(ctx context.Context, errors chan<- error) {
	ticker := time.NewTicker(p.monitor.checkInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Check memory usage
			if p.monitor.maxMemoryMB > 0 {
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				currentMemMB := int64(m.Alloc) / 1024 / 1024
				
				if currentMemMB > p.monitor.maxMemoryMB {
					if p.monitor.stopOnOverage {
						errors <- fmt.Errorf("memory usage (%d MB) exceeds limit (%d MB)", currentMemMB, p.monitor.maxMemoryMB)
						return
					}
					
					// Try to free memory
					runtime.GC()
					debug.FreeOSMemory()
				}
			}
			
			// CPU monitoring would require additional libraries or OS-specific code
			// For now, we focus on memory monitoring
		}
	}
}

// ProcessorOptimizer automatically selects the best processing strategy
type ProcessorOptimizer struct {
	baseProcessor   Processor
	fileCountThresholds map[string]int
	memoryThresholds    map[string]int64
}

// NewProcessorOptimizer creates an optimizer that selects processing strategies
func NewProcessorOptimizer(baseProcessor Processor) *ProcessorOptimizer {
	return &ProcessorOptimizer{
		baseProcessor: baseProcessor,
		fileCountThresholds: map[string]int{
			"parallel":  50,   // Use parallel processing for 50+ files
			"chunked":   500,  // Use chunked processing for 500+ files
			"streaming": 5000, // Use streaming for 5000+ files
		},
		memoryThresholds: map[string]int64{
			"memory_optimized": 512, // Use memory optimization if available memory < 512MB
			"streaming":        256, // Use streaming if available memory < 256MB
		},
	}
}

// OptimizeProcessor returns the best processor for the given vault
func (o *ProcessorOptimizer) OptimizeProcessor(vault *Vault, config BatchConfig) Processor {
	fileCount := len(vault.Files)
	availableMemory := o.getAvailableMemoryMB()
	
	processor := o.baseProcessor
	
	// Apply memory optimization if needed
	if availableMemory < o.memoryThresholds["memory_optimized"] {
		processor = NewMemoryOptimizedProcessor(processor, MemoryOptimizedConfig{
			MaxMemoryMB: availableMemory / 2, // Use half available memory
			ChunkSize:   25,
			EnableGC:    true,
			GCInterval:  3 * time.Second,
		})
	}
	
	// Apply streaming for very large vaults or low memory
	if fileCount >= o.fileCountThresholds["streaming"] || availableMemory < o.memoryThresholds["streaming"] {
		processor = NewStreamingProcessor(processor, 10)
	} else if fileCount >= o.fileCountThresholds["chunked"] {
		processor = NewChunkedProcessor(processor, 100)
	} else if fileCount >= o.fileCountThresholds["parallel"] && config.Parallel {
		processor = NewParallelProcessor(processor, config.MaxWorkers)
	}
	
	return processor
}

// getAvailableMemoryMB estimates available memory in MB
func (o *ProcessorOptimizer) getAvailableMemoryMB() int64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	// Rough estimation: total system memory minus currently allocated
	// This is simplified - in production you'd want more accurate system memory detection
	usedMemMB := int64(m.Alloc) / 1024 / 1024
	totalMemMB := int64(m.Sys) / 1024 / 1024
	
	// Conservative estimate
	availableMB := totalMemMB - usedMemMB
	if availableMB < 128 {
		availableMB = 128 // Minimum assumption
	}
	
	return availableMB
}