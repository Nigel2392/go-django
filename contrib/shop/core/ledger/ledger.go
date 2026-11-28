package ledger

import (
	"context"
	"time"
	"uuid"
)

type Status string

const (
	PENDING   Status = "pending"
	RUNNING   Status = "running"
	COMPLETED Status = "completed"
	FAILED    Status = "failed"
	EXPIRED   Status = "expired"
)

type Task interface {
	ID() uuid.UUID
	ExpiresAt() time.Time
}

type TaskState interface {
	Task
	Tries() int
	Status() Status
	StartedAt() time.Time
	UpdatedAt() time.Time
	FinishedAt() (time.Time, bool)
	Failure() (code string, message string)
}

type Ledger[TASK any] interface {
	// Start enqueues a task, or returns the existing task for the key.
	Start(ctx context.Context, taskData TASK) (Task, error)
	Progress(ctx context.Context, taskID uuid.UUID) (TaskState, error)
}
