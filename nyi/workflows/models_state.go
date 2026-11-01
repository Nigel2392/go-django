package workflows

import (
	"context"
	"time"

	"github.com/Nigel2392/go-django/contrib/auth/users"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/trans"
)

type Status string

const (
	STATUS_IN_PROGRESS   Status = "in_progress"
	STATUS_APPROVED      Status = "approved"
	STATUS_NEEDS_CHANGES Status = "needs_changes"
	STATUS_CANCELLED     Status = "cancelled"
)

var STATUS_LABELS = map[Status]func(context.Context) string{
	STATUS_IN_PROGRESS:   trans.S("In Progress"),
	STATUS_APPROVED:      trans.S("Approved"),
	STATUS_NEEDS_CHANGES: trans.S("Needs Changes"),
	STATUS_CANCELLED:     trans.S("Cancelled"),
}

type BaseContentTypeDefiner interface {
	BaseContentType() *contenttypes.BaseContentType[attrs.Definer]
}

func GetBaseContentType(obj attrs.Definer) *contenttypes.BaseContentType[attrs.Definer] {
	if d, ok := obj.(BaseContentTypeDefiner); ok {
		return d.BaseContentType()
	}
	return contenttypes.NewContentType(obj)
}

type State struct {
	ID              drivers.ULID
	ContentType     *contenttypes.BaseContentType[attrs.Definer]
	BaseContentType *contenttypes.BaseContentType[attrs.Definer]
	ObjectID        string
	Status          Status
	CreatedAt       time.Time
	RequestedBy     users.User // FK
	Workflow        *Workflow  // FK
	TaskState       *TaskState // O2O
}
