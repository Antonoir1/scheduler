package job

import (
	"encoding/json"
	"time"
)

// Job is the internal representation persisted by the scheduler.
type Job struct {
	ID         string            `bson:"_id"`
	Name       string            `bson:"name"`
	Schedule   string            `bson:"schedule"`
	WebhookURL string            `bson:"webhook_url"`
	Parameters map[string]string `bson:"parameters,omitempty"`
	Payload    map[string]any    `bson:"payload,omitempty"`
	LastResult *JobResult        `bson:"last_result,omitempty"`
	CreatedAt  time.Time         `bson:"created_at"`
	UpdatedAt  time.Time         `bson:"updated_at"`
}

// JobResult is the internal representation of the latest webhook execution.
type JobResult struct {
	JobID      string          `bson:"job_id"`
	StatusCode int             `bson:"status_code"`
	Response   json.RawMessage `bson:"response"`
	Error      string          `bson:"error,omitempty"`
	ExecutedAt time.Time       `bson:"executed_at"`
}
