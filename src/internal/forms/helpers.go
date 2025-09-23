package forms

import (
	"context"
	"reflect"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/core/filesystem"
)

type IsValidDefiner interface {
	IsValid() bool
}

type FormWrapper interface {
	Unwrap() []Form
}

func FullClean(ctx context.Context, f Form) (invalid, defaults, cleaned map[string]any) {
	var rawData, files = f.Data()
	return fullClean(ctx, f, rawData, files)
}

type pointerContextKey struct {
	ptr uintptr
}

func IsValid(ctx context.Context, f Form) bool {

	var rv = reflect.ValueOf(f)
	if rv.Kind() != reflect.Pointer {
		panic("IsValid() only accepts a pointer to a Form, not a value.")
	}

	var topKey = pointerContextKey{ptr: rv.Pointer()}
	var _, hasPtr = ctx.Value(topKey).(struct{})
	if unwrapper, ok := f.(FormWrapper); ok && !hasPtr {
		var isValid = true
		for _, form := range unwrapper.Unwrap() {

			// create a unique key for every form based on its pointer address
			// so we don't get stuck in an infinite loop if the same form is included in the unwrap chain
			var wrappedFormKey = pointerContextKey{
				ptr: reflect.ValueOf(form).Pointer(),
			}

			// make sure every form wrapped still gets cleaned and validated
			// by using the & operator on isValid
			isValid = isValid && IsValid(
				context.WithValue(ctx, wrappedFormKey, struct{}{}),
				form,
			)
		}
		return isValid
	}

	var rawData, files = f.Data()
	assert.False(
		rawData == nil,
		"You cannot call IsValid() without setting the data first.",
	)

	if f.WasCleaned() {
		var errorList = f.ErrorList()
		if len(errorList) > 0 {
			return false
		}

		var boundErrors = f.BoundErrors()
		if boundErrors != nil && boundErrors.Len() > 0 {
			return false
		}

		return f.CleanedDataUnsafe() != nil
	}

	var (
		invalid, defaults, cleaned = fullClean(ctx, f, rawData, files)
		errs                       = f.ErrorList()
		bndErrs                    = f.BoundErrors()
	)

	if bndErrs == nil || bndErrs.Len() == 0 {
		for _, validator := range f.Validators() {
			var errors = validator(f, cleaned)
			if len(errors) > 0 {
				f.AddFormError(errors...)
			}
		}
	}

	errs = f.ErrorList()
	bndErrs = f.BoundErrors()

	if (bndErrs == nil || bndErrs.Len() == 0) && len(errs) == 0 {

		for _, fn := range f.CallbackOnValid() {
			fn(f)
		}
	} else {

		f.BindCleanedData(invalid, defaults, nil)
		for _, fn := range f.CallbackOnInvalid() {
			fn(f)
		}
	}

	for _, fn := range f.CallbackOnFinalize() {
		fn(f)
	}

	if bndErrs != nil && bndErrs.Len() > 0 || len(errs) > 0 {
		f.BindCleanedData(invalid, defaults, nil)
	}

	errs = f.ErrorList()
	bndErrs = f.BoundErrors()
	if (bndErrs == nil || bndErrs.Len() == 0) && len(errs) == 0 {
		if isValidDef, ok := f.(IsValidDefiner); ok {
			return isValidDef.IsValid()
		}
		return true
	}
	return false
}

func fullClean(ctx context.Context, f Form, rawData map[string][]string, files map[string][]filesystem.FileHeader) (invalid_, defaults_, cleaned_ map[string]any) {
	var (
		invalid  = make(map[string]any)
		defaults = make(map[string]any)
		cleaned  = make(map[string]any)
		err      error
	)

	for head := f.FieldMap().Front(); head != nil; head = head.Next() {
		var (
			k       = head.Key
			v       = head.Value
			errors  []error
			initial interface{}
			data    interface{}
		)

		if v.ReadOnly() {
			continue
		}

		var widget, ok = f.Widget(k)
		if !ok {
			widget = v.Widget()
		}

		if !widget.ValueOmittedFromData(ctx, rawData, files, f.PrefixName(k)) {
			initial, errors = widget.ValueFromDataDict(ctx, rawData, files, f.PrefixName(k))
		}

		if len(errors) > 0 {
			f.AddError(k, errors...)
			invalid[k] = initial
			continue
		}

		if v.Required() && v.IsEmpty(initial) {
			f.AddError(k, errs.NewValidationError(k, errs.ErrFieldRequired))
			invalid[k] = initial
			continue
		}

		data, err = v.ValueToGo(initial)
		if err != nil {
			f.AddError(k, err)
			invalid[k] = initial
			continue
		}

		// Set the initial value again in case the value was modified by ValueToGo.
		// This is important so we add the right value to the invalid defaults.
		initial = data

		data, err = v.Clean(ctx, initial)
		if err != nil {
			f.AddError(k, err)
			invalid[k] = initial
			continue
		}

		errors = v.Validate(ctx, data)
		if len(errors) > 0 {
			var errList = make([]error, 0, len(errors))
			for _, err := range errors {
				switch e := err.(type) {
				case interface{ Unwrap() []error }:
					errList = append(errList, e.Unwrap()...)
				default:
					errList = append(errList, err)
				}
			}

			f.AddError(k, errList...)
			invalid[k] = data
			continue
		}

		defaults[k] = data
		cleaned[k] = data
	}

	f.BindCleanedData(invalid, defaults, cleaned)

	return invalid, defaults, cleaned
}
