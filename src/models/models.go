package models

import (
	"context"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/goldcrest"
)

type ContextSaver interface {
	Save(c context.Context) error
}

type Saver interface {
	Save() error
}

type ContextDeleter interface {
	Delete(c context.Context) error
}

type Deleter interface {
	Delete() error
}

func ImplementsMethods(m attrs.Definer) (definesSave, definesDelete bool) {
	if _, ok := m.(ContextSaver); ok {
		definesSave = true
	}

	if _, ok := m.(Saver); ok {
		definesSave = true
	}

	if _, ok := m.(ContextDeleter); ok {
		definesDelete = true
	}

	if _, ok := m.(Deleter); ok {
		definesDelete = true
	}

	return definesSave, definesDelete
}

type ModelFunc func(c context.Context, m attrs.Definer) (changed bool, err error)

const (
	MODEL_SAVE_HOOK   = "django.Model.Save"
	MODEL_DELETE_HOOK = "django.Model.Delete"
)

func SaveModel(c context.Context, m attrs.Definer) (saved bool, err error) {
	if m == nil {
		return false, nil
	}

	if canSetup, ok := m.(interface{ Setup(obj attrs.Definer) error }); ok {
		if err = canSetup.Setup(m); err != nil {
			return false, err
		}
	}

	if s, ok := m.(ContextSaver); ok {
		return true, s.Save(c)
	}

	if s, ok := m.(Saver); ok {
		return true, s.Save()
	}

	var hooks = goldcrest.Get[ModelFunc](MODEL_SAVE_HOOK)
	for _, hook := range hooks {
		if saved, err = hook(c, m); err != nil || saved {
			return saved, err
		}
	}

	return false, nil
}

func DeleteModel(c context.Context, m attrs.Definer) (deleted bool, err error) {
	if m == nil {
		return false, nil
	}
	if d, ok := m.(ContextDeleter); ok {
		return true, d.Delete(c)
	}

	if d, ok := m.(Deleter); ok {
		return true, d.Delete()
	}

	var hooks = goldcrest.Get[ModelFunc](MODEL_DELETE_HOOK)
	for _, hook := range hooks {
		if deleted, err = hook(c, m); err != nil || deleted {
			return deleted, err
		}
	}

	return false, nil
}
