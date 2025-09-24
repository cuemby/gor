package queue

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Helper function to create a test queue with temporary database
func setupTestQueue(t *testing.T) *SolidQueue {
	tmpFile, err := os.CreateTemp("", "queue_test_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp database: %v", err)
	}
	tmpFile.Close()

	// Clean up database file after test
	t.Cleanup(func() {
		os.Remove(tmpFile.Name())
	})

	queue, err := NewSolidQueue(tmpFile.Name(), 2) // 2 workers for testing
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}

	// Clean up queue after test
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = queue.Stop(ctx)
	})

	return queue
}

func TestNewSolidQueue(t *testing.T) {
	t.Run("ValidDatabase", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "queue_test_*.db")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		queue, err := NewSolidQueue(tmpFile.Name(), 3)
		if err != nil {
			t.Fatalf("NewSolidQueue() should not return error: %v", err)
		}
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			_ = queue.Stop(ctx)
		}()

		if queue.workers != 3 {
			t.Errorf("Expected 3 workers, got %d", queue.workers)
		}

		if queue.pollInterval != time.Second {
			t.Errorf("Expected 1 second poll interval, got %v", queue.pollInterval)
		}
	})

	t.Run("DefaultWorkers", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "queue_test_*.db")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		queue, err := NewSolidQueue(tmpFile.Name(), 0) // Should default to 5
		if err != nil {
			t.Fatalf("NewSolidQueue() should not return error: %v", err)
		}
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			_ = queue.Stop(ctx)
		}()

		if queue.workers != 5 {
			t.Errorf("Expected 5 default workers, got %d", queue.workers)
		}
	})

	t.Run("InvalidDatabasePath", func(t *testing.T) {
		_, err := NewSolidQueue("/invalid/path/database.db", 1)
		if err == nil {
			t.Error("NewSolidQueue() should return error for invalid database path")
		}
	})
}

func TestSolidQueue_Enqueue(t *testing.T) {
	queue := setupTestQueue(t)

	t.Run("BasicEnqueue", func(t *testing.T) {
		job := &Job{
			Handler: "test_handler",
			Payload: map[string]string{"message": "hello world"},
		}

		err := queue.Enqueue(job)
		if err != nil {
			t.Fatalf("Enqueue() should not return error: %v", err)
		}

		if job.ID == "" {
			t.Error("Job ID should be set after enqueue")
		}

		if job.Queue != "default" {
			t.Errorf("Expected default queue, got %s", job.Queue)
		}

		if job.MaxAttempts != 3 {
			t.Errorf("Expected 3 max attempts, got %d", job.MaxAttempts)
		}
	})

	t.Run("CustomQueue", func(t *testing.T) {
		job := &Job{
			Handler:     "test_handler",
			Queue:       "custom_queue",
			Payload:     "test payload",
			MaxAttempts: 5,
		}

		err := queue.Enqueue(job)
		if err != nil {
			t.Fatalf("Enqueue() should not return error: %v", err)
		}

		if job.Queue != "custom_queue" {
			t.Errorf("Expected custom_queue, got %s", job.Queue)
		}

		if job.MaxAttempts != 5 {
			t.Errorf("Expected 5 max attempts, got %d", job.MaxAttempts)
		}
	})

	t.Run("InvalidPayload", func(t *testing.T) {
		job := &Job{
			Handler: "test_handler",
			Payload: make(chan int), // Unmarshallable type
		}

		err := queue.Enqueue(job)
		if err == nil {
			t.Error("Enqueue() should return error for invalid payload")
		}
	})
}

func TestSolidQueue_EnqueueAt(t *testing.T) {
	queue := setupTestQueue(t)

	future := time.Now().Add(1 * time.Hour)
	job := &Job{
		Handler: "test_handler",
		Payload: "delayed job",
	}

	err := queue.EnqueueAt(job, future)
	if err != nil {
		t.Fatalf("EnqueueAt() should not return error: %v", err)
	}

	// Check that scheduled time is set correctly
	timeDiff := job.ScheduledAt.Sub(future)
	if timeDiff > time.Second || timeDiff < -time.Second {
		t.Errorf("Scheduled time not set correctly, diff: %v", timeDiff)
	}
}

func TestSolidQueue_EnqueueIn(t *testing.T) {
	queue := setupTestQueue(t)

	delay := 30 * time.Minute
	beforeEnqueue := time.Now()

	job := &Job{
		Handler: "test_handler",
		Payload: "delayed job",
	}

	err := queue.EnqueueIn(job, delay)
	if err != nil {
		t.Fatalf("EnqueueIn() should not return error: %v", err)
	}

	expectedTime := beforeEnqueue.Add(delay)
	timeDiff := job.ScheduledAt.Sub(expectedTime)

	// Allow for some time variance (should be within 1 second)
	if timeDiff > time.Second || timeDiff < -time.Second {
		t.Errorf("Scheduled time not set correctly, diff: %v", timeDiff)
	}
}

func TestSolidQueue_RegisterHandler(t *testing.T) {
	queue := setupTestQueue(t)

	handler := func(ctx *JobContext) error {
		return nil
	}

	queue.RegisterHandler("test_handler", handler)

	// Verify handler is registered
	queue.mu.RLock()
	_, exists := queue.handlers["test_handler"]
	queue.mu.RUnlock()

	if !exists {
		t.Error("Handler should be registered")
	}
}

func TestSolidQueue_ProcessJob(t *testing.T) {
	queue := setupTestQueue(t)

	t.Run("SuccessfulJob", func(t *testing.T) {
		var processedPayload interface{}
		var processedContext *JobContext

		queue.RegisterHandler("success_handler", func(ctx *JobContext) error {
			processedPayload = ctx.Payload
			processedContext = ctx
			return nil
		})

		job := &Job{
			Handler: "success_handler",
			Payload: map[string]string{"key": "value"},
		}

		err := queue.Enqueue(job)
		if err != nil {
			t.Fatalf("Failed to enqueue job: %v", err)
		}

		// Start queue processing
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		_ = queue.Start(ctx)

		// Wait for job to be processed
		time.Sleep(500 * time.Millisecond)

		if processedPayload == nil {
			t.Error("Job should have been processed")
		}

		if processedContext == nil {
			t.Error("Job context should be set")
		} else {
			if processedContext.Handler != "success_handler" {
				t.Errorf("Expected handler 'success_handler', got %s", processedContext.Handler)
			}
			if processedContext.Queue != "default" {
				t.Errorf("Expected queue 'default', got %s", processedContext.Queue)
			}
		}
	})

	t.Run("FailingJob", func(t *testing.T) {
		queue.RegisterHandler("failing_handler", func(ctx *JobContext) error {
			return fmt.Errorf("job failed")
		})

		job := &Job{
			Handler:     "failing_handler",
			Payload:     "failing job",
			MaxAttempts: 1, // Fail immediately
		}

		err := queue.Enqueue(job)
		if err != nil {
			t.Fatalf("Failed to enqueue job: %v", err)
		}

		// Start queue processing
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		_ = queue.Start(ctx)

		// Wait for job to be processed
		time.Sleep(500 * time.Millisecond)

		// Check job status in database
		var status string
		query := "SELECT status FROM jobs WHERE id = ?"
		err = queue.db.QueryRow(query, job.ID).Scan(&status)
		if err != nil {
			t.Fatalf("Failed to query job status: %v", err)
		}

		if status != JobStatusFailed {
			t.Errorf("Expected job status to be '%s', got '%s'", JobStatusFailed, status)
		}
	})

	t.Run("UnknownHandler", func(t *testing.T) {
		job := &Job{
			Handler: "unknown_handler",
			Payload: "test",
		}

		err := queue.Enqueue(job)
		if err != nil {
			t.Fatalf("Failed to enqueue job: %v", err)
		}

		// Start queue processing
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		_ = queue.Start(ctx)

		// Wait for job to be processed (first attempt)
		time.Sleep(500 * time.Millisecond)

		// Check job status in database - should be retrying since MaxAttempts defaults to 3
		var status string
		query := "SELECT status FROM jobs WHERE id = ?"
		err = queue.db.QueryRow(query, job.ID).Scan(&status)
		if err != nil {
			t.Fatalf("Failed to query job status: %v", err)
		}

		if status != JobStatusRetrying {
			t.Errorf("Expected job status to be '%s', got '%s'", JobStatusRetrying, status)
		}
	})
}

func TestSolidQueue_RetryJob(t *testing.T) {
	queue := setupTestQueue(t)

	attemptCount := int32(0)
	queue.RegisterHandler("retry_handler", func(ctx *JobContext) error {
		attempts := atomic.AddInt32(&attemptCount, 1)
		return fmt.Errorf("attempt %d failed", attempts) // Always fail to test retry behavior
	})

	job := &Job{
		Handler:     "retry_handler",
		Payload:     "retry job",
		MaxAttempts: 2, // Limit attempts for faster test
	}

	err := queue.Enqueue(job)
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	// Start queue processing
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = queue.Start(ctx)

	// Wait for initial job attempt
	time.Sleep(500 * time.Millisecond)

	// Check that at least one attempt was made
	finalAttempts := atomic.LoadInt32(&attemptCount)
	if finalAttempts < 1 {
		t.Errorf("Expected at least 1 attempt, got %d", finalAttempts)
	}

	// Check job status - should be retrying after first failure
	var status string
	query := "SELECT status FROM jobs WHERE id = ?"
	err = queue.db.QueryRow(query, job.ID).Scan(&status)
	if err != nil {
		t.Fatalf("Failed to query job status: %v", err)
	}

	if status != JobStatusRetrying {
		t.Errorf("Expected job status to be '%s', got '%s'", JobStatusRetrying, status)
	}
}

func TestSolidQueue_ConcurrentProcessing(t *testing.T) {
	queue := setupTestQueue(t)

	processedJobs := int32(0)
	concurrentJobs := int32(0)
	maxConcurrency := int32(0)

	queue.RegisterHandler("concurrent_handler", func(ctx *JobContext) error {
		current := atomic.AddInt32(&concurrentJobs, 1)

		// Track max concurrency
		for {
			max := atomic.LoadInt32(&maxConcurrency)
			if current <= max || atomic.CompareAndSwapInt32(&maxConcurrency, max, current) {
				break
			}
		}

		// Simulate some work
		time.Sleep(200 * time.Millisecond)

		atomic.AddInt32(&concurrentJobs, -1)
		atomic.AddInt32(&processedJobs, 1)
		return nil
	})

	// Enqueue multiple jobs
	numJobs := 5
	for i := 0; i < numJobs; i++ {
		job := &Job{
			Handler: "concurrent_handler",
			Payload: fmt.Sprintf("job_%d", i),
		}

		err := queue.Enqueue(job)
		if err != nil {
			t.Fatalf("Failed to enqueue job %d: %v", i, err)
		}
	}

	// Start queue processing
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = queue.Start(ctx)

	// Wait for all jobs to complete
	time.Sleep(2 * time.Second)

	finalProcessed := atomic.LoadInt32(&processedJobs)
	if finalProcessed != int32(numJobs) {
		t.Errorf("Expected %d processed jobs, got %d", numJobs, finalProcessed)
	}

	finalMaxConcurrency := atomic.LoadInt32(&maxConcurrency)
	if finalMaxConcurrency > 2 {
		t.Errorf("Expected max concurrency <= 2 (number of workers), got %d", finalMaxConcurrency)
	}
	if finalMaxConcurrency < 1 {
		t.Error("Expected at least some concurrent processing")
	}
}

func TestSolidQueue_GetStats(t *testing.T) {
	queue := setupTestQueue(t)

	// Enqueue jobs in different states
	completedJob := &Job{Handler: "test_handler", Payload: "completed"}
	_ = queue.Enqueue(completedJob)
	queue.markJobCompleted(&jobRecord{ID: 1})

	failedJob := &Job{Handler: "test_handler", Payload: "failed"}
	_ = queue.Enqueue(failedJob)
	queue.markJobFailed(&jobRecord{ID: 2, Attempts: 3, MaxAttempts: 3}, fmt.Errorf("test error"))

	pendingJob := &Job{Handler: "test_handler", Payload: "pending"}
	_ = queue.Enqueue(pendingJob)

	stats, err := queue.GetStats()
	if err != nil {
		t.Fatalf("GetStats() should not return error: %v", err)
	}

	if stats["workers"] != 2 {
		t.Errorf("Expected 2 workers in stats, got %v", stats["workers"])
	}

	jobsByStatus, ok := stats["jobs_by_status"].(map[string]int)
	if !ok {
		t.Error("jobs_by_status should be map[string]int")
	}

	if jobsByStatus[JobStatusCompleted] < 1 {
		t.Error("Should have at least 1 completed job")
	}
	if jobsByStatus[JobStatusFailed] < 1 {
		t.Error("Should have at least 1 failed job")
	}
	if jobsByStatus[JobStatusPending] < 1 {
		t.Error("Should have at least 1 pending job")
	}
}

func TestSolidQueue_Purge(t *testing.T) {
	queue := setupTestQueue(t)

	// Create old completed job
	oldJob := &Job{Handler: "test_handler", Payload: "old job"}
	err := queue.Enqueue(oldJob)
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	// Mark as completed with old timestamp
	oldTime := time.Now().Add(-2 * time.Hour)
	query := `UPDATE jobs SET status = ?, completed_at = ? WHERE id = ?`
	_, err = queue.db.Exec(query, JobStatusCompleted, oldTime, oldJob.ID)
	if err != nil {
		t.Fatalf("Failed to update job: %v", err)
	}

	// Create recent completed job
	recentJob := &Job{Handler: "test_handler", Payload: "recent job"}
	err = queue.Enqueue(recentJob)
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}
	queue.markJobCompleted(&jobRecord{ID: 2})

	// Purge jobs older than 1 hour
	err = queue.Purge(1 * time.Hour)
	if err != nil {
		t.Fatalf("Purge() should not return error: %v", err)
	}

	// Verify old job was purged
	var count int
	err = queue.db.QueryRow("SELECT COUNT(*) FROM jobs WHERE id = ?", oldJob.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query job count: %v", err)
	}
	if count != 0 {
		t.Error("Old completed job should have been purged")
	}

	// Verify recent job still exists
	err = queue.db.QueryRow("SELECT COUNT(*) FROM jobs WHERE id = ?", recentJob.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query job count: %v", err)
	}
	if count != 1 {
		t.Error("Recent completed job should still exist")
	}
}

func TestSolidQueue_Retry(t *testing.T) {
	queue := setupTestQueue(t)

	job := &Job{
		Handler:     "test_handler",
		Payload:     "retry test",
		MaxAttempts: 1,
	}

	err := queue.Enqueue(job)
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	// Mark job as failed
	queue.markJobFailed(&jobRecord{ID: 1, Attempts: 1, MaxAttempts: 1}, fmt.Errorf("test failure"))

	// Retry the job
	err = queue.Retry(job.ID)
	if err != nil {
		t.Fatalf("Retry() should not return error: %v", err)
	}

	// Verify job status is back to pending
	var status string
	var attempts int
	query := "SELECT status, attempts FROM jobs WHERE id = ?"
	err = queue.db.QueryRow(query, job.ID).Scan(&status, &attempts)
	if err != nil {
		t.Fatalf("Failed to query job: %v", err)
	}

	if status != JobStatusPending {
		t.Errorf("Expected status to be '%s', got '%s'", JobStatusPending, status)
	}
	if attempts != 0 {
		t.Errorf("Expected attempts to be 0, got %d", attempts)
	}
}

func TestSolidQueue_Cancel(t *testing.T) {
	queue := setupTestQueue(t)

	job := &Job{
		Handler: "test_handler",
		Payload: "cancel test",
	}

	err := queue.Enqueue(job)
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	// Cancel the job
	err = queue.Cancel(job.ID)
	if err != nil {
		t.Fatalf("Cancel() should not return error: %v", err)
	}

	// Verify job is deleted
	var count int
	err = queue.db.QueryRow("SELECT COUNT(*) FROM jobs WHERE id = ?", job.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query job count: %v", err)
	}
	if count != 0 {
		t.Error("Job should have been deleted")
	}
}

func TestSolidQueue_CancelNonPending(t *testing.T) {
	queue := setupTestQueue(t)

	job := &Job{
		Handler: "test_handler",
		Payload: "cancel test",
	}

	err := queue.Enqueue(job)
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	// Mark job as running
	query := "UPDATE jobs SET status = ? WHERE id = ?"
	_, err = queue.db.Exec(query, JobStatusRunning, job.ID)
	if err != nil {
		t.Fatalf("Failed to update job status: %v", err)
	}

	// Try to cancel running job (should fail)
	err = queue.Cancel(job.ID)
	if err == nil {
		t.Error("Cancel() should return error for non-pending job")
	}
}

func TestSolidQueue_StartStop(t *testing.T) {
	queue := setupTestQueue(t)

	// Start the queue
	ctx, cancel := context.WithCancel(context.Background())
	err := queue.Start(ctx)
	if err != nil {
		t.Fatalf("Start() should not return error: %v", err)
	}

	// Let it run briefly
	time.Sleep(200 * time.Millisecond)

	// Stop the queue
	cancel()
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()

	err = queue.Stop(stopCtx)
	if err != nil {
		t.Fatalf("Stop() should not return error: %v", err)
	}
}

func TestJobContext_Structure(t *testing.T) {
	ctx := &JobContext{
		ID:       "123",
		Handler:  "test_handler",
		Payload:  map[string]string{"key": "value"},
		Attempt:  2,
		Queue:    "test_queue",
		Metadata: make(map[string]interface{}),
	}

	if ctx.ID != "123" {
		t.Errorf("Expected ID '123', got '%s'", ctx.ID)
	}
	if ctx.Handler != "test_handler" {
		t.Errorf("Expected Handler 'test_handler', got '%s'", ctx.Handler)
	}
	if ctx.Attempt != 2 {
		t.Errorf("Expected Attempt 2, got %d", ctx.Attempt)
	}
	if ctx.Queue != "test_queue" {
		t.Errorf("Expected Queue 'test_queue', got '%s'", ctx.Queue)
	}
	if ctx.Metadata == nil {
		t.Error("Metadata should not be nil")
	}
}

func TestJob_Structure(t *testing.T) {
	scheduledAt := time.Now()
	job := &Job{
		ID:          "456",
		Handler:     "example_handler",
		Queue:       "example_queue",
		Payload:     "example payload",
		MaxAttempts: 5,
		ScheduledAt: scheduledAt,
	}

	if job.ID != "456" {
		t.Errorf("Expected ID '456', got '%s'", job.ID)
	}
	if job.Handler != "example_handler" {
		t.Errorf("Expected Handler 'example_handler', got '%s'", job.Handler)
	}
	if job.Queue != "example_queue" {
		t.Errorf("Expected Queue 'example_queue', got '%s'", job.Queue)
	}
	if job.MaxAttempts != 5 {
		t.Errorf("Expected MaxAttempts 5, got %d", job.MaxAttempts)
	}
	if !job.ScheduledAt.Equal(scheduledAt) {
		t.Errorf("Expected ScheduledAt %v, got %v", scheduledAt, job.ScheduledAt)
	}
}