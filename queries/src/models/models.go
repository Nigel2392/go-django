package models

import (
	"context"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/errs"
)

const (
	ErrModelInitialized errs.Error = "model is improperly initialized"
	ErrObjectInvalid    errs.Error = "the object must be a valid pointer to a struct"
	ErrModelEmbedded    errs.Error = "the object must embed the Model struct"
	ErrModelAdressable  errs.Error = "the Model is not addressable"
)

type SaveableObject interface {
	SaveObject(ctx context.Context, cnf SaveConfig) error
}

type DeleteableObject interface {
	DeleteObject(ctx context.Context) error
}

type private struct{}

type _ModelInterface interface {
	__Model() private
}

type CanTargetDefiner interface {
	attrs.Definer
	TargetContentTypeField() attrs.FieldDefinition
	TargetPrimaryField() attrs.FieldDefinition
}

type CanControlSaving interface {
	attrs.Definer
	ControlsEmbedderSaving() bool
}
