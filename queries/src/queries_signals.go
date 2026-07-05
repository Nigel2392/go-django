package queries

import (
	"context"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-signals"
)

// Signals are used to notify when a model instance is saved or deleted.
type ModelSignal struct {
	Context  context.Context
	Instance attrs.Definer
	Using    QueryCompiler
}

// Signals are used to notify when a model instance is saved or deleted.
//
// Deprecated: SignalSave exists for historical compatibility
// please use [ModelSignal] instead.
type SignalSave = ModelSignal

var (
	signals_model = signals.NewPool[ModelSignal]()

	// Signal to be executed before inserting / updating a model instance.
	SignalPreModelSave = signals_model.Get("queries.model.pre_save")

	// Signal to be executed after inserting / updating a model instance.
	SignalPostModelSave = signals_model.Get("queries.model.post_save")

	// Signal to be executed before inserting model instance.
	SignalPreModelCreate = signals_model.Get("queries.model.pre_create")

	// Signal to be executed after inserting a model instance.
	SignalPostModelCreate = signals_model.Get("queries.model.post_create")

	// Signal to be executed before deleting a model instance.
	SignalPreModelDelete = signals_model.Get("queries.model.pre_delete")

	// Signal to be executed after deleting a model instance.
	SignalPostModelDelete = signals_model.Get("queries.model.post_delete")
)
