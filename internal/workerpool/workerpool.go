package workerpool

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// Task represents a unit of work to be processed
type Task func(ctx context.Context) error

// TaskResult represents the result of a task execution
type TaskResult struct {
	Error     error
	Duration  time.Duration
	TaskID    int
	Completed bool
}

// WorkerPool manages a pool of workers for parallel task execution
type WorkerPool struct {
	workers    int
	taskQueue  chan Task
	resultChan chan TaskResult
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	stats      Stats
	mu         sync.RWMutex
	closeOnce  sync.Once
	closed     bool
}

// Stats tracks worker pool performance metrics
type Stats struct {
	TasksSubmitted  int64
	TasksCompleted  int64
	TasksFailed     int64
	TotalDuration   time.Duration
	AverageDuration time.Duration
	PeakWorkers     int
}

// Config holds worker pool configuration
type Config struct {
	MaxWorkers  int
	QueueSize   int
	TaskTimeout time.Duration
	EnableStats bool
}

// DefaultConfig returns sensible defaults for worker pool configuration
func DefaultConfig() Config {
	return Config{
		MaxWorkers:  runtime.NumCPU(),
		QueueSize:   runtime.NumCPU() * 10,
		TaskTimeout: 30 * time.Second,
		EnableStats: true,
	}
}

// NewWorkerPool creates a new worker pool with the given configuration
func NewWorkerPool(config Config) *WorkerPool {
	if config.MaxWorkers <= 0 {
		config.MaxWorkers = runtime.NumCPU()
	}
	if config.QueueSize <= 0 {
		config.QueueSize = config.MaxWorkers * 10
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool := &WorkerPool{
		workers:    config.MaxWorkers,
		taskQueue:  make(chan Task, config.QueueSize),
		resultChan: make(chan TaskResult, config.QueueSize),
		ctx:        ctx,
		cancel:     cancel,
		stats:      Stats{PeakWorkers: config.MaxWorkers},
	}

	// Start workers
	for i := 0; i < config.MaxWorkers; i++ {
		pool.wg.Add(1)
		go pool.worker(i, config.TaskTimeout, config.EnableStats)
	}

	return pool
}

// Submit adds a task to the worker pool queue
func (wp *WorkerPool) Submit(task Task) error {
	wp.mu.RLock()
	if wp.closed {
		wp.mu.RUnlock()
		return fmt.Errorf("worker pool is closed")
	}
	wp.mu.RUnlock()

	select {
	case wp.taskQueue <- task:
		wp.mu.Lock()
		wp.stats.TasksSubmitted++
		wp.mu.Unlock()
		return nil
	case <-wp.ctx.Done():
		return fmt.Errorf("worker pool is shutting down")
	default:
		return fmt.Errorf("task queue is full")
	}
}

// SubmitWithTimeout adds a task with a timeout for queue submission
func (wp *WorkerPool) SubmitWithTimeout(task Task, timeout time.Duration) error {
	wp.mu.RLock()
	if wp.closed {
		wp.mu.RUnlock()
		return fmt.Errorf("worker pool is closed")
	}
	wp.mu.RUnlock()

	ctx, cancel := context.WithTimeout(wp.ctx, timeout)
	defer cancel()

	select {
	case wp.taskQueue <- task:
		wp.mu.Lock()
		wp.stats.TasksSubmitted++
		wp.mu.Unlock()
		return nil
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("timeout submitting task to queue")
		}
		return fmt.Errorf("worker pool is shutting down")
	}
}

// Results returns a channel for receiving task results
func (wp *WorkerPool) Results() <-chan TaskResult {
	return wp.resultChan
}

// Stats returns current worker pool statistics
func (wp *WorkerPool) Stats() Stats {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	stats := wp.stats
	if stats.TasksCompleted > 0 {
		stats.AverageDuration = time.Duration(int64(stats.TotalDuration) / stats.TasksCompleted)
	}
	return stats
}

// Shutdown gracefully shuts down the worker pool
func (wp *WorkerPool) Shutdown(timeout time.Duration) error {
	// Stop accepting new tasks
	close(wp.taskQueue)

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		wp.cancel()
		close(wp.resultChan)
		return nil
	case <-time.After(timeout):
		wp.cancel()
		close(wp.resultChan)
		return fmt.Errorf("worker pool shutdown timed out after %v", timeout)
	}
}

// ForceShutdown immediately cancels all workers
func (wp *WorkerPool) ForceShutdown() {
	wp.mu.Lock()
	if wp.closed {
		wp.mu.Unlock()
		return
	}
	wp.closed = true
	wp.mu.Unlock()

	// Cancel context first to stop workers
	wp.cancel()

	// Then close channels safely
	wp.closeOnce.Do(func() {
		// Close task queue first to stop new submissions
		defer func() {
			if r := recover(); r != nil {
				// Channel already closed, ignore
			}
		}()
		close(wp.taskQueue)

		// Wait a bit for workers to finish current tasks
		time.Sleep(10 * time.Millisecond)

		// Close result channel
		defer func() {
			if r := recover(); r != nil {
				// Channel already closed, ignore
			}
		}()
		close(wp.resultChan)
	})
}

// Wait blocks until all submitted tasks are completed
func (wp *WorkerPool) Wait() {
	wp.wg.Wait()
}

// WaitWithTimeout waits for completion with a timeout
func (wp *WorkerPool) WaitWithTimeout(timeout time.Duration) error {
	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for workers to complete")
	}
}

// worker runs in a goroutine and processes tasks from the queue
func (wp *WorkerPool) worker(id int, taskTimeout time.Duration, enableStats bool) {
	defer wp.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			// Handle any panics from sending on closed channels
		}
	}()

	for {
		select {
		case task, ok := <-wp.taskQueue:
			if !ok {
				return // Queue closed, worker exits
			}

			// Create task context with timeout
			taskCtx, cancel := context.WithTimeout(wp.ctx, taskTimeout)

			start := time.Now()
			var err error

			// Execute task
			err = task(taskCtx)
			duration := time.Since(start)

			cancel()

			// Update statistics
			if enableStats {
				wp.mu.Lock()
				wp.stats.TasksCompleted++
				wp.stats.TotalDuration += duration
				if err != nil {
					wp.stats.TasksFailed++
				}
				wp.mu.Unlock()
			}

			// Send result (non-blocking)
			result := TaskResult{
				Error:     err,
				Duration:  duration,
				TaskID:    id,
				Completed: err == nil,
			}

			// Try to send result (non-blocking)
			select {
			case wp.resultChan <- result:
				// Result sent successfully
			case <-wp.ctx.Done():
				return
			default:
				// Result channel full or closed, skip (we don't want to block workers)
			}

		case <-wp.ctx.Done():
			return // Pool shutdown
		}
	}
}

// ProcessBatch processes a batch of tasks and returns when all are complete
func (wp *WorkerPool) ProcessBatch(tasks []Task) []TaskResult {
	results := make([]TaskResult, 0, len(tasks))
	resultCount := 0

	// Submit all tasks
	for _, task := range tasks {
		if err := wp.Submit(task); err != nil {
			results = append(results, TaskResult{
				Error:     err,
				Completed: false,
			})
			resultCount++
		}
	}

	// Collect results
loop:
	for resultCount < len(tasks) {
		select {
		case result := <-wp.resultChan:
			results = append(results, result)
			resultCount++
		case <-wp.ctx.Done():
			break loop
		}
	}

	return results
}

// ProcessBatchWithProgress processes tasks with progress reporting
func (wp *WorkerPool) ProcessBatchWithProgress(tasks []Task, progress chan<- int) []TaskResult {
	results := make([]TaskResult, 0, len(tasks))
	resultCount := 0

	// Submit all tasks
	for _, task := range tasks {
		if err := wp.Submit(task); err != nil {
			results = append(results, TaskResult{
				Error:     err,
				Completed: false,
			})
			resultCount++
			if progress != nil {
				select {
				case progress <- resultCount:
				default:
					// Non-blocking send - skip if channel is full
				}
			}
		}
	}

	// Collect results with progress updates
progressLoop:
	for resultCount < len(tasks) {
		select {
		case result := <-wp.resultChan:
			results = append(results, result)
			resultCount++
			if progress != nil {
				select {
				case progress <- resultCount:
				default:
					// Non-blocking send - skip if channel is full
				}
			}
		case <-wp.ctx.Done():
			break progressLoop
		}
	}

	return results
}
