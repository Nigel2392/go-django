package queries

import (
	"context"
	"fmt"
	"reflect"

	"github.com/Nigel2392/go-django/queries/src/query_errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/pkg/errors"
)

type actorFlag int

func (a actorFlag) String() string {
	if str, ok := actorStrings[a]; ok {
		return str
	}
	return fmt.Sprintf("UnknownActorFlag(%d)", a)
}

const (
	actsAfterQuery actorFlag = iota
	actsBeforeSave
	actsAfterSave
	actsBeforeCreate
	actsAfterCreate
	actsBeforeUpdate
	actsAfterUpdate
	actsBeforeDelete
	actsAfterDelete
)

var actorStrings = map[actorFlag]string{
	actsAfterQuery:   "AfterQuery",
	actsBeforeSave:   "BeforeSave",
	actsAfterSave:    "AfterSave",
	actsBeforeCreate: "BeforeCreate",
	actsAfterCreate:  "AfterCreate",
	actsBeforeUpdate: "BeforeUpdate",
	actsAfterUpdate:  "AfterUpdate",
	actsBeforeDelete: "BeforeDelete",
	actsAfterDelete:  "AfterDelete",
}

type ActsAfterQuery interface {
	AfterQuery(ctx context.Context) error
}

type ActsBeforeSave interface {
	BeforeSave(ctx context.Context) error
}

type ActsAfterSave interface {
	AfterSave(ctx context.Context) error
}

type ActsBeforeCreate interface {
	BeforeCreate(ctx context.Context) error
}

type ActsAfterCreate interface {
	AfterCreate(ctx context.Context) error
}

type ActsBeforeUpdate interface {
	BeforeUpdate(ctx context.Context) error
}

type ActsAfterUpdate interface {
	AfterUpdate(ctx context.Context) error
}

type ActsBeforeDelete interface {
	BeforeDelete(ctx context.Context) error
}

type ActsAfterDelete interface {
	AfterDelete(ctx context.Context) error
}

func runActor[T attrs.Definer](ctx context.Context, which actorFlag, target T) (context.Context, error) {
	return (&ObjectActor{obj: target}).execute(ctx, which)
}

// ObjectActor is a struct that can be used to run actions on objects that implement
// the respective actor interfaces.
//
// The [Actor] function must be called to create a new ObjectActor instance,
// otherwise the ObjectActor's methods will immediately return the context unchanged.
//
// If the object does not implement the respective actor interface, the method called
// will return the context unchanged and no error.
//
// It can be used to run the following actions:
// - BeforeSave: runs before the object is saved to the database.
// - AfterSave: runs after the object is saved to the database.
// - BeforeCreate: runs before the object is created in the database.
// - AfterCreate: runs after the object is created in the database.
// - BeforeUpdate: runs before the object is updated in the database.
// - AfterUpdate: runs after the object is updated in the database.
// - BeforeDelete: runs before the object is deleted from the database.
// - AfterDelete: runs after the object is deleted from the database.
//
// When an object has it's actor methods called, it will
// automatically mark the object as seen in the context.
// This prevents the same actor from being run multiple times
// for the same object in the same context.
//
// It is only needed for advanced use cases if you are saving objects
// to the database manually and want to ensure that the
// BeforeSave and AfterSave methods are called.
type ObjectActor struct {
	obj attrs.Definer
}

// Actor returns a new ObjectActor that can be used to run actions on objects.
// If the object does not implement the respective actor interface, the method called
// will return the context unchanged and no error.
func Actor(obj attrs.Definer) *ObjectActor {
	var (
		_, canBeforeSave   = obj.(ActsBeforeSave)
		_, canAfterSave    = obj.(ActsAfterSave)
		_, canBeforeCreate = obj.(ActsBeforeCreate)
		_, canAfterCreate  = obj.(ActsAfterCreate)
		_, canBeforeUpdate = obj.(ActsBeforeUpdate)
		_, canAfterUpdate  = obj.(ActsAfterUpdate)
		_, canBeforeDelete = obj.(ActsBeforeDelete)
		_, canAfterDelete  = obj.(ActsAfterDelete)
		_, canAfterQuery   = obj.(ActsAfterQuery)
		actOnSave          = canBeforeSave || canAfterSave
		actOnCreate        = canBeforeCreate || canAfterCreate
		actOnUpdate        = canBeforeUpdate || canAfterUpdate
		actOnDelete        = canBeforeDelete || canAfterDelete
	)

	if !actOnSave && !actOnCreate && !actOnUpdate && !actOnDelete && !canAfterQuery {
		return &ObjectActor{} // no-ops if obj is nil
	}

	return &ObjectActor{
		obj: obj,
	}
}

func (s *ObjectActor) execute(ctx context.Context, which actorFlag) (context.Context, error) {
	// no op if the wrapper is nil
	if s == nil {
		return ctx, nil
	}

	// no op if the object is nil
	if s.obj == nil {
		return ctx, nil
	}

	// if the object does not implement the actor interface, we just return the context unchanged
	// this is to avoid having to write a lot of boilerplate code for each actor interface
	// if the object implements the actor interface, we will skip the early return directly below
	var fn func(ctx context.Context) error
	switch which {
	case actsBeforeSave:
		if saver, ok := s.obj.(ActsBeforeSave); ok {
			fn = func(ctx context.Context) error {
				// we have not yet checked if the actor has been seen, wrap the signal
				var err = SignalPreModelSave.Send(SignalSave{
					Instance: s.obj,
				})
				if err != nil {
					return fmt.Errorf("error running pre-save signal for %T: %w", s.obj, err)
				}
				return saver.BeforeSave(ctx)
			}
			goto performAction
		}
	case actsAfterSave:
		if saver, ok := s.obj.(ActsAfterSave); ok {
			fn = func(ctx context.Context) error {
				// we have not yet checked if the actor has been seen, wrap the signal
				var err = SignalPostModelSave.Send(SignalSave{
					Instance: s.obj,
				})
				if err != nil {
					return fmt.Errorf("error running post-save signal for %T: %w", s.obj, err)
				}
				return saver.AfterSave(ctx)
			}
			goto performAction
		}
	case actsBeforeCreate:
		if creator, ok := s.obj.(ActsBeforeCreate); ok {
			fn = creator.BeforeCreate
			goto performAction
		}
	case actsAfterCreate:
		if creator, ok := s.obj.(ActsAfterCreate); ok {
			fn = creator.AfterCreate
			goto performAction
		}
	case actsBeforeUpdate:
		if updater, ok := s.obj.(ActsBeforeUpdate); ok {
			fn = updater.BeforeUpdate
			goto performAction
		}
	case actsAfterUpdate:
		if updater, ok := s.obj.(ActsAfterUpdate); ok {
			fn = updater.AfterUpdate
			goto performAction
		}
	case actsBeforeDelete:
		if deleter, ok := s.obj.(ActsBeforeDelete); ok {
			fn = deleter.BeforeDelete
			goto performAction
		}
	case actsAfterDelete:
		if deleter, ok := s.obj.(ActsAfterDelete); ok {
			fn = deleter.AfterDelete
			goto performAction
		}
	case actsAfterQuery:
		if afterQuery, ok := s.obj.(ActsAfterQuery); ok {
			fn = afterQuery.AfterQuery
			goto performAction
		}
	default:
		return ctx, fmt.Errorf("unknown actor flag: %d", which)
	}

	// this return statement might look iffy, but it saves us from
	// having to write a lot of boilerplate code for each actor interface
	//
	// if we are here, it means the object does not implement the
	// respective actor interface, so we just return the context unchanged
	return ctx, nil

	// if we are here, it means the object implements the actor interface
	// as it has jumped to the performAction label
performAction:
	if hasSeenActor(ctx, which, s.obj) {
		return ctx, nil
	}

	if ctx == nil {
		ctx = context.Background()
	}

	ctx = markActorSeen(ctx, which, s.obj)

	var err = fn(ctx)
	var isErrSkip = errors.Is(err, query_errors.ErrNotImplemented)
	if err != nil && !isErrSkip {
		return ctx, fmt.Errorf(
			"failed to execute %s for object %T: %w",
			which, s.obj, err,
		)
	}

	if isErrSkip {
		logger.Warnf(
			"Skipped %s for object %T: %s",
			which, s.obj, attrs.ToString(s.obj),
		)
	}

	switch which {
	case actsBeforeCreate, actsBeforeUpdate:
		// we must execute the BeforeSave action
		return s.execute(ctx, actsBeforeSave)
	case actsAfterCreate, actsAfterUpdate:
		// we must execute the AfterSave action
		return s.execute(ctx, actsAfterSave)
	default:
		//	logger.Warnf(
		//		"Executed %s for object %T: %s",
		//		which, s.obj, attrs.ToString(s.obj),
		//	)

		// no additional actions to perform
		return ctx, nil
	}
}

// Fake is a no-op method that marks the actor as seen in the context.
//
// # It is used to ensure that the actor is seen in the context
//
// This is useful if a function which might execute an actor
// returns no context - if an actor was already executed in said function
// the actor could be executed again
//
// See [QuerySet.Create] and [QuerySet.Update] for examples of this.
func (s *ObjectActor) Fake(ctx context.Context, flags ...actorFlag) context.Context {
	// no op if the wrapper is nil
	if s == nil {
		return ctx
	}

	// no op if the object is nil
	if s.obj == nil {
		return ctx
	}

	for _, flag := range flags {
		if hasSeenActor(ctx, flag, s.obj) {
			continue
		}
		ctx = markActorSeen(ctx, flag, s.obj)
	}

	return ctx
}

func (s *ObjectActor) BeforeSave(ctx context.Context) (context.Context, error) {
	return s.execute(ctx, actsBeforeSave)
}

func (s *ObjectActor) AfterSave(ctx context.Context) (context.Context, error) {
	return s.execute(ctx, actsAfterSave)
}

func (s *ObjectActor) BeforeCreate(ctx context.Context) (context.Context, error) {
	return s.execute(ctx, actsBeforeCreate)
}

func (s *ObjectActor) AfterCreate(ctx context.Context) (context.Context, error) {
	return s.execute(ctx, actsAfterCreate)
}

func (s *ObjectActor) BeforeUpdate(ctx context.Context) (context.Context, error) {
	return s.execute(ctx, actsBeforeUpdate)
}

func (s *ObjectActor) AfterUpdate(ctx context.Context) (context.Context, error) {
	return s.execute(ctx, actsAfterUpdate)
}

func (s *ObjectActor) BeforeDelete(ctx context.Context) (context.Context, error) {
	return s.execute(ctx, actsBeforeDelete)
}

func (s *ObjectActor) AfterDelete(ctx context.Context) (context.Context, error) {
	return s.execute(ctx, actsAfterDelete)
}

func (s *ObjectActor) AfterQuery(ctx context.Context) (context.Context, error) {
	return s.execute(ctx, actsAfterQuery)
}

type seenActorsContextKey struct {
	which actorFlag
	obj   uintptr
}

func newSeenActorsContextKey(which actorFlag, obj attrs.Definer) seenActorsContextKey {
	var objPtr = reflect.ValueOf(obj).Pointer()
	return seenActorsContextKey{
		which: which,
		obj:   objPtr,
	}
}

func hasSeenActor(ctx context.Context, which actorFlag, obj attrs.Definer) bool {
	if ctx == nil {
		return false
	}
	var key = newSeenActorsContextKey(which, obj)
	var _, seen = ctx.Value(key).(bool)
	return seen
}

func markActorSeen(ctx context.Context, which actorFlag, obj attrs.Definer) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	var key = newSeenActorsContextKey(which, obj)
	return context.WithValue(ctx, key, true)
}
