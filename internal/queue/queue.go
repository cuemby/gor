package queue

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Job status constants
const (
	JobStatusPending   = "pending"
	JobStatusRunning   = "running"
	JobStatusCompleted = "completed"
	JobStatusFailed    = "failed"
	JobStatusRetrying  = "retrying"
)

// jobRecord represents a job in the database
type jobRecord struct {
	ID          int64
	Queue       string
	Handler     string
	Payload     string
	Status      string
	Attempts    int
	MaxAttempts int
	Error       sql.NullString
	ScheduledAt time.Time
	StartedAt   sql.NullTime
	CompletedAt sql.NullTime
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// SolidQueue implements a database-backed job queue similar to Rails' Solid Queue
type SolidQueue struct {
	db           *sql.DB
	handlers     map[string]JobHandler
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	workers      int
	wg           sync.WaitGroup
	processing   sync.Map // Track jobs being processed
	pollInterval time.Duration
}

// NewSolidQueue creates a new database-backed queue
func NewSolidQueue(dbPath string, workers int) (*SolidQueue, error) {
	if workers <= 0 {
		workers = 5 // Default number of workers
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	sq := &SolidQueue{
		db:           db,
		handlers:     make(map[string]JobHandler),
		ctx:          ctx,
		cancel:       cancel,
		workers:      workers,
		pollInterval: 1 * time.Second,
	}

	// Create jobs table
	if err := sq.createTables(); err != nil {
		db.Close()
		return nil, err
	}

	return sq, nil
}

// createTables creates the necessary database tables
func (sq *SolidQueue) createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS jobs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		queue TEXT NOT NULL DEFAULT 'default',
		handler TEXT NOT NULL,
		payload TEXT,
		status TEXT NOT NULL DEFAULT 'pending',
		attempts INTEGER DEFAULT 0,
		max_attempts INTEGER DEFAULT 3,
		error TEXT,
		scheduled_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		started_at TIMESTAMP,
		completed_at TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_jobs_status_scheduled ON jobs(status, scheduled_at);
	CREATE INDEX IF NOT EXISTS idx_jobs_queue ON jobs(queue);
	`

	_, err := sq.db.Exec(schema)
	return err
}

// Job represents a simple job structure
type Job struct {
	ID          string
	Handler     string
	Queue       string
	Payload     interface{}
	MaxAttempts int
	ScheduledAt time.Time
}

// JobHandler is a function that processes a job
type JobHandler func(ctx *JobContext) error

// JobContext provides context for job execution
type JobContext struct {
	ID       string
	Handler  string
	Payload  interface{}
	Attempt  int
	Queue    string
	Metadata map[string]interface{}
}

// Enqueue adds a job to the queue
func (sq *SolidQueue) Enqueue(job *Job) error {
	if job.Queue == "" {
		job.Queue = "default"
	}
	if job.MaxAttempts <= 0 {
		job.MaxAttempts = 3
	}
	if job.ScheduledAt.IsZero() {
		job.ScheduledAt = time.Now()
	}

	payloadJSON, err := json.Marshal(job.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	query := `
		INSERT INTO jobs (queue, handler, payload, scheduled_at, max_attempts)
		VALUES (?, ?, ?, ?, ?)
	`

	result, err := sq.db.Exec(query, job.Queue, job.Handler, string(payloadJSON), job.ScheduledAt, job.MaxAttempts)
	if err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	id, _ := result.LastInsertId()
	job.ID = fmt.Sprintf("%d", id)

	return nil
}

// EnqueueAt schedules a job for future execution
func (sq *SolidQueue) EnqueueAt(job *Job, at time.Time) error {
	job.ScheduledAt = at
	return sq.Enqueue(job)
}

// EnqueueIn schedules a job to run after a delay
func (sq *SolidQueue) EnqueueIn(job *Job, delay time.Duration) error {
	return sq.EnqueueAt(job, time.Now().Add(delay))
}

// RegisterHandler registers a job handler
func (sq *SolidQueue) RegisterHandler(name string, handler JobHandler) {
	sq.mu.Lock()
	defer sq.mu.Unlock()
	sq.handlers[name] = handler
}

// Start begins processing jobs
func (sq *SolidQueue) Start(ctx context.Context) error {
	log.Printf("Starting Solid Queue with %d workers", sq.workers)

	// Start worker goroutines
	for i := 0; i < sq.workers; i++ {
		sq.wg.Add(1)
		go sq.worker(i)
	}

	// Start job poller
	sq.wg.Add(1)
	go sq.poller()

	return nil
}

// Stop gracefully shuts down the queue
func (sq *SolidQueue) Stop(ctx context.Context) error {
	log.Println("Stopping Solid Queue...")
	sq.cancel()

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		sq.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("Solid Queue stopped gracefully")
	case <-ctx.Done():
		log.Println("Solid Queue stop timeout")
	}

	return sq.db.Close()
}

// worker processes jobs from the queue
func (sq *SolidQueue) worker(id int) {
	defer sq.wg.Done()
	log.Printf("Worker %d started", id)

	for {
		select {
		case <-sq.ctx.Done():
			log.Printf("Worker %d stopping", id)
			return
		default:
			// Try to claim a job
			job := sq.claimNextJob()
			if job == nil {
				// No jobs available, wait before trying again
				time.Sleep(100 * time.Millisecond)
				continue
			}

			// Process the job
			sq.processJob(job)
		}
	}
}

// poller periodically retries failed jobs
func (sq *SolidQueue) poller() {
	defer sq.wg.Done()
	ticker := time.NewTicker(sq.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-sq.ctx.Done():
			return
		case <-ticker.C:
			sq.retryFailedJobs()
		}
	}
}

// claimNextJob atomically claims the next available job
func (sq *SolidQueue) claimNextJob() *jobRecord {
	tx, err := sq.db.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		return nil
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Select next available job
	query := `
		SELECT id, queue, handler, payload, attempts, max_attempts
		FROM jobs
		WHERE status = ?
		  AND scheduled_at <= ?
		ORDER BY scheduled_at
		LIMIT 1
	`

	var job jobRecord
	err = tx.QueryRow(query, JobStatusPending, time.Now()).
		Scan(&job.ID, &job.Queue, &job.Handler, &job.Payload, &job.Attempts, &job.MaxAttempts)

	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		log.Printf("Failed to query job: %v", err)
		return nil
	}

	// Check if job is already being processed
	if _, exists := sq.processing.LoadOrStore(job.ID, true); exists {
		return nil
	}

	// Update job status to running
	updateQuery := `
		UPDATE jobs
		SET status = ?, started_at = ?, attempts = attempts + 1, updated_at = ?
		WHERE id = ? AND status = ?
	`

	now := time.Now()
	result, err := tx.Exec(updateQuery, JobStatusRunning, now, now, job.ID, JobStatusPending)
	if err != nil {
		sq.processing.Delete(job.ID)
		log.Printf("Failed to update job status: %v", err)
		return nil
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		// Job was already claimed by another worker
		sq.processing.Delete(job.ID)
		return nil
	}

	if err := tx.Commit(); err != nil {
		sq.processing.Delete(job.ID)
		log.Printf("Failed to commit transaction: %v", err)
		return nil
	}

	job.Status = JobStatusRunning
	job.StartedAt = sql.NullTime{Time: now, Valid: true}
	job.Attempts++

	return &job
}

// processJob executes a job
func (sq *SolidQueue) processJob(job *jobRecord) {
	defer sq.processing.Delete(job.ID)

	sq.mu.RLock()
	handler, exists := sq.handlers[job.Handler]
	sq.mu.RUnlock()

	if !exists {
		sq.markJobFailed(job, fmt.Errorf("handler '%s' not found", job.Handler))
		return
	}

	// Unmarshal payload
	var payload interface{}
	if job.Payload != "" {
		if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil {
			sq.markJobFailed(job, fmt.Errorf("failed to unmarshal payload: %w", err))
			return
		}
	}

	// Create job context
	jobCtx := &JobContext{
		ID:       fmt.Sprintf("%d", job.ID),
		Handler:  job.Handler,
		Payload:  payload,
		Attempt:  job.Attempts,
		Queue:    job.Queue,
		Metadata: make(map[string]interface{}),
	}

	// Execute the handler
	err := handler(jobCtx)
	if err != nil {
		sq.markJobFailed(job, err)
	} else {
		sq.markJobCompleted(job)
	}
}

// markJobCompleted marks a job as completed
func (sq *SolidQueue) markJobCompleted(job *jobRecord) {
	query := `
		UPDATE jobs
		SET status = ?, completed_at = ?, updated_at = ?
		WHERE id = ?
	`

	now := time.Now()
	if _, err := sq.db.Exec(query, JobStatusCompleted, now, now, job.ID); err != nil {
		log.Printf("Failed to mark job %d as completed: %v", job.ID, err)
	}
}

// markJobFailed marks a job as failed
func (sq *SolidQueue) markJobFailed(job *jobRecord, jobErr error) {
	status := JobStatusFailed
	if job.Attempts < job.MaxAttempts {
		status = JobStatusRetrying
	}

	query := `
		UPDATE jobs
		SET status = ?, error = ?, updated_at = ?
		WHERE id = ?
	`

	now := time.Now()
	if _, err := sq.db.Exec(query, status, jobErr.Error(), now, job.ID); err != nil {
		log.Printf("Failed to mark job %d as failed: %v", job.ID, err)
	}

	log.Printf("Job %d failed (attempt %d/%d): %v", job.ID, job.Attempts, job.MaxAttempts, jobErr)
}

// retryFailedJobs retries jobs that have failed but haven't exceeded max attempts
func (sq *SolidQueue) retryFailedJobs() {
	query := `
		UPDATE jobs
		SET status = ?, scheduled_at = ?, updated_at = ?
		WHERE status = ?
		  AND attempts < max_attempts
		  AND updated_at < ?
	`

	// Retry jobs that have been in retry status for at least 30 seconds
	retryAfter := time.Now().Add(-30 * time.Second)
	scheduledAt := time.Now().Add(5 * time.Second) // Schedule retry 5 seconds from now

	result, err := sq.db.Exec(query, JobStatusPending, scheduledAt, time.Now(), JobStatusRetrying, retryAfter)
	if err != nil {
		log.Printf("Failed to retry jobs: %v", err)
		return
	}

	if rows, _ := result.RowsAffected(); rows > 0 {
		log.Printf("Scheduled %d jobs for retry", rows)
	}
}

// Purge removes completed jobs older than the specified duration
func (sq *SolidQueue) Purge(olderThan time.Duration) error {
	query := `
		DELETE FROM jobs
		WHERE status = ?
		  AND completed_at < ?
	`

	cutoff := time.Now().Add(-olderThan)
	result, err := sq.db.Exec(query, JobStatusCompleted, cutoff)
	if err != nil {
		return fmt.Errorf("failed to purge jobs: %w", err)
	}

	if rows, _ := result.RowsAffected(); rows > 0 {
		log.Printf("Purged %d completed jobs", rows)
	}

	return nil
}

// GetStats returns queue statistics
func (sq *SolidQueue) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Count jobs by status
	query := `
		SELECT status, COUNT(*) as count
		FROM jobs
		GROUP BY status
	`

	rows, err := sq.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	statusCounts := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		statusCounts[status] = count
	}

	stats["jobs_by_status"] = statusCounts
	stats["workers"] = sq.workers
	stats["processing_count"] = sq.getProcessingCount()

	return stats, nil
}

// getProcessingCount returns the number of jobs currently being processed
func (sq *SolidQueue) getProcessingCount() int {
	count := 0
	sq.processing.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// Retry retries a specific job by ID
func (sq *SolidQueue) Retry(jobID string) error {
	query := `
		UPDATE jobs
		SET status = ?, scheduled_at = ?, attempts = 0, error = NULL, updated_at = ?
		WHERE id = ?
	`

	now := time.Now()
	_, err := sq.db.Exec(query, JobStatusPending, now, now, jobID)
	if err != nil {
		return fmt.Errorf("failed to retry job: %w", err)
	}

	return nil
}

// Cancel cancels a pending job
func (sq *SolidQueue) Cancel(jobID string) error {
	query := `
		DELETE FROM jobs
		WHERE id = ? AND status = ?
	`

	result, err := sq.db.Exec(query, jobID, JobStatusPending)
	if err != nil {
		return fmt.Errorf("failed to cancel job: %w", err)
	}

	if rows, _ := result.RowsAffected(); rows == 0 {
		return errors.New("job not found or not in pending status")
	}

	return nil
}
