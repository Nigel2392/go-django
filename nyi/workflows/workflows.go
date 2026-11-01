package workflows

import (
	"context"

	"github.com/Nigel2392/go-django/src/core/attrs"
)

type Workflow struct {
	//	 attrs.Definer
	//	 Label(ctx context.Context) string
	//	 IsActive(ctx context.Context) bool
	//	 Start(ctx context.Context, obj attrs.Definer, user users.User) (State[attrs.Definer], error)
	//	 Deactivate(ctx context.Context, user users.User) error
	//	 CurrentState(ctx context.Context, obj attrs.Definer) (State[attrs.Definer], error)
}

type WorkflowStep interface {
	attrs.Definer
	Tasks(ctx context.Context) ([]Task, error)
}
