package gor

import (
	"context"
	"time"
)

// Queue defines the background job processing interface.
// Inspired by Rails 8's Solid Queue - database-backed job processing.
type Queue interface {
	// Job scheduling and execution
	Enqueue(ctx context.Context, job Job) error
	EnqueueAt(ctx context.Context, job Job, at time.Time) error
	EnqueueIn(ctx context.Context, job Job, delay time.Duration) error

	// Job management
	Cancel(ctx context.Context, jobID string) error
	Retry(ctx context.Context, jobID string) error
	Delete(ctx context.Context, jobID string) error

	// Queue management
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Pause(ctx context.Context, queueName string) error
	Resume(ctx context.Context, queueName string) error

	// Monitoring and statistics
	Stats(ctx context.Context) (QueueStats, error)
	JobStatus(ctx context.Context, jobID string) (JobStatus, error)
	ListJobs(ctx context.Context, opts ListJobsOptions) ([]JobInfo, error)

	// Worker management
	RegisterWorker(name string, worker Worker) error
	UnregisterWorker(name string) error
	Workers() map[string]Worker
}

// Job represents a background job to be processed.
type Job interface {
	// Job identification
	ID() string
	Type() string
	Queue() string

	// Job data and configuration
	Payload() interface{}
	Priority() int
	MaxRetries() int
	RetryDelay() time.Duration

	// Execution
	Perform(ctx context.Context) error

	// Serialization for database storage
	Marshal() ([]byte, error)
	Unmarshal(data []byte) error
}

// BaseJob provides a default implementation of common job functionality.
type BaseJob struct {
	JobID        string        `json:"id"`
	JobType      string        `json:"type"`
	QueueName    string        `json:"queue"`
	JobPayload   interface{}   `json:"payload"`
	JobPriority  int           `json:"priority"`
	JobRetries   int           `json:"max_retries"`
	JobDelay     time.Duration `json:"retry_delay"`
}

func (j *BaseJob) ID() string                    { return j.JobID }
func (j *BaseJob) Type() string                  { return j.JobType }
func (j *BaseJob) Queue() string                 { return j.QueueName }
func (j *BaseJob) Payload() interface{}          { return j.JobPayload }
func (j *BaseJob) Priority() int                 { return j.JobPriority }
func (j *BaseJob) MaxRetries() int               { return j.JobRetries }
func (j *BaseJob) RetryDelay() time.Duration     { return j.JobDelay }

// Worker defines the interface for job workers.
type Worker interface {
	// Worker identification
	Name() string
	Concurrency() int

	// Job processing
	Process(ctx context.Context, job Job) error

	// Lifecycle hooks
	BeforeProcess(ctx context.Context, job Job) error
	AfterProcess(ctx context.Context, job Job, err error) error
	OnError(ctx context.Context, job Job, err error) error
}

// JobStatus represents the current status of a job.
type JobStatus string

const (
	JobPending    JobStatus = "pending"
	JobProcessing JobStatus = "processing"
	JobCompleted  JobStatus = "completed"
	JobFailed     JobStatus = "failed"
	JobCancelled  JobStatus = "cancelled"
	JobRetrying   JobStatus = "retrying"
)

// JobInfo contains metadata about a job.
type JobInfo struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Queue       string                 `json:"queue"`
	Status      JobStatus              `json:"status"`
	Priority    int                    `json:"priority"`
	Attempts    int                    `json:"attempts"`
	MaxRetries  int                    `json:"max_retries"`
	Payload     map[string]interface{} `json:"payload"`
	Error       string                 `json:"error,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	ScheduledAt *time.Time             `json:"scheduled_at,omitempty"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// QueueStats provides statistics about queue performance.
type QueueStats struct {
	Queues map[string]QueueInfo `json:"queues"`
	Total  QueueInfo            `json:"total"`
}

type QueueInfo struct {
	Name       string `json:"name"`
	Pending    int64  `json:"pending"`
	Processing int64  `json:"processing"`
	Completed  int64  `json:"completed"`
	Failed     int64  `json:"failed"`
	Cancelled  int64  `json:"cancelled"`
	Workers    int    `json:"workers"`
}

// ListJobsOptions provides options for listing jobs.
type ListJobsOptions struct {
	Queue    string     `json:"queue,omitempty"`
	Status   JobStatus  `json:"status,omitempty"`
	Type     string     `json:"type,omitempty"`
	Limit    int        `json:"limit,omitempty"`
	Offset   int        `json:"offset,omitempty"`
	SortBy   string     `json:"sort_by,omitempty"`
	SortDesc bool       `json:"sort_desc,omitempty"`
	After    *time.Time `json:"after,omitempty"`
	Before   *time.Time `json:"before,omitempty"`
}

// Recurring job support
type RecurringJob interface {
	Job
	Schedule() string // Cron expression
	NextRun() time.Time
	LastRun() *time.Time
}

// Job middleware for cross-cutting concerns
type JobMiddleware interface {
	Process(ctx context.Context, job Job, next func(context.Context, Job) error) error
}

// Common job types that can be embedded
type EmailJob struct {
	BaseJob
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
	HTML    bool     `json:"html"`
}

type WebhookJob struct {
	BaseJob
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    interface{}       `json:"body"`
	Timeout time.Duration     `json:"timeout"`
}

type CleanupJob struct {
	BaseJob
	ResourceType string    `json:"resource_type"`
	Before       time.Time `json:"before"`
	BatchSize    int       `json:"batch_size"`
}

// Job events for monitoring and hooks
type JobEvent struct {
	Type      JobEventType `json:"type"`
	JobID     string       `json:"job_id"`
	Queue     string       `json:"queue"`
	Worker    string       `json:"worker,omitempty"`
	Error     string       `json:"error,omitempty"`
	Duration  time.Duration `json:"duration,omitempty"`
	Timestamp time.Time    `json:"timestamp"`
}

type JobEventType string

const (
	JobEventEnqueued   JobEventType = "enqueued"
	JobEventStarted    JobEventType = "started"
	JobEventCompleted  JobEventType = "completed"
	JobEventFailed     JobEventType = "failed"
	JobEventCancelled  JobEventType = "cancelled"
	JobEventRetried    JobEventType = "retried"
)

// JobEventListener allows listening to job events
type JobEventListener interface {
	OnJobEvent(ctx context.Context, event JobEvent) error
}