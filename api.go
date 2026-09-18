package scheduler

import "time"

// CreateJobRequest contains the configuration for creating or updating a job.
type CreateJobRequest struct {
	Name       string            `json:"name" binding:"required"`
	Schedule   string            `json:"schedule" binding:"required"`
	WebhookURL string            `json:"webhook_url" binding:"required,url"`
	Parameters map[string]string `json:"parameters"`
	Headers    map[string]string `json:"headers"`
	Payload    string            `json:"payload"`
}

// Job is the public representation of a scheduled webhook job.
type Job struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Schedule   string            `json:"schedule"`
	WebhookURL string            `json:"webhook_url"`
	Parameters map[string]string `json:"parameters,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	Payload    string            `json:"payload,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

// JobResult contains the latest webhook execution result for a job.
type JobResult struct {
	JobID      string    `json:"job_id"`
	StatusCode int       `json:"status_code"`
	Response   string    `json:"response"`
	Error      string    `json:"error,omitempty"`
	ExecutedAt time.Time `json:"executed_at"`
}
