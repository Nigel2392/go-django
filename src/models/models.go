package models

import (
	"context"

	"github.com/Nigel2392/go-django/src/core/attrs"
)

type Saver interface {
	Save(c context.Context) error
}

type Deleter interface {
	Delete(c context.Context) error
}

type Model interface {
	attrs.Definer
	Saver
	Deleter
}
