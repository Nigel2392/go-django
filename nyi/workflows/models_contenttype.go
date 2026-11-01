package workflows

import (
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
)

type ContentType struct {
	Workflow    *Workflow
	ContentType *contenttypes.BaseContentType[attrs.Definer]
}
