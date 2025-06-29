package workerpool

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWorkerPool(t *testing.T) {
	config := Config{
		MaxWorkers:  4,
		QueueSize:   20,
		TaskTimeout: 5 * time.Second,
		EnableStats: true,
	}

	pool := NewWorkerPool(config)
	defer pool.ForceShutdown()

	assert.Equal(t, 4, pool.workers)
	assert.Equal(t, 4, pool.Stats().PeakWorkers)
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Greater(t, config.MaxWorkers, 0)
	assert.Greater(t, config.QueueSize, 0)
	assert.Greater(t, config.TaskTimeout, time.Duration(0))
	assert.True(t, config.EnableStats)
}

func TestWorkerPoolSubmit(t *testing.T) {
	config := DefaultConfig()
	config.MaxWorkers = 2
	config.QueueSize = 5

	pool := NewWorkerPool(config)
	defer pool.ForceShutdown()

	var counter int64
	task := func(ctx context.Context) error {
		atomic.AddInt64(&counter, 1)
		return nil
	}

	// Submit tasks
	for i := 0; i < 3; i++ {
		err := pool.Submit(task)
		assert.NoError(t, err)
	}

	// Wait for completion
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int64(3), atomic.LoadInt64(&counter))
}

func TestWorkerPoolSubmitWithTimeout(t *testing.T) {
	config := DefaultConfig()
	config.QueueSize = 0  // No queue buffer
	config.MaxWorkers = 1 // Single worker

	pool := NewWorkerPool(config)
	defer pool.ForceShutdown()

	// Fill the worker with a blocking task
	slowTask := func(ctx context.Context) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	}

	// Submit task to occupy the worker
	err := pool.Submit(slowTask)
	assert.NoError(t, err)

	// Give worker time to start
	time.Sleep(10 * time.Millisecond)

	// This should timeout since worker is busy and no queue
	err = pool.SubmitWithTimeout(slowTask, 10*time.Millisecond)
	assert.Error(t, err)
	// Could be timeout or queue full error
}

func TestWorkerPoolResults(t *testing.T) {
	config := DefaultConfig()
	config.MaxWorkers = 1

	pool := NewWorkerPool(config)
	defer pool.ForceShutdown()

	testError := errors.New("test error")

	successTask := func(ctx context.Context) error {
		return nil
	}

	errorTask := func(ctx context.Context) error {
		return testError
	}

	// Submit tasks
	err := pool.Submit(successTask)
	require.NoError(t, err)

	err = pool.Submit(errorTask)
	require.NoError(t, err)

	// Collect results
	results := make([]TaskResult, 0, 2)
	for i := 0; i < 2; i++ {
		select {
		case result := <-pool.Results():
			results = append(results, result)
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for result")
		}
	}

	assert.Len(t, results, 2)

	// Check we got one success and one error
	var successCount, errorCount int
	for _, result := range results {
		if result.Error == nil {
			successCount++
			assert.True(t, result.Completed)
		} else {
			errorCount++
			assert.False(t, result.Completed)
			assert.Equal(t, testError, result.Error)
		}
	}

	assert.Equal(t, 1, successCount)
	assert.Equal(t, 1, errorCount)
}

func TestWorkerPoolStats(t *testing.T) {
	config := DefaultConfig()
	config.MaxWorkers = 2
	config.EnableStats = true

	pool := NewWorkerPool(config)
	defer pool.ForceShutdown()

	var taskCount int32
	task := func(ctx context.Context) error {
		atomic.AddInt32(&taskCount, 1)
		time.Sleep(10 * time.Millisecond)
		return nil
	}

	errorTask := func(ctx context.Context) error {
		return errors.New("task error")
	}

	// Submit tasks
	for i := 0; i < 3; i++ {
		err := pool.Submit(task)
		require.NoError(t, err)
	}

	err := pool.Submit(errorTask)
	require.NoError(t, err)

	// Wait for completion
	time.Sleep(200 * time.Millisecond)

	stats := pool.Stats()
	assert.Equal(t, int64(4), stats.TasksSubmitted)
	assert.Equal(t, int64(4), stats.TasksCompleted)
	assert.Equal(t, int64(1), stats.TasksFailed)
	assert.Greater(t, stats.TotalDuration, time.Duration(0))
	assert.Greater(t, stats.AverageDuration, time.Duration(0))
}

func TestWorkerPoolShutdown(t *testing.T) {
	config := DefaultConfig()
	config.MaxWorkers = 2

	pool := NewWorkerPool(config)

	var counter int64
	task := func(ctx context.Context) error {
		atomic.AddInt64(&counter, 1)
		time.Sleep(50 * time.Millisecond)
		return nil
	}

	// Submit some tasks
	for i := 0; i < 3; i++ {
		err := pool.Submit(task)
		require.NoError(t, err)
	}

	// Shutdown with timeout
	err := pool.Shutdown(500 * time.Millisecond)
	assert.NoError(t, err)

	// Verify tasks completed
	assert.Equal(t, int64(3), atomic.LoadInt64(&counter))
}

func TestWorkerPoolShutdownTimeout(t *testing.T) {
	config := DefaultConfig()
	config.MaxWorkers = 1

	pool := NewWorkerPool(config)

	longTask := func(ctx context.Context) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	}

	// Submit a long-running task
	err := pool.Submit(longTask)
	require.NoError(t, err)

	// Shutdown with short timeout
	err = pool.Shutdown(50 * time.Millisecond)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
}

func TestWorkerPoolForceShutdown(t *testing.T) {
	config := DefaultConfig()
	pool := NewWorkerPool(config)

	longTask := func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
			return nil
		}
	}

	// Submit task
	err := pool.Submit(longTask)
	require.NoError(t, err)

	// Force shutdown immediately
	pool.ForceShutdown()

	// Should not be able to submit more tasks
	err = pool.Submit(longTask)
	assert.Error(t, err)
}

func TestWorkerPoolWait(t *testing.T) {
	config := DefaultConfig()
	config.MaxWorkers = 2

	pool := NewWorkerPool(config)
	defer pool.ForceShutdown()

	var counter int64
	task := func(ctx context.Context) error {
		atomic.AddInt64(&counter, 1)
		time.Sleep(50 * time.Millisecond)
		return nil
	}

	// Submit tasks
	for i := 0; i < 4; i++ {
		err := pool.Submit(task)
		require.NoError(t, err)
	}

	// Close queue and wait
	close(pool.taskQueue)
	pool.Wait()

	assert.Equal(t, int64(4), atomic.LoadInt64(&counter))
}

func TestWorkerPoolWaitWithTimeout(t *testing.T) {
	config := DefaultConfig()
	config.MaxWorkers = 1

	pool := NewWorkerPool(config)
	defer pool.ForceShutdown()

	longTask := func(ctx context.Context) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	}

	// Submit task
	err := pool.Submit(longTask)
	require.NoError(t, err)

	// Wait with short timeout
	err = pool.WaitWithTimeout(50 * time.Millisecond)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestWorkerPoolProcessBatch(t *testing.T) {
	config := DefaultConfig()
	config.MaxWorkers = 2

	pool := NewWorkerPool(config)
	defer pool.ForceShutdown()

	var counter int64
	tasks := make([]Task, 5)
	for i := 0; i < 5; i++ {
		tasks[i] = func(ctx context.Context) error {
			atomic.AddInt64(&counter, 1)
			return nil
		}
	}

	results := pool.ProcessBatch(tasks)

	assert.Len(t, results, 5)
	assert.Equal(t, int64(5), atomic.LoadInt64(&counter))

	for _, result := range results {
		assert.NoError(t, result.Error)
		assert.True(t, result.Completed)
	}
}

func TestWorkerPoolProcessBatchWithProgress(t *testing.T) {
	config := DefaultConfig()
	config.MaxWorkers = 2

	pool := NewWorkerPool(config)
	defer pool.ForceShutdown()

	var counter int64
	tasks := make([]Task, 3)
	for i := 0; i < 3; i++ {
		tasks[i] = func(ctx context.Context) error {
			atomic.AddInt64(&counter, 1)
			time.Sleep(10 * time.Millisecond)
			return nil
		}
	}

	progress := make(chan int, 10)
	results := pool.ProcessBatchWithProgress(tasks, progress)
	close(progress)

	assert.Len(t, results, 3)
	assert.Equal(t, int64(3), atomic.LoadInt64(&counter))

	// Check progress updates
	progressUpdates := make([]int, 0)
	for p := range progress {
		progressUpdates = append(progressUpdates, p)
	}

	assert.Greater(t, len(progressUpdates), 0)
	assert.Equal(t, 3, progressUpdates[len(progressUpdates)-1]) // Final progress should be 3
}

func TestWorkerPoolTaskTimeout(t *testing.T) {
	config := DefaultConfig()
	config.MaxWorkers = 1
	config.TaskTimeout = 50 * time.Millisecond

	pool := NewWorkerPool(config)
	defer pool.ForceShutdown()

	timeoutTask := func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
			return nil
		}
	}

	err := pool.Submit(timeoutTask)
	require.NoError(t, err)

	// Wait for result
	select {
	case result := <-pool.Results():
		assert.Error(t, result.Error)
		assert.Equal(t, context.DeadlineExceeded, result.Error)
		assert.False(t, result.Completed)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for result")
	}
}

func TestWorkerPoolConcurrentAccess(t *testing.T) {
	config := DefaultConfig()
	config.MaxWorkers = 4
	config.QueueSize = 100

	pool := NewWorkerPool(config)
	defer pool.ForceShutdown()

	var counter int64
	task := func(ctx context.Context) error {
		atomic.AddInt64(&counter, 1)
		return nil
	}

	// Submit tasks concurrently
	const numGoroutines = 10
	const tasksPerGoroutine = 10

	done := make(chan struct{})
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for j := 0; j < tasksPerGoroutine; j++ {
				err := pool.Submit(task)
				assert.NoError(t, err)
			}
		}()
	}

	// Wait for all submissions
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	expectedTasks := int64(numGoroutines * tasksPerGoroutine)
	assert.Equal(t, expectedTasks, atomic.LoadInt64(&counter))

	stats := pool.Stats()
	assert.Equal(t, expectedTasks, stats.TasksSubmitted)
	assert.Equal(t, expectedTasks, stats.TasksCompleted)
	assert.Equal(t, int64(0), stats.TasksFailed)
}
