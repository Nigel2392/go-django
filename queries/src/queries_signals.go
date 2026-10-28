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
	// Signal to be executed when a model is setup.
	SignalModelInit = signals.Get[ModelSignal]("queries.model.init")

	// Signal to be executed before inserting / updating a model instance.
	SignalPreModelSave = signals.Get[ModelSignal]("queries.model.pre_save")

	// Signal to be executed after inserting / updating a model instance.
	SignalPostModelSave = signals.Get[ModelSignal]("queries.model.post_save")

	// Signal to be executed before inserting model instance.
	SignalPreModelCreate = signals.Get[ModelSignal]("queries.model.pre_create")

	// Signal to be executed after inserting a model instance.
	SignalPostModelCreate = signals.Get[ModelSignal]("queries.model.post_create")

	// Signal to be executed before deleting a model instance.
	SignalPreModelDelete = signals.Get[ModelSignal]("queries.model.pre_delete")

	// Signal to be executed after deleting a model instance.
	SignalPostModelDelete = signals.Get[ModelSignal]("queries.model.post_delete")
)
