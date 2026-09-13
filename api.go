package scheduler

import (
	"time"
)

type Scheduler struct {
	tasks []Task
}

type Task struct {
	ID          string    `json:"id"`
	Description string    `json:"description"`
	DueDate     time.Time `json:"due_date"`
	Completed   bool      `json:"completed"`
}
