package workflows

import (
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
)

type WorkflowTask interface {
}

type Task struct {
	ID          drivers.ULID
	ContentType *contenttypes.BaseContentType[WorkflowTask]
}
