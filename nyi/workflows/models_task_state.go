package workflows

import (
	"context"
	"time"

	"github.com/Nigel2392/go-django/contrib/auth/users"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/src/core/trans"
)

type TaskStatus string

const (
	TSTATUS_IN_PROGRESS TaskStatus = "in_progress"
	TSTATUS_APPROVED    TaskStatus = "approved"
	TSTATUS_REJECTED    TaskStatus = "rejected"
	TSTATUS_SKIPPED     TaskStatus = "skipped"
	TSTATUS_CANCELLED   TaskStatus = "cancelled"
)

var TASK_STATUS_LABELS = map[TaskStatus]func(context.Context) string{
	TSTATUS_IN_PROGRESS: trans.S("In progress"),
	TSTATUS_APPROVED:    trans.S("Approved"),
	TSTATUS_REJECTED:    trans.S("Rejected"),
	TSTATUS_SKIPPED:     trans.S("Skipped"),
	TSTATUS_CANCELLED:   trans.S("Cancelled"),
}

type TaskState struct {
	ID            drivers.ULID
	Status        TaskStatus
	Comment       drivers.Text
	StartedAt     time.Time
	FinishedAt    time.Time
	WorkflowState *State     // FK
	Task          *Task      // FK
	FinishedBy    users.User // FK
}
