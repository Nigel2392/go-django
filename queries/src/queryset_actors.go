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
)

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

func runActor(which actorFlag, targetObj attrs.Definer, qs *QuerySet[attrs.Definer]) error {
	switch {
	case which&actsAfterQuery != 0:
		if s, ok := targetObj.(ActsAfterQuery); ok {
			return s.AfterQuery(qs)
		}
	case which&actsBeforeSave != 0:
		if s, ok := targetObj.(ActsBeforeSave); ok {
			return s.BeforeSave(qs)
		}
	case which&actsAfterSave != 0:
		if s, ok := targetObj.(ActsAfterSave); ok {
			return s.AfterSave(qs)
		}
	case which&actsBeforeCreate != 0:
		if err := runActor(actsBeforeSave, targetObj, qs); err != nil {
			return err
		}
		if s, ok := targetObj.(ActsBeforeCreate); ok {
			return s.BeforeCreate(qs)
		}
	case which&actsAfterCreate != 0:
		if err := runActor(actsAfterSave, targetObj, qs); err != nil {
			return err
		}
		if s, ok := targetObj.(ActsAfterCreate); ok {
			return s.AfterCreate(qs)
		}
	case which&actsBeforeUpdate != 0:
		if err := runActor(actsBeforeSave, targetObj, qs); err != nil {
			return err
		}
		if s, ok := targetObj.(ActsBeforeUpdate); ok {
			return s.BeforeUpdate(qs)
		}
	case which&actsAfterUpdate != 0:
		if err := runActor(actsAfterSave, targetObj, qs); err != nil {
			return err
		}
		if s, ok := targetObj.(ActsAfterUpdate); ok {
			return s.AfterUpdate(qs)
		}
	default:
		return fmt.Errorf("unknown actor flag: %d", which)
	}
	return nil
}
