package queries

import (
	"fmt"

	"github.com/Nigel2392/go-django/src/core/attrs"
)

type actorFlag int

const (
	acts_INVALID   actorFlag = 0
	actsAfterQuery actorFlag = 1 << iota
	actsBeforeSave
	actsAfterSave
	actsBeforeCreate
	actsAfterCreate
	actsBeforeUpdate
	actsAfterUpdate
	actsBeforeDelete
	actsAfterDelete
)

func (a actorFlag) wasSet(flag actorFlag) bool {
	return a&flag != 0
}

type ActsAfterQuery interface {
	AfterQuery(qs *GenericQuerySet) error
}

type ActsBeforeSave interface {
	BeforeSave(qs *GenericQuerySet) error
}

type ActsAfterSave interface {
	AfterSave(qs *GenericQuerySet) error
}

type ActsBeforeCreate interface {
	BeforeCreate(qs *GenericQuerySet) error
}

type ActsAfterCreate interface {
	AfterCreate(qs *GenericQuerySet) error
}

type ActsBeforeUpdate interface {
	BeforeUpdate(qs *GenericQuerySet) error
}

type ActsAfterUpdate interface {
	AfterUpdate(qs *GenericQuerySet) error
}

type ActsBeforeDelete interface {
	BeforeDelete(qs *GenericQuerySet) error
}

type ActsAfterDelete interface {
	AfterDelete(qs *GenericQuerySet) error
}

func runActor(which actorFlag, targetObj attrs.Definer, qs *QuerySet[attrs.Definer]) error {
	switch {
	case which.wasSet(actsAfterQuery):
		if s, ok := targetObj.(ActsAfterQuery); ok {
			return s.AfterQuery(qs)
		}
	case which.wasSet(actsBeforeSave):

		var err = SignalPreModelSave.Send(SignalSave{
			Instance: targetObj,
			Using:    qs.Compiler(),
		})
		if err != nil {
			return fmt.Errorf("error running pre-save signal: %w", err)
		}

		if s, ok := targetObj.(ActsBeforeSave); ok {
			return s.BeforeSave(qs)
		}
	case which.wasSet(actsAfterSave):

		var err = SignalPostModelSave.Send(SignalSave{
			Instance: targetObj,
			Using:    qs.Compiler(),
		})
		if err != nil {
			return fmt.Errorf("error running post-save signal: %w", err)
		}

		if s, ok := targetObj.(ActsAfterSave); ok {
			return s.AfterSave(qs)
		}
	case which.wasSet(actsBeforeCreate):
		if err := runActor(actsBeforeSave, targetObj, qs); err != nil {
			return err
		}
		if s, ok := targetObj.(ActsBeforeCreate); ok {
			return s.BeforeCreate(qs)
		}
	case which.wasSet(actsAfterCreate):
		if err := runActor(actsAfterSave, targetObj, qs); err != nil {
			return err
		}
		if s, ok := targetObj.(ActsAfterCreate); ok {
			return s.AfterCreate(qs)
		}
	case which.wasSet(actsBeforeUpdate):
		if err := runActor(actsBeforeSave, targetObj, qs); err != nil {
			return err
		}
		if s, ok := targetObj.(ActsBeforeUpdate); ok {
			return s.BeforeUpdate(qs)
		}
	case which.wasSet(actsAfterUpdate):
		if err := runActor(actsAfterSave, targetObj, qs); err != nil {
			return err
		}
		if s, ok := targetObj.(ActsAfterUpdate); ok {
			return s.AfterUpdate(qs)
		}
	case which.wasSet(actsBeforeDelete):
		if s, ok := targetObj.(ActsBeforeDelete); ok {
			return s.BeforeDelete(qs)
		}
	case which.wasSet(actsAfterDelete):
		if s, ok := targetObj.(ActsAfterDelete); ok {
			return s.AfterDelete(qs)
		}
	default:
		return fmt.Errorf("unknown actor flag: %d", which)
	}
	return nil
}
