package queries

import (
	"context"
	"reflect"
	"sync"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/embedder"
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
	queries = signals.NewPool[ModelSignal]()

	// Signal to be executed when a model is setup.
	SignalModelInit = queries.Get("queries.model.init")

	// Signal to be executed before inserting / updating a model instance.
	SignalPreModelSave = queries.Get("queries.model.pre_save")

	// Signal to be executed after inserting / updating a model instance.
	SignalPostModelSave = queries.Get("queries.model.post_save")

	// Signal to be executed before inserting model instance.
	SignalPreModelCreate = queries.Get("queries.model.pre_create")

	// Signal to be executed after inserting a model instance.
	SignalPostModelCreate = queries.Get("queries.model.post_create")

	// Signal to be executed before deleting a model instance.
	SignalPreModelDelete = queries.Get("queries.model.pre_delete")

	// Signal to be executed after deleting a model instance.
	SignalPostModelDelete = queries.Get("queries.model.post_delete")
)

func Listen(object attrs.Definer, signal signals.Signal[ModelSignal], fn func(ctx context.Context, s signals.Signal[ModelSignal], ms ModelSignal) error) (cancel func(context.Context)) {
	return modelSignals.listen(reflect.TypeOf(object), signal, fn)
}

var (
	modelSignals = new(modelSignalReg{
		models: make(map[reflect.Type]*signals.Pool[ModelSignal]),
	})

	_, _ = SignalPreModelSave.Listen(context.Background(), func(ctx context.Context, s signals.Signal[ModelSignal], ms ModelSignal) error {
		return embedder.PreSave.Bind(ctx, ms.Instance)
	})
	_, _ = SignalPostModelSave.Listen(context.Background(), func(ctx context.Context, s signals.Signal[ModelSignal], ms ModelSignal) error {
		return embedder.PostSave.Bind(ctx, ms.Instance)
	})

	_, _ = SignalModelInit.Listen(context.Background(), modelSignals.call)
	_, _ = SignalPreModelSave.Listen(context.Background(), modelSignals.call)
	_, _ = SignalPostModelSave.Listen(context.Background(), modelSignals.call)
	_, _ = SignalPreModelCreate.Listen(context.Background(), modelSignals.call)
	_, _ = SignalPostModelCreate.Listen(context.Background(), modelSignals.call)
	_, _ = SignalPreModelDelete.Listen(context.Background(), modelSignals.call)
	_, _ = SignalPostModelDelete.Listen(context.Background(), modelSignals.call)
)

type modelSignalReg struct {
	mu     sync.RWMutex
	models map[reflect.Type]*signals.Pool[ModelSignal]
}

func (m *modelSignalReg) listen(object reflect.Type, signal signals.Signal[ModelSignal], fn func(ctx context.Context, s signals.Signal[ModelSignal], ms ModelSignal) error) (cancel func(context.Context)) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sigs, ok := m.models[object]
	if !ok {
		sigs = signals.NewPool[ModelSignal]()
		m.models[object] = sigs
	}

	sig := sigs.Get(signal.Name())
	recv, _ := sig.Listen(context.Background(), fn)

	return func(ctx context.Context) {
		sig.Disconnect(ctx, recv)
	}
}

func (m *modelSignalReg) call(ctx context.Context, s signals.Signal[ModelSignal], ms ModelSignal) error {
	object := reflect.TypeOf(ms.Instance)

	m.mu.RLock()
	sigs, ok := m.models[object]
	m.mu.RUnlock()

	if !ok {
		return nil
	}

	err := sigs.Send(ctx, s.Name(), ms)
	if err != nil && !errors.Is(err, signals.ErrSignalNotFound) {
		return err
	}

	return nil
}
