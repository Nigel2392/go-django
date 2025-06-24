package queries

import (
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-signals"
)

// Signals are used to notify when a model instance is saved or deleted.
//
// SignalSave is only meant to hold the model instance and the query compiler'
type SignalSave struct {
	Instance attrs.Definer
	Using    QueryCompiler
}

var (
	signals_saving   = signals.NewPool[SignalSave]()
	signals_deleting = signals.NewPool[attrs.Definer]()

	// Signal to be executed before inserting / updating a model instance.
	SignalPreModelSave = signals_saving.Get("queries.model.pre_save")

	// Signal to be executed after inserting / updating a model instance.
	SignalPostModelSave = signals_saving.Get("queries.model.post_save")

	// Signal to be executed before deleting a model instance.
	SignalPreModelDelete = signals_deleting.Get("queries.model.pre_delete")

	// Signal to be executed after deleting a model instance.
	SignalPostModelDelete = signals_deleting.Get("queries.model.post_delete")
)
