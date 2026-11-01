package workflows

import (
	"github.com/Nigel2392/go-django/contrib/auth/users"
	"github.com/Nigel2392/go-signals"
)

type SignalState[INSTANCE any] struct {
	Instance INSTANCE
	User     users.User
}

var (
	WorkflowSignalPool = signals.NewGPool()

	// Workflow signals
	// provides args: *Workflow, users.User
	WorkflowSubmitted = WorkflowSignalPool.Get[SignalState[*Workflow]]("workflows.Workflow.Submitted")
	WorkflowApproved  = WorkflowSignalPool.Get[SignalState[*Workflow]]("workflows.Workflow.Approved")
	WorkflowRejected  = WorkflowSignalPool.Get[SignalState[*Workflow]]("workflows.Workflow.Rejected")
	WorkflowCancelled = WorkflowSignalPool.Get[SignalState[*Workflow]]("workflows.Workflow.Cancelled")

	// Workflow task signals
	// provides args: *Task, users.User
	TaskSubmitted = WorkflowSignalPool.Get[SignalState[*Task]]("workflows.Task.Submitted")
	TaskApproved  = WorkflowSignalPool.Get[SignalState[*Task]]("workflows.Task.Approved")
	TaskRejected  = WorkflowSignalPool.Get[SignalState[*Task]]("workflows.Task.Rejected")
	TaskCancelled = WorkflowSignalPool.Get[SignalState[*Task]]("workflows.Task.Cancelled")
)
